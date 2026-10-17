// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"log"

	"github.com/tfior/doc-tracker/internal/app"
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

	srv := platform.NewServer(cfg, app.NewHandler(db))

	log.Printf("server listening on :%s", cfg.ServerPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
