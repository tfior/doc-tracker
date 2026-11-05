// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Command seed fills a development database with example cases. It is for
// development only.
package main

import (
	"context"
	"log"

	"github.com/tfior/doc-tracker/internal/seed"
	"github.com/tfior/doc-tracker/platform"
)

func main() {
	cfg := platform.LoadConfig()

	db, err := platform.OpenDatabase(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	if err := seed.Run(context.Background(), db); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seed complete")
}
