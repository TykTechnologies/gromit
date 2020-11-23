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
	log.Info().Msgf("Found table %s", a.Env.TableName)

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
	// Github sends a path like refs/.../integration/<ref that we want>
	// Removing . from the ref as it will be used as the cluster name and in DNS
	ref := strings.ReplaceAll(getTrailingElement(newBuild["ref"], "/"), ".", "")
	sha := newBuild["sha"]

	log.Debug().Str("repo", repo).Str("ref", ref).Str("sha", sha).Msg("to be inserted")

	ecrState, err := devenv.GetECRState(a.ECR, a.Env.RegistryID, ref, a.Env.Repos)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		log.Warn().
			Err(err).
			Msgf("could not get ecr state for %s using registry %s with repo list %v", ref, a.Env.RegistryID, a.Env.Repos)
		respondWithError(w, http.StatusInternalServerError, "could got retrieve ecr state")
		return
	}
	log.Trace().Interface("ecrState", ecrState).Msgf("for ref %s", ref)
	ecrState[repo] = sha
	log.Trace().Interface("ecrState", ecrState).Msgf("for ref %s after update", ref)

	// Set state so that the runner will pick this up
	ecrState[devenv.STATE] = devenv.NEW
	err = devenv.UpsertEnv(a.DB, a.Env.TableName, ref, ecrState)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		log.Warn().
			Interface("ecrState", ecrState).
			Err(err).
			Msgf("could not add new build for %s", ref)
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}
	respondWithJSON(w, http.StatusOK, ecrState)
}

// ReST API for /env

// TODO Implement listing of all environments
func (a *App) getEnvs(w http.ResponseWriter, r *http.Request) {
	return
}

func (a *App) createEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.create.count", 1)
	vars := mux.Vars(r)
	env := vars["name"]

	newEnv := make(devenv.DevEnv)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}
	log.Debug().Interface("env", newEnv).Msgf("new env %s received", env)

	err = devenv.InsertEnv(a.DB, a.Env.TableName, vars["name"], newEnv)
	if err != nil {
		if ierr, ok := err.(devenv.ExistsError); ok {
			respondWithError(w, http.StatusConflict, ierr.Error())
			return
		}
		util.StatCount("env.create.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Info().Interface("env", newEnv).Msgf("new env %s created", env)

	respondWithJSON(w, http.StatusCreated, newEnv)
}

func (a *App) updateEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.update.count", 1)
	vars := mux.Vars(r)
	env := vars["name"]

	newEnv := make(devenv.DevEnv)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}
	log.Debug().Interface("env", newEnv).Msgf("update for %s received", env)

	newEnv[devenv.STATE] = devenv.NEW
	err = devenv.UpsertEnv(a.DB, a.Env.TableName, env, newEnv)
	if err != nil {
		if ierr, ok := err.(devenv.ExistsError); ok {
			respondWithError(w, http.StatusConflict, ierr.Error())
			return
		}
		util.StatCount("env.update.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Info().Interface("env", newEnv).Msgf("%s env updated", env)

	respondWithJSON(w, http.StatusOK, newEnv)
}

func (a *App) getEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.get.count", 1)
	vars := mux.Vars(r)
	name := vars["name"]
	log.Debug().Interface("vars", vars).Msgf("get for %s received", name)

	env, err := devenv.GetEnv(a.DB, a.Env.TableName, vars["name"])
	if err != nil {
		if ierr, ok := err.(devenv.NotFoundError); ok {
			respondWithError(w, http.StatusNotFound, ierr.Error())
			return
		}
		util.StatCount("env.get.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, env)
}

func (a *App) deleteEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.delete.count", 1)
	vars := mux.Vars(r)
	name := vars["name"]
	log.Debug().Interface("vars", vars).Msgf("delete for %s received", name)

	err := devenv.DeleteEnv(a.DB, a.Env.TableName, name)
	if err != nil {
		util.StatCount("env.delete.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Info().Msgf("env %s deleted", name)
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
