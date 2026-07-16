package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/logging"
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

	logger, err := logging.New(
		os.Stdout,
		cfg.Logging.Level,
		cfg.Logging.Format,
	)
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	slog.SetDefault(logger)

	logger.Info(
		"application initialized",
		slog.String("application", applicationName),
		slog.Duration("sync_interval", cfg.SyncInterval),
		slog.String("log_level", cfg.Logging.Level),
		slog.String("log_format", cfg.Logging.Format),
		slog.String("proxmox_url", cfg.Proxmox.BaseURL),
		slog.String("adguard_url", cfg.AdGuard.BaseURL),
	)

	return nil
}
