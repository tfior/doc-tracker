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
	cfg Config
	db  *sql.DB
	mux *http.ServeMux
}

func NewServer(cfg Config, db *sql.DB, registrars ...RouteRegistrar) *Server {
	s := &Server{
		cfg: cfg,
		db:  db,
		mux: http.NewServeMux(),
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
	return http.ListenAndServe(fmt.Sprintf(":%s", s.cfg.ServerPort), s.mux)
}
