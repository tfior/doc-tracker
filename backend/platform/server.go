package platform

import (
	"database/sql"
	"fmt"
	"net/http"
)

// RouteRegistrar is implemented by each feature module's handler.
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Server struct {
	cfg        Config
	db         *sql.DB
	mux        *http.ServeMux
	middleware func(http.Handler) http.Handler
}

func NewServer(cfg Config, db *sql.DB, middleware func(http.Handler) http.Handler, registrars ...RouteRegistrar) *Server {
	s := &Server{
		cfg:        cfg,
		db:         db,
		mux:        http.NewServeMux(),
		middleware: middleware,
	}
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	for _, r := range registrars {
		r.RegisterRoutes(s.mux)
	}
	return s
}

func (s *Server) Start() error {
	var handler http.Handler = s.mux
	if s.middleware != nil {
		handler = s.middleware(handler)
	}
	return http.ListenAndServe(fmt.Sprintf(":%s", s.cfg.ServerPort), handler)
}
