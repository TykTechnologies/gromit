package policy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
)

// Serve starts the embedded test controller UI
func Serve(port, tvFile string) error {
	s := CreateNewServer(tvFile)
	s.MountHandlers()
	return http.ListenAndServe(":3000", s.Router)
}

// HelloWorld api Handler
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

type Server struct {
	Router         *chi.Mux
	TestVariations TestsuiteVariations
	// Db, config can be added here
}

func CreateNewServer(tvFile string) *Server {
	tv, err := loadVariations(tvFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not load test variation from %s", tvFile)
	}
	log.Debug().Interface("tv", tv).Msgf("loaded from %s", tvFile)

	s := &Server{
		TestVariations: tv,
	}
	s.Router = chi.NewRouter()
	return s
}

func (s *Server) MountHandlers() {
	// Order is important
	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.CleanPath)
	s.Router.Use(middleware.Timeout(60 * time.Second))
	s.Router.Use(httprate.LimitAll(100, 1*time.Minute))

	s.Router.Mount("/pprof", middleware.Profiler())

	s.Router.Route("/variations", func(r chi.Router) {
		r.Get("/", s.TestVariations.DumpJson)
		r.Get("/{repo}/{branch}/{trigger}/{ts}", s.TestVariations.Lookup)
	})
}

// API handlers
func (tv TestsuiteVariations) Lookup(w http.ResponseWriter, r *http.Request) {
	repo := chi.URLParam(r, "repo")
	branch := chi.URLParam(r, "branch")
	trigger := chi.URLParam(r, "trigger")
	testsuite := chi.URLParam(r, "ts")

	var m ghMatrix
	repoVariations, found := tv[repo]
	if !found {
		http.Error(w, fmt.Sprintf("%s not known", repo), http.StatusNotFound)
	}
	m, found = repoVariations.Leaves[fmt.Sprintf("%s-%s-%s", branch, trigger, testsuite)]
	if !found {
		http.Error(w, fmt.Sprintf("%s-%s-%s not known for %s", branch, trigger, testsuite, repo), http.StatusNotFound)
	}
	render.JSON(w, r, m)
	return
}

func (tv TestsuiteVariations) DumpJson(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, tv)
	return
}
