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

	"github.com/TykTechnologies/gromit/devenv"
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
		log.Fatal().Err(err)
	}
	log.Info().Interface("env", e).Msg("loaded env")
	a.Env = &e

	// Set global cfg
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

	a.Router = mux.NewRouter()
	a.initRoutes()
}

// Run will start GromitServer
func (a *App) Run(addr string, cert string, key string) {
	server := &http.Server{
		Addr:      addr,
		TLSConfig: a.tlsConfig,
	}
	log.Info().Msg("starting server")
	if err := server.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server startup failed")
	}
}

func (a *App) initRoutes() {
	a.Router.HandleFunc("/healthcheck", a.healthCheck).Methods("GET")
	a.Router.HandleFunc("/loglevel/{level}", a.setLoglevel).Methods("PUT")
	a.Router.HandleFunc("/loglevel", a.getLoglevel).Methods("GET")

	//a.Router.HandleFunc("/newbuild", a.newBuild).Methods("POST")

	a.Router.HandleFunc("/envs", a.getEnvs).Methods("GET")
	a.Router.HandleFunc("/env/{name}", a.createEnv).Methods("PUT")
	a.Router.HandleFunc("/env/{name}", a.updateEnv).Methods("PATCH")
	a.Router.HandleFunc("/env/{name}", a.deleteEnv).Methods("DELETE")
	a.Router.HandleFunc("/env/{name}", a.getEnv).Methods("GET")
}

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

func (a *App) getEnvs(w http.ResponseWriter, r *http.Request) {
	return
}

func (a *App) createEnv(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Info().Interface("env", newEnv).Msgf("new env %s created", env)

	respondWithJSON(w, http.StatusCreated, newEnv)
}

func (a *App) updateEnv(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env := vars["name"]

	newEnv := make(devenv.DevEnv)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}
	log.Debug().Interface("env", newEnv).Msgf("update for %s received", env)

	err = devenv.UpsertEnv(a.DB, a.Env.TableName, vars["name"], newEnv)
	if err != nil {
		if ierr, ok := err.(devenv.ExistsError); ok {
			respondWithError(w, http.StatusConflict, ierr.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Info().Interface("env", newEnv).Msgf("%s env updated", env)

	respondWithJSON(w, http.StatusCreated, newEnv)
}

func (a *App) getEnv(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	log.Debug().Interface("vars", vars).Msgf("get for %s received", name)

	env, err := devenv.GetEnv(a.DB, a.Env.TableName, vars["name"], a.Env.Repos)
	if err != nil {
		if ierr, ok := err.(devenv.NotFoundError); ok {
			respondWithError(w, http.StatusNotFound, ierr.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, env)
}

func (a *App) deleteEnv(w http.ResponseWriter, r *http.Request) {
	return
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
