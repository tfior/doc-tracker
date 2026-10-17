// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package app assembles the HTTP application from its modules. It is the only
// place where modules are wired together, and it is used both by the server
// and by tests, so tests exercise exactly what runs in production.
package app

import (
	"database/sql"
	"net/http"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/internal/auth"
	"github.com/tfior/doc-tracker/internal/cases"
	"github.com/tfior/doc-tracker/internal/claimlines"
	"github.com/tfior/doc-tracker/internal/documents"
	"github.com/tfior/doc-tracker/internal/lifeevents"
	"github.com/tfior/doc-tracker/internal/people"
	"github.com/tfior/doc-tracker/internal/trash"
	"github.com/tfior/doc-tracker/internal/users"
	"github.com/tfior/doc-tracker/platform"
)

// NewHandler returns the complete HTTP handler: every module's routes behind
// the authentication middleware.
func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	authSvc := auth.NewService(auth.NewSessionStore(), users.NewService(users.NewStore(db)))
	actlogSvc := activitylog.NewService(activitylog.NewStore(db), auth.UserIDFromContext)

	casesSvc := cases.NewService(cases.NewStore(db), db, actlogSvc)
	peopleSvc := people.NewService(people.NewStore(db), db, actlogSvc)
	lifeEventsSvc := lifeevents.NewService(lifeevents.NewStore(db), db, actlogSvc, peopleSvc)
	documentsSvc := documents.NewService(documents.NewStore(db), db, actlogSvc, peopleSvc, lifeEventsSvc)
	claimLinesSvc := claimlines.NewService(claimlines.NewStore(db), db, actlogSvc, peopleSvc)
	trashSvc := trash.NewService(trash.NewStore(db), db, trash.Owners{
		Cases:      casesSvc,
		People:     peopleSvc,
		LifeEvents: lifeEventsSvc,
		Documents:  documentsSvc,
		ClaimLines: claimLinesSvc,
	})

	for _, r := range []platform.RouteRegistrar{
		auth.NewHandler(authSvc),
		cases.NewHandler(casesSvc),
		people.NewHandler(peopleSvc),
		claimlines.NewHandler(claimLinesSvc),
		lifeevents.NewHandler(lifeEventsSvc),
		documents.NewHandler(documentsSvc),
		trash.NewHandler(trashSvc),
	} {
		r.RegisterRoutes(mux)
	}

	return auth.Middleware(authSvc)(mux)
}
