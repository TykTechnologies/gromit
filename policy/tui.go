package policy

import (
	"embed"
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
	s := CreateNewServer(tvFile)
	log.Info().Msg("starting tui server")
	return http.ListenAndServe(port, s.Router)
}

// ping api Handler
func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pong!"))
}

type Server struct {
	Router *chi.Mux
	// Db, config can be added here
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

func CreateNewServer(tvFile string) *Server {
	r := chi.NewRouter()
	// Order is important
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httprate.LimitAll(100, 1*time.Minute))

	// Standard handlers
	r.Mount("/pprof", middleware.Profiler())
	r.Get("/ping", ping)
	r.Mount("/static/", assets())

	_, err := loadVariations(tvFile, r)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not load test variation from %s", tvFile)
	}

	return &Server{
		Router: r,
	}
}

// TestsuiteVariations maps savedVariations to a form suitable for runtime use
type TestsuiteVariations struct {
	Repos           map[string]repoVariations
	SavedData       ghMatrix
	SaveDir         string
	SavedVariations []string
}

// loadVariations unrolls the compact saved representation from a file
// it also sets up handlers for the loaded variations
func loadVariations(tvFile string, router *chi.Mux) (*TestsuiteVariations, error) {
	data, err := os.ReadFile(tvFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", tvFile, err)
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
	rvs := make(map[string]repoVariations)
	for repo, matrix := range saved.Level {
		var rv repoVariations
		var vp variationPath
		rv.Leaves = make(map[string]ghMatrix)
		// apply defaults to every repo
		matrix.EnvFiles = append(matrix.EnvFiles, global.EnvFiles...)
		matrix.Pump = append(matrix.Pump, global.Pump...)
		matrix.Sink = append(matrix.Sink, global.Sink...)

		parseVariations(matrix, 0, &rv, vp)
		rvs[repo] = rv
	}
	saveDir := filepath.Dir(tvFile)
	if saveDir == "" {
		saveDir = "."
	}
	savedFiles, err := findVariations(saveDir)
	if err != nil {
		return nil, fmt.Errorf("cannot find saved variations in %s: %w", saveDir, err)
	}
	tv := TestsuiteVariations{
		Repos:           rvs,
		SavedData:       saved,
		SaveDir:         saveDir,
		SavedVariations: savedFiles,
	}
	log.Debug().Interface("tv", tv).Msgf("loaded from %s", tvFile)

	router.Route("/api", func(r chi.Router) {
		r.Get("/", tv.DumpJson)
		r.Get("/{repo}/{branch}/{trigger}/{ts}/{field}", tv.Lookup)
	})
	router.Get("/", tv.renderSPA())
	router.Mount("/show/", savedVariations(saveDir))
	router.Put("/save/{name}", tv.saveVariation)
	router.Post("/use/{name}", func(w http.ResponseWriter, r *http.Request) {
		tvFile := chi.URLParam(r, "name")
		tvFile = filepath.Join(tv.SaveDir, tvFile)
		_, err := loadVariations(tvFile, router)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error().Err(err).Msgf("cannot use %s", tvFile)
		}
	})
	return &tv, nil
}

// findVariations returns a list of files in the directory
func findVariations(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return []string{}, err
	}
	fnames := make([]string, len(files))
	for i, f := range files {
		fnames[i] = f.Name()
	}
	return fnames, nil
}

// savedVariations serves all the saved test suite variations from the supplied dir
func savedVariations(dir string) http.Handler {
	fs := os.DirFS(dir)
	return http.StripPrefix("/show/", http.FileServer(http.FS(fs)))
}

// API handlers

func (tv TestsuiteVariations) saveVariation(w http.ResponseWriter, r *http.Request) {
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
	name = filepath.Join(tv.SaveDir, name)
	err = os.WriteFile(name, body, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("writing test variations to %s", name)
		return
	}

	savedFiles, err := findVariations(tv.SaveDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msgf("could not re-read saved variations from %s", tv.SaveDir)
		return
	}
	tv.SavedVariations = savedFiles
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s saved successfully", name)))
}

func (tv TestsuiteVariations) Lookup(w http.ResponseWriter, r *http.Request) {
	repo := chi.URLParam(r, "repo")
	branch := chi.URLParam(r, "branch")
	trigger := chi.URLParam(r, "trigger")
	testsuite := chi.URLParam(r, "ts")
	field := chi.URLParam(r, "field")

	var m *ghMatrix
	repoVariations, found := tv.Repos[repo]
	if !found {
		http.Error(w, fmt.Sprintf("%s not known", repo), http.StatusNotFound)
	}
	m = repoVariations.Lookup(branch, trigger, testsuite)
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

func (tv TestsuiteVariations) DumpJson(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, tv)
	return
}

func (tv TestsuiteVariations) String() string {
	y, err := yaml.Marshal(tv.SavedData)
	if err != nil {
		log.Err(err).Msgf("cannot marshal %v", tv.SavedData)
	}
	return string(y)
}

//go:embed app/*
var templateFS embed.FS

// HTML endpoints
func (tv TestsuiteVariations) renderSPA() http.HandlerFunc {
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
		err := t.Execute(w, tv)
		if err != nil {
			log.Error().Err(err).Msg("rendering template index.html")
		}
	}
}
