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
	"regexp"
	"strings"
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
func Serve(port, tvDir string) error {
	creds := getCredentials(os.Getenv("CREDENTIALS"))
	s := CreateNewServer(tvDir, creds)
	log.Info().Msg("starting tui server")
	return http.ListenAndServe(port, s.Router)
}

type Server struct {
	Router         *chi.Mux
	ProdVariations RepoTestsuiteVariations
	SaveDir        string
	AllVariations  AllTestsuiteVariations
	// Db, config can be added here
}

func CreateNewServer(tvDir string, creds map[string]string) *Server {
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

	av, err := loadAllVariations(tvDir)
	if err != nil {
		log.Fatal().Err(err).Msgf("loading variatios from %s", tvDir)
	}
	s := &Server{
		Router:         r,
		SaveDir:        tvDir,
		ProdVariations: av["prod-variations.yml"],
		AllVariations:  av,
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/{repo}/{branch}/{trigger}/{ts}/{field}", s.lookup)
	})
	r.Route("/v2", func(r chi.Router) {
		r.Get("/dump/{tsv}", s.dumpJson)
		r.Get("/{tsv}/{repo}/{branch}/{trigger}/{ts}/{field}", s.lookup2)
	})
	r.Get("/", s.renderSPA())
	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("tui", creds))
		r.Put("/save/{name}", s.saveVariation)
		r.Post("/reload", s.reload)
	})
	r.Mount("/show/", savedVariations(tvDir))
	return s
}

// RepoTestsuiteVariations maps savedVariations to a form suitable for runtime use
type RepoTestsuiteVariations map[string]repoVariations
type AllTestsuiteVariations map[string]RepoTestsuiteVariations

func (av AllTestsuiteVariations) Files() []string {
	keys := make([]string, 0, len(av))
	for k := range av {
		keys = append(keys, k)
	}
	return keys
}

// finds the supplied name as a sub-string among the keys of AllVariations
func (s *Server) findVariation(name string) (RepoTestsuiteVariations, bool) {
	found := false
	re := regexp.MustCompile(name)
	for k, v := range s.AllVariations {
		found = re.MatchString(k)
		if found {
			return v, found
		}
	}
	return nil, found
}

// API handlers

func (s *Server) reload(w http.ResponseWriter, r *http.Request) {
	av, err := loadAllVariations(s.SaveDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("cannot reload, leaving current config unchanged")
	}
	s.AllVariations = av
	s.ProdVariations = av["prod-variations.yml"]
	w.Write([]byte(fmt.Sprintf("Using [%s] now", strings.Join(s.AllVariations.Files(), ", "))))
}

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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s saved successfully", name)))
}

// FIXME: Remove when lookup2 migration is done
func (s *Server) lookup(w http.ResponseWriter, r *http.Request) {
	repo := chi.URLParam(r, "repo")
	branch := chi.URLParam(r, "branch")
	trigger := chi.URLParam(r, "trigger")
	testsuite := chi.URLParam(r, "ts")
	field := chi.URLParam(r, "field")

	var m *ghMatrix
	rv, found := s.ProdVariations[repo]
	if !found {
		http.Error(w, fmt.Sprintf("%s not known", repo), http.StatusNotFound)
		return
	}
	m = rv.Lookup(branch, trigger, testsuite)
	if m == nil {
		// if branch not known, send down master's config
		m = rv.Lookup("master", trigger, testsuite)
		if m == nil {
			http.Error(w, fmt.Sprintf("(master, %s, %s) not known for %s", trigger, testsuite, repo), http.StatusNotFound)
			return
		}
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

// lookup2 looks up RepoTestsuiteVariations based on the tsv parameter
// being a regexp match for one of the map keys of s.AllVariations
func (s *Server) lookup2(w http.ResponseWriter, r *http.Request) {
	tsv := chi.URLParam(r, "tsv")
	repo := chi.URLParam(r, "repo")
	branch := chi.URLParam(r, "branch")
	trigger := chi.URLParam(r, "trigger")
	testsuite := chi.URLParam(r, "ts")
	field := chi.URLParam(r, "field")

	log.Debug().Msgf("looking for %s in %v", tsv, s.AllVariations.Files())
	rtsv, found := s.findVariation(tsv)
	if !found {
		http.Error(w, fmt.Sprintf("%s not found among %v", tsv, s.AllVariations.Files()), http.StatusNotFound)
		return
	}
	var m *ghMatrix
	rv, found := rtsv[repo]
	if !found {
		http.Error(w, fmt.Sprintf("%s not known", repo), http.StatusNotFound)
		return
	}
	m = rv.Lookup(branch, trigger, testsuite)
	if m == nil {
		// if branch not known, send down master's config
		m = rv.Lookup("master", trigger, testsuite)
		if m == nil {
			http.Error(w, fmt.Sprintf("default (master, %s, %s) not known for %s", trigger, testsuite, repo), http.StatusNotFound)
			return
		}
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
	tsv := chi.URLParam(r, "tsv")
	rtsv, found := s.findVariation(tsv)
	if !found {
		http.Error(w, fmt.Sprintf("%s not found among %v", tsv, s.AllVariations.Files()), http.StatusNotFound)
		return
	}
	render.JSON(w, r, rtsv)
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

// loadAllVariations loads all yaml files in tvDir returning it in a
// map indexed by filename
func loadAllVariations(tvDir string) (AllTestsuiteVariations, error) {
	files, err := os.ReadDir(tvDir)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read directory")
	}

	numVariations := 0
	av := make(AllTestsuiteVariations)
	for _, file := range files {
		fname := file.Name()
		yaml, _ := regexp.MatchString("\\.ya?ml$", fname)
		if !yaml {
			continue
		}
		pathName := filepath.Join(tvDir, fname)
		tv, err := loadVariation(pathName)
		if err != nil {
			log.Warn().Err(err).Msgf("could not load test variation from %s", pathName)
		}
		av[fname] = tv
		numVariations++
	}
	if numVariations < 1 {
		return av, fmt.Errorf("No loadable files in %s", tvDir)
	}
	return av, nil
}

// loadVariation unrolls the compact saved representation from a file
// it also sets up handlers for the loaded variations
func loadVariation(tvFile string) (RepoTestsuiteVariations, error) {
	data, err := os.ReadFile(tvFile)
	if err != nil {
		return nil, err
	}
	var saved ghMatrix
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal data from %s: %s: %w", tvFile, string(data), err)
	}
	// top level variations
	var global ghMatrix
	err = copier.CopyWithOption(&global, &saved, copier.Option{IgnoreEmpty: true})
	if err != nil {
		log.Warn().Err(err).Msgf("could not copy global variations")
	}
	tv := make(RepoTestsuiteVariations)
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
	return tv, nil
}
