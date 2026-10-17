// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"fmt"
	"net/http"
)

// RouteRegistrar is implemented by each feature module's handler.
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Server struct {
	cfg     Config
	handler http.Handler
}

func NewServer(cfg Config, handler http.Handler) *Server {
	return &Server{cfg: cfg, handler: handler}
}

func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%s", s.cfg.ServerPort), s.handler)
}
