package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/discovery"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/logging"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/selection"
)

const applicationName = "proxmox-adguard-sync"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%s: %v\n",
			applicationName,
			err,
		)

		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf(
			"load configuration: %w",
			err,
		)
	}

	logger, err := logging.New(
		os.Stdout,
		cfg.Logging.Level,
		cfg.Logging.Format,
	)
	if err != nil {
		return fmt.Errorf(
			"initialize logger: %w",
			err,
		)
	}

	slog.SetDefault(logger)

	logger.Info(
		"Starting application",
		slog.String("application", applicationName),
		slog.String(
			"sync_interval",
			cfg.SyncInterval.String(),
		),
		slog.String(
			"log_level",
			cfg.Logging.Level,
		),
		slog.String(
			"log_format",
			cfg.Logging.Format,
		),
		slog.String(
			"proxmox_url",
			cfg.Proxmox.BaseURL,
		),
		slog.Bool(
			"proxmox_verify_tls",
			cfg.Proxmox.VerifyTLS,
		),
		slog.String(
			"adguard_url",
			cfg.AdGuard.BaseURL,
		),
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
		return fmt.Errorf(
			"initialize Proxmox client: %w",
			err,
		)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	version, err := proxmoxClient.Version(ctx)
	if err != nil {
		return fmt.Errorf(
			"verify Proxmox connection: %w",
			err,
		)
	}

	logger.Info(
		"Connected to Proxmox",
		slog.String("version", version.Version),
		slog.String("release", version.Release),
		slog.String(
			"repository_id",
			version.RepoID,
		),
	)

	guests, err := proxmoxClient.ListGuests(ctx)
	if err != nil {
		return fmt.Errorf(
			"retrieve Proxmox guests: %w",
			err,
		)
	}

	selector := selection.New(cfg.Filters)

	selectedGuests, excludedGuests := selector.Filter(guests)

	logGuestSelection(
		logger,
		guests,
		selectedGuests,
		excludedGuests,
	)

	resolvedGuests := resolveGuests(
		ctx,
		logger,
		proxmoxClient,
		cfg.Discovery,
		selectedGuests,
	)

	// This will be consumed by the AdGuard planning layer later.
	_ = resolvedGuests

	return nil
}

func logGuestSelection(
	logger *slog.Logger,
	allGuests []proxmox.Guest,
	selectedGuests []proxmox.Guest,
	excludedGuests []selection.Result,
) {
	for _, guest := range selectedGuests {
		logger.Debug(
			"Selected guest",
			slog.Int("vmid", guest.VMID),
			slog.String("name", guest.Name),
			slog.String(
				"type",
				string(guest.Type),
			),
			slog.String("status", guest.Status),
			slog.Any("tags", guest.ParsedTags()),
		)
	}

	for _, result := range excludedGuests {
		logger.Debug(
			"Excluded guest",
			slog.Int(
				"vmid",
				result.Guest.VMID,
			),
			slog.String(
				"name",
				result.Guest.Name,
			),
			slog.String(
				"type",
				string(result.Guest.Type),
			),
			slog.String(
				"reason",
				string(result.Reason),
			),
			slog.Any(
				"tags",
				result.GuestTags,
			),
		)
	}

	logger.Info(
		"Guest filtering complete",
		slog.Int(
			"discovered",
			len(allGuests),
		),
		slog.Int(
			"selected",
			len(selectedGuests),
		),
		slog.Int(
			"excluded",
			len(excludedGuests),
		),
	)
}

func resolveGuests(
	ctx context.Context,
	logger *slog.Logger,
	client *proxmox.Client,
	discoveryConfig config.DiscoveryConfig,
	guests []proxmox.Guest,
) []discovery.ResolvedGuest {
	resolver := discovery.NewResolver(discoveryConfig)

	resolved := make(
		[]discovery.ResolvedGuest,
		0,
		len(guests),
	)

	var failed int

	for _, guest := range guests {
		guestConfig, err := retrieveGuestConfig(
			ctx,
			client,
			guest,
		)
		if err != nil {
			failed++

			logger.Warn(
				"Guest configuration retrieval failed",
				slog.Int("vmid", guest.VMID),
				slog.String("name", guest.Name),
				slog.String(
					"type",
					string(guest.Type),
				),
				slog.String(
					"error",
					err.Error(),
				),
			)

			continue
		}

		var agentInterfaces []proxmox.QEMUAgentInterface

		if guest.Type == proxmox.GuestTypeQEMU &&
			discoverySourceEnabled(
				discoveryConfig.QEMUOrder,
				"guest-agent",
			) {
			agentInterfaces, err =
				client.GetQEMUAgentInterfaces(
					ctx,
					guest.Node,
					guest.VMID,
				)

			if err != nil {
				logger.Debug(
					"QEMU Guest Agent unavailable",
					slog.Int("vmid", guest.VMID),
					slog.String("name", guest.Name),
					slog.String(
						"error",
						err.Error(),
					),
				)

				agentInterfaces = nil
			}
		}

		result, err := resolver.ResolveWithQEMUAgent(
			guest,
			guestConfig,
			agentInterfaces,
		)
		if err != nil {
			failed++

			logger.Warn(
				"Guest resolution failed",
				slog.Int("vmid", guest.VMID),
				slog.String("name", guest.Name),
				slog.String(
					"type",
					string(guest.Type),
				),
				slog.String(
					"error",
					err.Error(),
				),
			)

			continue
		}

		resolved = append(resolved, result)

		logResolvedGuest(logger, result)
	}

	logger.Info(
		"Guest resolution complete",
		slog.Int("selected", len(guests)),
		slog.Int("resolved", len(resolved)),
		slog.Int("failed", failed),
	)

	return resolved
}

func retrieveGuestConfig(
	ctx context.Context,
	client *proxmox.Client,
	guest proxmox.Guest,
) (proxmox.GuestConfig, error) {
	switch guest.Type {
	case proxmox.GuestTypeLXC:
		return client.GetLXCConfig(
			ctx,
			guest.Node,
			guest.VMID,
		)

	case proxmox.GuestTypeQEMU:
		return client.GetQEMUConfig(
			ctx,
			guest.Node,
			guest.VMID,
		)

	default:
		return nil, fmt.Errorf(
			"unsupported guest type %q",
			guest.Type,
		)
	}
}

func discoverySourceEnabled(
	sources []string,
	wanted string,
) bool {
	for _, source := range sources {
		if source == wanted {
			return true
		}
	}

	return false
}

func logResolvedGuest(
	logger *slog.Logger,
	result discovery.ResolvedGuest,
) {
	attributes := []any{
		slog.Int(
			"vmid",
			result.Guest.VMID,
		),
		slog.String(
			"name",
			result.Guest.Name,
		),
		slog.String(
			"hostname",
			result.Hostname,
		),
		slog.String(
			"address",
			result.Address.String(),
		),
		slog.String(
			"source",
			string(result.Source),
		),
	}

	if result.InterfaceName != "" {
		attributes = append(
			attributes,
			slog.String(
				"interface",
				result.InterfaceName,
			),
		)
	}

	logger.Info(
		"Resolved guest",
		attributes...,
	)
}
