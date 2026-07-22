package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/discovery"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/logging"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/reconcile"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/selection"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/state"
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
		slog.String(
			"application",
			applicationName,
		),
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
		slog.String(
			"dns_suffix",
			cfg.DNS.Suffix,
		),
		slog.String(
			"state_file",
			cfg.State.File,
		),
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

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

	version, err := proxmoxClient.Version(ctx)
	if err != nil {
		return fmt.Errorf(
			"verify Proxmox connection: %w",
			err,
		)
	}

	logger.Info(
		"Connected to Proxmox",
		slog.String(
			"version",
			version.Version,
		),
		slog.String(
			"release",
			version.Release,
		),
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

	selectedGuests, excludedGuests :=
		selector.Filter(guests)

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

	desiredRewrites, err :=
		reconcile.BuildDesiredRewrites(
			resolvedGuests,
			cfg.DNS.Suffix,
		)
	if err != nil {
		return fmt.Errorf(
			"build desired DNS rewrites: %w",
			err,
		)
	}

	adguardClient, err := adguard.NewClient(
		adguard.ClientOptions{
			BaseURL:  cfg.AdGuard.BaseURL,
			Username: cfg.AdGuard.Username,
			Password: cfg.AdGuard.Password,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"initialize AdGuard client: %w",
			err,
		)
	}

	currentRewrites, err :=
		adguardClient.ListRewrites(ctx)
	if err != nil {
		return fmt.Errorf(
			"retrieve AdGuard rewrites: %w",
			err,
		)
	}

	stateStore, err := state.NewStore(
		cfg.State.File,
	)
	if err != nil {
		return fmt.Errorf(
			"initialize state store: %w",
			err,
		)
	}

	stateFile, err := stateStore.Load()
	if err != nil {
		return fmt.Errorf(
			"load ownership state: %w",
			err,
		)
	}

	logger.Info(
		"Ownership state loaded",
		slog.String(
			"path",
			stateStore.Path(),
		),
		slog.Int(
			"managed",
			len(stateFile.Records),
		),
	)

	plan := reconcile.BuildPlan(
		desiredRewrites,
		currentRewrites,
		stateFile.ManagedRecords(),
	)

	logReconciliationPlan(
		logger,
		plan,
	)

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
			slog.Int(
				"vmid",
				guest.VMID,
			),
			slog.String(
				"name",
				guest.Name,
			),
			slog.String(
				"type",
				string(guest.Type),
			),
			slog.String(
				"status",
				guest.Status,
			),
			slog.Any(
				"tags",
				guest.ParsedTags(),
			),
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
	resolver := discovery.NewResolver(
		discoveryConfig,
	)

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
				slog.Int(
					"vmid",
					guest.VMID,
				),
				slog.String(
					"name",
					guest.Name,
				),
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
					slog.Int(
						"vmid",
						guest.VMID,
					),
					slog.String(
						"name",
						guest.Name,
					),
					slog.String(
						"error",
						err.Error(),
					),
				)

				agentInterfaces = nil
			}
		}

		result, err :=
			resolver.ResolveWithQEMUAgent(
				guest,
				guestConfig,
				agentInterfaces,
			)
		if err != nil {
			failed++

			logger.Warn(
				"Guest resolution failed",
				slog.Int(
					"vmid",
					guest.VMID,
				),
				slog.String(
					"name",
					guest.Name,
				),
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

		resolved = append(
			resolved,
			result,
		)

		logResolvedGuest(
			logger,
			result,
		)
	}

	logger.Info(
		"Guest resolution complete",
		slog.Int(
			"selected",
			len(guests),
		),
		slog.Int(
			"resolved",
			len(resolved),
		),
		slog.Int(
			"failed",
			failed,
		),
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

func logReconciliationPlan(
	logger *slog.Logger,
	plan reconcile.Plan,
) {
	for _, rewrite := range plan.Add {
		logger.Info(
			"DNS rewrite would be added",
			slog.String(
				"domain",
				rewrite.Domain,
			),
			slog.String(
				"answer",
				rewrite.Answer,
			),
		)
	}

	for _, change := range plan.Update {
		logger.Info(
			"DNS rewrite would be updated",
			slog.String(
				"domain",
				change.Desired.Domain,
			),
			slog.String(
				"current_answer",
				change.Current.Answer,
			),
			slog.String(
				"desired_answer",
				change.Desired.Answer,
			),
		)
	}

	for _, rewrite := range plan.Delete {
		logger.Info(
			"Managed DNS rewrite would be deleted",
			slog.String(
				"domain",
				rewrite.Domain,
			),
			slog.String(
				"answer",
				rewrite.Answer,
			),
		)
	}

	for _, rewrite := range plan.Unchanged {
		logger.Debug(
			"DNS rewrite already current",
			slog.String(
				"domain",
				rewrite.Domain,
			),
			slog.String(
				"answer",
				rewrite.Answer,
			),
		)
	}

	for _, rewrite := range plan.Unmanaged {
		logger.Debug(
			"Leaving unrelated DNS rewrite unchanged",
			slog.String(
				"domain",
				rewrite.Domain,
			),
			slog.String(
				"answer",
				rewrite.Answer,
			),
		)
	}

	logger.Info(
		"DNS reconciliation plan complete",
		slog.Int(
			"add",
			len(plan.Add),
		),
		slog.Int(
			"update",
			len(plan.Update),
		),
		slog.Int(
			"delete",
			len(plan.Delete),
		),
		slog.Int(
			"unchanged",
			len(plan.Unchanged),
		),
		slog.Int(
			"unmanaged",
			len(plan.Unmanaged),
		),
		slog.Bool(
			"dry_run",
			true,
		),
	)
}
