package policy

import (
	"bytes"
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
	"github.com/rs/zerolog/log"
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
	ProdVariations variations
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
		r.Get("/{tsv}/{repo}/{branch}/{trigger}/{ts}/{field}.json", s.renderObj("json"))
		r.Get("/{tsv}/{repo}/{branch}/{trigger}/{ts}/{field}.gho", s.renderObj("gho"))
		r.Get("/{tsv}/{repo}/{branch}/{trigger}/{ts}.gho", s.renderObj("gho"))
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

// finds the supplied name as a sub-string among the keys of AllVariations
func (s *Server) findVariation(name string) (variations, bool) {
	found := false
	re := regexp.MustCompile(name)
	for k, v := range s.AllVariations {
		found = re.MatchString(k)
		if found {
			return v, found
		}
	}
	return variations{}, found
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
	m = s.ProdVariations.Lookup(repo, branch, trigger, testsuite)
	if m == nil {
		// if branch not known, send down master's config
		m = s.ProdVariations.Lookup(repo, "master", trigger, testsuite)
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

func (s *Server) renderObj(format string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tsv := chi.URLParam(r, "tsv")
		repo := chi.URLParam(r, "repo")
		branch := chi.URLParam(r, "branch")
		trigger := chi.URLParam(r, "trigger")
		testsuite := chi.URLParam(r, "ts")
		field := chi.URLParam(r, "field")

		m, err := s.findMatrix(tsv, repo, branch, trigger, testsuite)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		var obj any
		if len(field) > 0 {
			v := reflect.ValueOf(m)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			f := v.FieldByName(field)
			if !f.IsValid() {
				http.Error(w, fmt.Sprintf("%s(%s, %s, %s) has no field %s", repo, branch, trigger, testsuite, field), http.StatusNotFound)
				return
			}
			obj = f.Interface()
		} else {
			obj = *m
		}
		switch format {
		case "json":
			render.JSON(w, r, obj)
		case "gho":
			renderGHO(w, obj)
		}
	}
}

// renderGHO writes the supplied object in a form that github actions can parse it as a variable
func renderGHO(w http.ResponseWriter, obj any) {
	val := reflect.ValueOf(obj)
	typ := val.Type()

	var buf bytes.Buffer
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		fieldName := field.Tag.Get("json")
		fjson, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(fieldName) > 0 {
			buf.WriteString(fieldName + "<<EOF\n")
			buf.Write(fjson)
			buf.WriteString("\nEOF\n")
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

// findMatrix looks up RepoTestsuiteVariations based on the tsv parameter
// being a regexp match for one of the map keys of s.AllVariations
func (s *Server) findMatrix(tsv, repo, branch, trigger, testsuite string) (*ghMatrix, error) {
	log.Debug().Msgf("looking for %s in %v", tsv, s.AllVariations.Files())
	v, found := s.findVariation(tsv)
	if !found {
		return nil, fmt.Errorf("%s not found among %v", tsv, s.AllVariations.Files())
	}
	m := v.Lookup(repo, branch, trigger, testsuite)
	if m == nil {
		return nil, fmt.Errorf("(%s or master, %s, %s) not known for %s", branch, trigger, testsuite, repo)
	}
	log.Debug().Interface("matrix", m).Msgf("found in %s", tsv)
	return m, nil
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
