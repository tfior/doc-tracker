package platform

import (
	"database/sql"
	"fmt"
	"net/http"
)

type Server struct {
	cfg Config
	db  *sql.DB
	mux *http.ServeMux
}

func NewServer(cfg Config, db *sql.DB) *Server {
	s := &Server{
		cfg: cfg,
		db:  db,
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%s", s.cfg.ServerPort), s.mux)
}
