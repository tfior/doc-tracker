package main

import (
	"log"

	"github.com/tfior/doc-tracker/internal/cases"
	"github.com/tfior/doc-tracker/internal/people"
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

	casesHandler := cases.NewHandler(cases.NewService(cases.NewStore(db)))
	peopleHandler := people.NewHandler(people.NewService(people.NewStore(db)))

	srv := platform.NewServer(cfg, db, casesHandler, peopleHandler)

	log.Printf("server listening on :%s", cfg.ServerPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
