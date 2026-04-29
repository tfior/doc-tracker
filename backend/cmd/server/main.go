package main

import (
	"log"

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

func main() {
	cfg := platform.LoadConfig()

	if err := platform.RunMigrations(cfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	db, err := platform.OpenDatabase(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	authStore := auth.NewSessionStore()
	authSvc := auth.NewService(authStore, users.NewService(users.NewStore(db)))
	actlogSvc := activitylog.NewService(activitylog.NewStore(db))

	srv := platform.NewServer(cfg, db,
		auth.Middleware(authSvc),
		auth.NewHandler(authSvc),
		cases.NewHandler(cases.NewService(cases.NewStore(db)), actlogSvc),
		people.NewHandler(people.NewService(people.NewStore(db)), actlogSvc),
		claimlines.NewHandler(claimlines.NewService(claimlines.NewStore(db)), actlogSvc),
		lifeevents.NewHandler(lifeevents.NewService(lifeevents.NewStore(db)), actlogSvc),
		documents.NewHandler(documents.NewService(documents.NewStore(db)), actlogSvc),
		trash.NewHandler(trash.NewService(trash.NewStore(db))),
	)

	log.Printf("server listening on :%s", cfg.ServerPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
