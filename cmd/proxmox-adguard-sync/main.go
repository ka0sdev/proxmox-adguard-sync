package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

type application struct {
	config        config.Config
	logger        *slog.Logger
	proxmoxClient *proxmox.Client
	adguardClient *adguard.Client
	stateStore    *state.Store
}

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

	stateStore, err := state.NewStore(cfg.State.File)
	if err != nil {
		return fmt.Errorf(
			"initialize state store: %w",
			err,
		)
	}

	app := application{
		config:        cfg,
		logger:        logger,
		proxmoxClient: proxmoxClient,
		adguardClient: adguardClient,
		stateStore:    stateStore,
	}

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
		slog.Bool(
			"dry_run",
			cfg.Runtime.DryRun,
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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.verifyConnections(ctx); err != nil {
		return err
	}

	if err := app.runLoop(ctx); err != nil {
		return err
	}

	logger.Info("Application stopped")

	return nil
}

func (a *application) verifyConnections(
	ctx context.Context,
) error {
	version, err := a.proxmoxClient.Version(ctx)
	if err != nil {
		return fmt.Errorf(
			"verify Proxmox connection: %w",
			err,
		)
	}

	a.logger.Info(
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

	if _, err := a.adguardClient.ListRewrites(ctx); err != nil {
		return fmt.Errorf(
			"verify AdGuard connection: %w",
			err,
		)
	}

	a.logger.Info("Connected to AdGuard Home")

	return nil
}

func (a *application) runLoop(
	ctx context.Context,
) error {
	if err := a.synchronize(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		a.logger.Error(
			"Initial synchronization failed",
			slog.String(
				"error",
				err.Error(),
			),
		)
	}

	ticker := time.NewTicker(
		a.config.SyncInterval,
	)
	defer ticker.Stop()

	a.logger.Info(
		"Synchronization loop started",
		slog.String(
			"interval",
			a.config.SyncInterval.String(),
		),
	)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info(
				"Shutdown signal received",
			)

			return nil

		case tickTime := <-ticker.C:
			a.logger.Debug(
				"Scheduled synchronization started",
				slog.Time(
					"scheduled_at",
					tickTime,
				),
			)

			if err := a.synchronize(ctx); err != nil {
				if errors.Is(
					err,
					context.Canceled,
				) {
					return nil
				}

				a.logger.Error(
					"Scheduled synchronization failed",
					slog.String(
						"error",
						err.Error(),
					),
				)
			}
		}
	}
}

func (a *application) synchronize(
	ctx context.Context,
) error {
	startedAt := time.Now()

	a.logger.Info("Synchronization started")

	guests, err := a.proxmoxClient.ListGuests(ctx)
	if err != nil {
		return fmt.Errorf(
			"retrieve Proxmox guests: %w",
			err,
		)
	}

	selector := selection.New(
		a.config.Filters,
	)

	selectedGuests, excludedGuests :=
		selector.Filter(guests)

	logGuestSelection(
		a.logger,
		guests,
		selectedGuests,
		excludedGuests,
	)

	resolvedGuests := resolveGuests(
		ctx,
		a.logger,
		a.proxmoxClient,
		a.config.Discovery,
		selectedGuests,
	)

	desiredRewrites, err :=
		reconcile.BuildDesiredRewrites(
			resolvedGuests,
			a.config.DNS.Suffix,
		)
	if err != nil {
		return fmt.Errorf(
			"build desired DNS rewrites: %w",
			err,
		)
	}

	currentRewrites, err :=
		a.adguardClient.ListRewrites(ctx)
	if err != nil {
		return fmt.Errorf(
			"retrieve AdGuard rewrites: %w",
			err,
		)
	}

	stateFile, err := a.stateStore.Load()
	if err != nil {
		return fmt.Errorf(
			"load ownership state: %w",
			err,
		)
	}

	a.logger.Info(
		"Ownership state loaded",
		slog.String(
			"path",
			a.stateStore.Path(),
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
		a.logger,
		plan,
		a.config.Runtime.DryRun,
	)

	if a.config.Runtime.DryRun {
		a.logger.Info(
			"Dry run complete",
			slog.Int(
				"planned_changes",
				len(plan.Add)+
					len(plan.Update)+
					len(plan.Delete),
			),
			slog.Duration(
				"duration",
				time.Since(startedAt),
			),
		)

		return nil
	}

	executionResult, err := reconcile.Execute(
		ctx,
		a.adguardClient,
		plan,
	)
	if err != nil {
		return fmt.Errorf(
			"apply DNS reconciliation plan: %w",
			err,
		)
	}

	a.logger.Info(
		"DNS reconciliation applied",
		slog.Int(
			"added",
			executionResult.Added,
		),
		slog.Int(
			"updated",
			executionResult.Updated,
		),
		slog.Int(
			"deleted",
			executionResult.Deleted,
		),
	)

	nextState := buildOwnershipState(
		desiredRewrites,
	)

	if err := a.stateStore.Save(nextState); err != nil {
		return fmt.Errorf(
			"save ownership state: %w",
			err,
		)
	}

	a.logger.Info(
		"Ownership state saved",
		slog.String(
			"path",
			a.stateStore.Path(),
		),
		slog.Int(
			"managed",
			len(nextState.Records),
		),
	)

	a.logger.Info(
		"Synchronization complete",
		slog.Int(
			"desired",
			len(desiredRewrites),
		),
		slog.Int(
			"changes",
			executionResult.Added+
				executionResult.Updated+
				executionResult.Deleted,
		),
		slog.Duration(
			"duration",
			time.Since(startedAt),
		),
	)

	return nil
}

func buildOwnershipState(
	desiredRewrites []adguard.Rewrite,
) state.File {
	nextState := state.File{
		Records: make(
			[]state.Record,
			0,
			len(desiredRewrites),
		),
	}

	for _, rewrite := range desiredRewrites {
		nextState.Records = append(
			nextState.Records,
			state.Record{
				Domain: rewrite.Domain,
				Answer: rewrite.Answer,
			},
		)
	}

	return nextState
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
		if err := ctx.Err(); err != nil {
			break
		}

		guestConfig, err := retrieveGuestConfig(
			ctx,
			client,
			guest,
		)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}

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
				if errors.Is(
					err,
					context.Canceled,
				) {
					break
				}

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
	dryRun bool,
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
			dryRun,
		),
	)
}
