package policy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-chi/render"
	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Serve starts the embedded test controller UI
func Serve(port, tvFile string) error {
	creds := getCredentials(os.Getenv("CREDENTIALS"))
	s := CreateNewServer(tvFile, creds)
	log.Info().Msg("starting tui server")
	return http.ListenAndServe(port, s.Router)
}

type Server struct {
	Router          *chi.Mux
	Variations      TestsuiteVariations
	SaveDir         string
	SavedVariations []string
	// Db, config can be added here
}

func CreateNewServer(tvFile string, creds map[string]string) *Server {
	r := chi.NewRouter()
	// Order is important
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httprate.LimitAll(100, 1*time.Minute))

	r.Mount("/pprof", middleware.Profiler())
	r.Get("/ping", ping)
	r.Mount("/static/", assets())

	saveDir := filepath.Dir(tvFile)
	if saveDir == "" {
		saveDir = "."
	}

	s := &Server{
		Router:  r,
		SaveDir: saveDir,
	}
	err := loadVariations(tvFile, s)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not load test variation from %s", tvFile)
	}
	r.Route("/api", func(r chi.Router) {
		r.Get("/", s.dumpJson)
		r.Get("/{repo}/{branch}/{trigger}/{ts}/{field}", s.lookup)
	})
	r.Get("/", s.renderSPA())
	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("tui", creds))
		r.Put("/save/{name}", s.saveVariation)
		r.Post("/use/{name}", s.useVariation)
	})
	r.Mount("/show/", savedVariations(s.SaveDir))
	return s
}

// TestsuiteVariations maps savedVariations to a form suitable for runtime use
type TestsuiteVariations map[string]repoVariations

// loadVariations unrolls the compact saved representation from a file
// it also sets up handlers for the loaded variations
func loadVariations(tvFile string, s *Server) error {
	data, err := os.ReadFile(tvFile)
	if err != nil {
		return err
	}
	var saved ghMatrix
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return fmt.Errorf("could not unmarshal data from %s: %s: %w", tvFile, string(data), err)
	}
	// top level variations
	var global ghMatrix
	err = copier.CopyWithOption(&global, &saved, copier.Option{IgnoreEmpty: true})
	if err != nil {
		log.Warn().Err(err).Msgf("could not copy global variations")
	}
	tv := make(TestsuiteVariations)
	for repo, matrix := range saved.Level {
		var rv repoVariations
		var vp variationPath
		rv.Leaves = make(map[string]ghMatrix)
		// apply defaults to every repo
		matrix.EnvFiles = append(matrix.EnvFiles, global.EnvFiles...)
		matrix.Pump = append(matrix.Pump, global.Pump...)
		matrix.Sink = append(matrix.Sink, global.Sink...)

		parseVariations(matrix, 0, &rv, vp)
		tv[repo] = rv
	}
	log.Debug().Interface("tv", tv).Msgf("loaded from %s", tvFile)

	s.Variations = tv
	savedFiles, err := s.findVariations()
	if err != nil {
		return fmt.Errorf("cannot find saved variations in %s: %w", s.SaveDir, err)
	}
	s.SavedVariations = savedFiles

	return nil
}

func (s *Server) useVariation(w http.ResponseWriter, r *http.Request) {
	tvFile := chi.URLParam(r, "name")
	tvFile = filepath.Join(s.SaveDir, tvFile)
	err := loadVariations(tvFile, s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("cannot use %s", tvFile)
	}
	w.Write([]byte(fmt.Sprintf("Using %s now", tvFile)))
}

// findVariations returns a list of files in the directory
func (s *Server) findVariations() ([]string, error) {
	files, err := os.ReadDir(s.SaveDir)
	if err != nil {
		return []string{}, err
	}
	fnames := make([]string, len(files))
	for i, f := range files {
		fnames[i] = f.Name()
	}
	return fnames, nil
}

// API handlers

func (s *Server) saveVariation(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("reading save body")
		return
	}
	defer r.Body.Close()

	name := chi.URLParam(r, "name")
	if name == "" {
		name = "new-test-variations.yml"
	}
	name = filepath.Join(s.SaveDir, name)
	err = os.WriteFile(name, body, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("writing test variations to %s", name)
		return
	}

	savedFiles, err := s.findVariations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("could not re-read saved variations from %s", s.SaveDir)
		return
	}
	s.SavedVariations = savedFiles
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s saved successfully", name)))
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request) {
	repo := chi.URLParam(r, "repo")
	branch := chi.URLParam(r, "branch")
	trigger := chi.URLParam(r, "trigger")
	testsuite := chi.URLParam(r, "ts")
	field := chi.URLParam(r, "field")

	var m *ghMatrix
	rv, found := s.Variations[repo]
	if !found {
		http.Error(w, fmt.Sprintf("%s not known", repo), http.StatusNotFound)
	}
	m = rv.Lookup(branch, trigger, testsuite)
	if m == nil {
		http.Error(w, fmt.Sprintf("(%s, %s, %s) not known for %s", branch, trigger, testsuite, repo), http.StatusNotFound)
		return
	}
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName(field)
	if !f.IsValid() {
		http.Error(w, fmt.Sprintf("%s(%s, %s, %s) has no field %s", repo, branch, trigger, testsuite, field), http.StatusNotFound)
		return
	}
	render.JSON(w, r, f.Interface())
}

func (s *Server) dumpJson(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, s.Variations)
	return
}

//go:embed app/*
var templateFS embed.FS

// HTML endpoints
func (s *Server) renderSPA() http.HandlerFunc {
	tFS, err := fs.Sub(templateFS, "app")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create template FS for app/")
	}
	t, err := template.New("index.html").ParseFS(tFS, "index.html")
	if err != nil {
		log.Error().Err(err).Msg("loading template index.html")
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, s)
		if err != nil {
			log.Error().Err(err).Msg("rendering template index.html")
		}
	}
}

// static assets that are served as is
//
//go:embed app/static/*
var staticFS embed.FS

func assets() http.Handler {
	f, err := fs.Sub(staticFS, "app/static")
	if err != nil {
		log.Fatal().Err(err).Msg("serving tui spa")
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(f)))
}

// savedVariations serves all the saved test suite variations from the supplied dir
func savedVariations(dir string) http.Handler {
	fs := os.DirFS(dir)
	return http.StripPrefix("/show/", http.FileServer(http.FS(fs)))
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pong!"))
}

// getCredentials parses the JSON into a form that the basic auth middleware can use
func getCredentials(jsontext string) map[string]string {
	creds := make(map[string]string)
	err := json.Unmarshal([]byte(jsontext), &creds)
	if err != nil {
		log.Fatal().Err(err).Msg("getting creds for authenticated APIs")
	}
	return creds
}
