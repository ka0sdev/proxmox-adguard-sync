package main

import (
	"context"
	"fmt"
	"io"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

var loadValidationConfig = config.Load

var validateProxmoxConnection = func(
	ctx context.Context,
	cfg config.Config,
) (string, error) {
	client, err := proxmox.NewClient(
		proxmox.ClientOptions{
			BaseURL:     cfg.Proxmox.BaseURL,
			TokenID:     cfg.Proxmox.APITokenID,
			TokenSecret: cfg.Proxmox.APITokenSecret,
			VerifyTLS:   cfg.Proxmox.VerifyTLS,
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"initialize Proxmox client: %w",
			err,
		)
	}

	version, err := client.Version(ctx)
	if err != nil {
		return "", fmt.Errorf(
			"connect to Proxmox: %w",
			err,
		)
	}

	if version.Release != "" {
		return fmt.Sprintf(
			"%s (%s)",
			version.Version,
			version.Release,
		), nil
	}

	return version.Version, nil
}

var validateAdGuardConnection = func(
	ctx context.Context,
	cfg config.Config,
) (int, error) {
	client, err := adguard.NewClient(
		adguard.ClientOptions{
			BaseURL:  cfg.AdGuard.BaseURL,
			Username: cfg.AdGuard.Username,
			Password: cfg.AdGuard.Password,
		},
	)
	if err != nil {
		return 0, fmt.Errorf(
			"initialize AdGuard Home client: %w",
			err,
		)
	}

	rewrites, err := client.ListRewrites(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"connect to AdGuard Home: %w",
			err,
		)
	}

	return len(rewrites), nil
}

func runValidation(
	ctx context.Context,
	writer io.Writer,
) error {
	_, _ = fmt.Fprintln(
		writer,
		"Validating configuration and connectivity...",
	)

	cfg, err := loadValidationConfig()
	if err != nil {
		return fmt.Errorf(
			"load configuration: %w",
			err,
		)
	}

	_, _ = fmt.Fprintln(
		writer,
		"✓ Configuration is valid",
	)

	proxmoxVersion, err :=
		validateProxmoxConnection(
			ctx,
			cfg,
		)
	if err != nil {
		return fmt.Errorf(
			"validate Proxmox connection: %w",
			err,
		)
	}

	_, _ = fmt.Fprintf(
		writer,
		"✓ Proxmox connection succeeded: %s\n",
		proxmoxVersion,
	)

	rewriteCount, err :=
		validateAdGuardConnection(
			ctx,
			cfg,
		)
	if err != nil {
		return fmt.Errorf(
			"validate AdGuard Home connection: %w",
			err,
		)
	}

	_, _ = fmt.Fprintf(
		writer,
		"✓ AdGuard Home connection succeeded: %d rewrites found\n",
		rewriteCount,
	)

	_, _ = fmt.Fprintln(
		writer,
		"✓ Validation completed successfully",
	)

	return nil
}
