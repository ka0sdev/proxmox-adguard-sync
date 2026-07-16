package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSyncInterval = 60 * time.Second
)

type Config struct {
	Proxmox ProxmoxConfig
	AdGuard AdGuardConfig

	SyncInterval time.Duration
}

type ProxmoxConfig struct {
	BaseURL        string
	APITokenID     string
	APITokenSecret string
}

type AdGuardConfig struct {
	BaseURL  string
	Username string
	Password string
}

func Load() (Config, error) {
	cfg := Config{
		Proxmox: ProxmoxConfig{
			BaseURL:        strings.TrimSpace(os.Getenv("PROXMOX_URL")),
			APITokenID:     strings.TrimSpace(os.Getenv("PROXMOX_TOKEN_ID")),
			APITokenSecret: strings.TrimSpace(os.Getenv("PROXMOX_TOKEN_SECRET")),
		},
		AdGuard: AdGuardConfig{
			BaseURL:  strings.TrimSpace(os.Getenv("ADGUARD_URL")),
			Username: strings.TrimSpace(os.Getenv("ADGUARD_USERNAME")),
			Password: strings.TrimSpace(os.Getenv("ADGUARD_PASSWORD")),
		},
		SyncInterval: defaultSyncInterval,
	}

	if value := strings.TrimSpace(os.Getenv("SYNC_INTERVAL_SECONDS")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf(
				"parse SYNC_INTERVAL_SECONDS: %w",
				err,
			)
		}

		if seconds <= 0 {
			return Config{}, errors.New(
				"SYNC_INTERVAL_SECONDS must be greater than zero",
			)
		}

		cfg.SyncInterval = time.Duration(seconds) * time.Second
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string

	if c.Proxmox.BaseURL == "" {
		missing = append(missing, "PROXMOX_URL")
	}

	if c.Proxmox.APITokenID == "" {
		missing = append(missing, "PROXMOX_TOKEN_ID")
	}

	if c.Proxmox.APITokenSecret == "" {
		missing = append(missing, "PROXMOX_TOKEN_SECRET")
	}

	if c.AdGuard.BaseURL == "" {
		missing = append(missing, "ADGUARD_URL")
	}

	if c.AdGuard.Username == "" {
		missing = append(missing, "ADGUARD_USERNAME")
	}

	if c.AdGuard.Password == "" {
		missing = append(missing, "ADGUARD_PASSWORD")
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"missing required environment variables: %s",
			strings.Join(missing, ", "),
		)
	}

	return nil
}
