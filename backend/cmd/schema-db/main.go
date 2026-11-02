// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Command schema-db drops and recreates a scratch database, named by
// SCHEMA_DB_NAME (default doctracker_schema), then runs every migration
// against it. It never touches DB_NAME, so the dev database is safe.
// `make schema` dumps the result to db/schema.sql, so the snapshot always
// comes from the migrations alone.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/tfior/doc-tracker/platform"
)

func main() {
	cfg := platform.LoadConfig()
	cfg.DBName = os.Getenv("SCHEMA_DB_NAME")
	if cfg.DBName == "" {
		cfg.DBName = "doctracker_schema"
	}

	admin := cfg
	admin.DBName = "postgres"
	db, err := platform.OpenDatabase(admin)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	if err := recreate(db, cfg.DBName); err != nil {
		log.Fatalf("recreate %s: %v", cfg.DBName, err)
	}
	if err := platform.RunMigrations(cfg); err != nil {
		log.Fatalf("migrate %s: %v", cfg.DBName, err)
	}
}

func recreate(db *sql.DB, name string) error {
	if name == "postgres" {
		return fmt.Errorf("refusing to recreate the postgres system database")
	}
	quoted := `"` + name + `"`
	if _, err := db.Exec("DROP DATABASE IF EXISTS " + quoted + " WITH (FORCE)"); err != nil {
		return err
	}
	_, err := db.Exec("CREATE DATABASE " + quoted)
	return err
}
