package main

import (
	"fmt"
	"os"

	"github.com/BetaLixT/podsync/internal/config"
	"github.com/BetaLixT/podsync/internal/db"
	"github.com/BetaLixT/podsync/internal/gpodder"
	"github.com/BetaLixT/podsync/internal/ipod"
	"github.com/BetaLixT/podsync/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	podcast := gpodder.New(cfg.GPodderHome)
	device := ipod.New(cfg.IPodMount, cfg.PodcastFolder)

	localDB := db.New(cfg.PodsyncDatabase())
	if err := localDB.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(cfg, podcast, device, localDB); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
