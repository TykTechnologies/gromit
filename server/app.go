package server

// app definition and handlers
// See model.go to manipulate the environment type

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/route53iface"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// EnvConfig loaded by kelseyhightower/envconfig
type EnvConfig struct {
	Repos      []string
	TableName  string
	RegistryID string
	ZoneID     string
	Domain     string
}

// App holds the API clients for the gromit server
type App struct {
	Router    *mux.Router
	awsCfg    *aws.Config
	Env       *EnvConfig
	tlsConfig *tls.Config
	ECR       ecriface.ClientAPI
	DB        dynamodbiface.ClientAPI
	R53       route53iface.ClientAPI
}

// Init loads env vars, AWS, TLS config
// Keep this separate from App.Run() for testing purposes
func (a *App) Init(ca string) {
	var e EnvConfig
	// Read env vars prefixed by GROMIT_
	err := envconfig.Process("gromit", &e)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load env")
	}
	log.Info().Interface("env", e).Msg("loaded env for gserve")
	a.Env = &e

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load SDK config")
	}
	a.awsCfg = &cfg
	a.ECR = ecr.New(cfg)
	a.R53 = route53.New(cfg)
	a.DB = dynamodb.New(cfg)

	err = devenv.EnsureTableExists(a.DB, a.Env.TableName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not ensure table %s exists", a.Env.TableName)
	}
	log.Info().Str("table", a.Env.TableName).Msg("Found")

	caCert, err := ioutil.ReadFile(ca)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not find CA certificate")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	a.tlsConfig = tlsConfig
	log.Debug().Msgf("CA cert loaded from %s", ca)

	a.initRoutes()
}

// Run will start GromitServer
func (a *App) Run(addr string, cert string, key string) {
	server := &http.Server{
		Addr:      addr,
		Handler:   a.Router,
		TLSConfig: a.tlsConfig,
	}
	log.Info().Msg("starting server")
	if err := server.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server startup failed")
	}
}

// Test returns a local server suitable for testing, remember to close it
func (a *App) Test(cert string, key string) *httptest.Server {
	log.Info().Msg("starting test server")
	server := httptest.NewUnstartedServer(nil)
	server.TLS = a.tlsConfig
	server.Config.Handler = a.Router
	server.Start()
	return server
}

func (a *App) initRoutes() {
	a.Router = mux.NewRouter()

	a.Router.HandleFunc("/healthcheck", a.healthCheck).Methods("GET")
	a.Router.HandleFunc("/loglevel/{level}", a.setLoglevel).Methods("PUT")
	a.Router.HandleFunc("/loglevel", a.getLoglevel).Methods("GET")

	// Endpoint for int-image GHA
	a.Router.HandleFunc("/newbuild", a.newBuild).Methods("POST")

	// ReST API
	a.Router.HandleFunc("/envs", a.getEnvs).Methods("GET")
	a.Router.HandleFunc("/env/{name}", a.createEnv).Methods("PUT")
	a.Router.HandleFunc("/env/{name}", a.updateEnv).Methods("PATCH")
	a.Router.HandleFunc("/env/{name}", a.deleteEnv).Methods("DELETE")
	a.Router.HandleFunc("/env/{name}", a.getEnv).Methods("GET")
}

// Infra routes

func (a *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
	log.Debug().Msg("Healthcheck")
}

func (a *App) setLoglevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	level, err := zerolog.ParseLevel(vars["level"])

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	zerolog.SetGlobalLevel(level)
	a.getLoglevel(w, r)
}

func (a *App) getLoglevel(w http.ResponseWriter, r *http.Request) {
	loglevel := make(map[string]string)
	loglevel["level"] = zerolog.GlobalLevel().String()
	log.Debug().Msgf("loglevel is %s", loglevel["level"])

	respondWithJSON(w, http.StatusOK, loglevel)
}

func getTrailingElement(string string, separator string) string {
	urlDecoded, err := url.QueryUnescape(string)
	if err != nil {
		log.Debug().Err(err).Msgf("could not decode %s, proceeding anyway", string)
	}
	stringArray := strings.Split(urlDecoded, separator)
	return stringArray[len(stringArray)-1]
}

// This is the handler that is invoked from github

func (a *App) newBuild(w http.ResponseWriter, r *http.Request) {
	util.StatCount("newbuild.count", 1)
	newBuild := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&newBuild)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Trace().Interface("newBuild", newBuild).Msg("parsed from github")

	// Github sends org/reponame
	repo := getTrailingElement(newBuild["repo"], "/")
	// Github sends a path like refs/.../heads/<ref that we want>
	// Also remove all . as it will cause a problem with DNS
	ref := strings.Replace(getTrailingElement(newBuild["ref"], "/"), ".", "", -1)
	sha := newBuild["sha"]

	log.Debug().Str("repo", repo).Str("ref", ref).Str("sha", sha).Msg("to be inserted")

	de, err := devenv.GetDevEnv(a.DB, a.Env.TableName, ref)
	if err != nil {
		if derr, ok := err.(devenv.NotFoundError); ok {
			log.Info().Str("env", ref).Msg("not found, creating")
			de = devenv.NewDevEnv(ref, a.DB, a.Env.TableName)
		} else {
			util.StatCount("newbuild.failures", 1)
			log.Error().Err(derr).Str("env", ref).Msg("could not lookup env")
			respondWithError(w, http.StatusInternalServerError, "could not lookup env "+ref)
			return
		}
	}
	de.MarkNew()
	vs, err := devenv.GetECRState(a.ECR, a.Env.RegistryID, ref, a.Env.Repos)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		log.Error().Err(err).Str("env", ref).Msg("could not find ecr state")
		respondWithError(w, http.StatusInternalServerError, "could not find ecr state "+ref)
		return
	}
	de.SetVersions(vs)
	de.SetVersion(repo, sha)
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not mark as new: "+ref)
	}
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}

// ReST API for /env

// TODO: Implement listing of all environments
func (a *App) getEnvs(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusNotImplemented, "Not implemented")
}

func (a *App) createEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.create.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]

	newEnv := make(devenv.VersionMap)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	log.Debug().Str("envname", ref).Interface("payload", newEnv).Msg("new env received")
	de := devenv.NewDevEnv(ref, a.DB, a.Env.TableName)
	if err != nil {
		util.StatCount("env.create.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	de.SetVersions(newEnv)
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not create: "+ref)
	}
	log.Info().Interface("env", de.VersionMap()).Msg("created")
	respondWithJSON(w, http.StatusCreated, de.VersionMap())
}

func (a *App) updateEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.update.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Debug().Interface("env", ref).Interface("payload", vars).Msgf("update received")

	newEnv := make(devenv.VersionMap)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	de, err := devenv.GetDevEnv(a.DB, a.Env.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	de.MarkNew()
	err = de.MergeVersions(newEnv)
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not update: "+ref)
		return
	}
	log.Info().Interface("env", ref).Msg("env updated")
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}

func (a *App) getEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.get.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Trace().Interface("env", ref).Interface("payload", vars).Msgf("get env received")

	de, err := devenv.GetDevEnv(a.DB, a.Env.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	log.Debug().Interface("env", ref).Msg("env found")
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}

func (a *App) deleteEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.delete.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Debug().Interface("env", ref).Interface("payload", vars).Msgf("new env received")

	de, err := devenv.GetDevEnv(a.DB, a.Env.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	de.MarkDeleted()
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not delete: "+ref)
		return
	}
	log.Info().Interface("env", ref).Msg("marked as deleted")
	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "ok")
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
