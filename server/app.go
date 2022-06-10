package server

// app definition and handlers
// See model.go to manipulate the environment type

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/TykTechnologies/gromit/config"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// App holds the API clients for the gromit server
type App struct {
	Router     *mux.Router
	awsCfg     *aws.Config
	tlsConfig  *tls.Config
	ECR        ecriface.ClientAPI
	DB         dynamodbiface.ClientAPI
	R53        route53iface.ClientAPI
	Repos      []string
	TableName  string
	RegistryID string
}

//go:embed debug
var debugger embed.FS

// Init loads env vars, AWS, TLS config
// Keep this separate from App.Run() for testing purposes
func (a *App) Init(ca []byte, cert []byte, key []byte) error {
	log.Info().Msg("server init")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load SDK config")
	}
	a.awsCfg = &cfg
	a.ECR = ecr.New(cfg)
	a.R53 = route53.New(cfg)
	a.DB = dynamodb.New(cfg)

	err = devenv.EnsureTableExists(a.DB, a.TableName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not ensure table %s exists", a.TableName)
	}
	log.Info().Str("table", a.TableName).Msg("Found")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)
	scert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("could not load server key pair: %w", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{scert},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}
	if tlsConfig.NextProtos == nil {
		tlsConfig.NextProtos = []string{"http/1.1"}
	}
	tlsConfig.BuildNameToCertificate()
	a.tlsConfig = tlsConfig
	a.initRoutes()
	return nil
}

func (a *App) Run(addr string) error {
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tlsListener := tls.NewListener(conn, a.tlsConfig)
	log.Info().Msg("starting gromit server")
	server := &http.Server{
		Addr:      addr,
		Handler:   a.Router,
		TLSConfig: a.tlsConfig,
	}
	return server.Serve(tlsListener)
}

// Test returns a local server suitable for testing, remember to close it
func (a *App) Test() *httptest.Server {
	log.Info().Msg("starting test server")
	server := httptest.NewUnstartedServer(a.Router)
	server.TLS = a.tlsConfig
	// server.Config.Handler = a.Router
	server.StartTLS()
	return server
}

// StartTestServer starts an HttpTest server that is suitable for testing
func StartTestServer(confFile string) (*httptest.Server, *App) {
	config.LoadConfig(confFile)
	a := App{
		TableName:  config.TableName,
		RegistryID: config.RegistryID,
		Repos:      config.Repos,
	}
	err := a.Init(
		[]byte(viper.GetString("ca")),
		[]byte(viper.GetString("serve.cert")),
		[]byte(viper.GetString("serve.key")),
	)
	if err != nil {
		fmt.Println("could not init test app", err)
		os.Exit(1)
	}

	ts := a.Test()
	os.Setenv("GROMIT_SERVE_URL", ts.URL)
	a.initRoutes()
	return ts, &a
}

func (a *App) initRoutes() {
	a.Router = mux.NewRouter()

	a.Router.HandleFunc("/healthcheck", a.healthCheck).Methods("GET")
	a.Router.HandleFunc("/loglevel/{level}", a.setLoglevel).Methods("PUT")
	a.Router.HandleFunc("/loglevel", a.getLoglevel).Methods("GET")
	a.Router.PathPrefix("/debug").Handler(http.FileServer(http.FS(debugger)))

	// Endpoint for int-image GHA
	a.Router.HandleFunc("/newbuild", a.newBuild).Methods("POST")

	// ReST API
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

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
