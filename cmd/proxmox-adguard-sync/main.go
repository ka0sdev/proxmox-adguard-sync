package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/logging"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

const (
	applicationName       = "proxmox-adguard-sync"
	startupRequestTimeout = 15 * time.Second
)

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
		"application initializing",
		slog.String("application", applicationName),
		slog.String("sync_interval", cfg.SyncInterval.String()),
		slog.String("log_level", cfg.Logging.Level),
		slog.String("log_format", cfg.Logging.Format),
		slog.String("proxmox_url", cfg.Proxmox.BaseURL),
		slog.Bool("proxmox_verify_tls", cfg.Proxmox.VerifyTLS),
		slog.String("adguard_url", cfg.AdGuard.BaseURL),
	)

	proxmoxClient, err := proxmox.NewClient(
		proxmox.ClientOptions{
			BaseURL:     cfg.Proxmox.BaseURL,
			TokenID:     cfg.Proxmox.APITokenID,
			TokenSecret: cfg.Proxmox.APITokenSecret,
			VerifyTLS:   cfg.Proxmox.VerifyTLS,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize Proxmox client: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		startupRequestTimeout,
	)
	defer cancel()

	version, err := proxmoxClient.Version(ctx)
	if err != nil {
		return fmt.Errorf("verify Proxmox connection: %w", err)
	}

	logger.Info(
		"connected to Proxmox",
		slog.String("version", version.Version),
		slog.String("release", version.Release),
		slog.String("repository_id", version.RepoID),
	)

	return nil
}
