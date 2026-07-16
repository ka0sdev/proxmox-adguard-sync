package main

import (
	"fmt"
	"os"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
)

const applicationName = "proxmox-adguard-sync"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", applicationName, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	fmt.Printf(
		"%s initialized with sync interval %s\n",
		applicationName,
		cfg.SyncInterval,
	)

	return nil
}
