package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDefaultSyncInterval(t *testing.T) {
	setValidEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.SyncInterval != 60*time.Second {
		t.Errorf(
			"SyncInterval = %s, expected %s",
			cfg.SyncInterval,
			60*time.Second,
		)
	}
}

func TestLoadReadsConfiguration(t *testing.T) {
	setValidEnvironment(t)

	t.Setenv("PROXMOX_URL", " https://proxmox.example.com:8006 ")
	t.Setenv("PROXMOX_TOKEN_ID", " root@pam!sync ")
	t.Setenv("PROXMOX_TOKEN_SECRET", " proxmox-secret ")
	t.Setenv("ADGUARD_URL", " http://adguard.example.com ")
	t.Setenv("ADGUARD_USERNAME", " admin ")
	t.Setenv("ADGUARD_PASSWORD", " adguard-secret ")
	t.Setenv("SYNC_INTERVAL_SECONDS", "30")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.Proxmox.BaseURL != "https://proxmox.example.com:8006" {
		t.Errorf(
			"Proxmox.BaseURL = %q, expected %q",
			cfg.Proxmox.BaseURL,
			"https://proxmox.example.com:8006",
		)
	}

	if cfg.Proxmox.APITokenID != "root@pam!sync" {
		t.Errorf(
			"Proxmox.APITokenID = %q, expected %q",
			cfg.Proxmox.APITokenID,
			"root@pam!sync",
		)
	}

	if cfg.Proxmox.APITokenSecret != "proxmox-secret" {
		t.Errorf(
			"Proxmox.APITokenSecret = %q, expected %q",
			cfg.Proxmox.APITokenSecret,
			"proxmox-secret",
		)
	}

	if cfg.AdGuard.BaseURL != "http://adguard.example.com" {
		t.Errorf(
			"AdGuard.BaseURL = %q, expected %q",
			cfg.AdGuard.BaseURL,
			"http://adguard.example.com",
		)
	}

	if cfg.AdGuard.Username != "admin" {
		t.Errorf(
			"AdGuard.Username = %q, expected %q",
			cfg.AdGuard.Username,
			"admin",
		)
	}

	if cfg.AdGuard.Password != "adguard-secret" {
		t.Errorf(
			"AdGuard.Password = %q, expected %q",
			cfg.AdGuard.Password,
			"adguard-secret",
		)
	}

	if cfg.SyncInterval != 30*time.Second {
		t.Errorf(
			"SyncInterval = %s, expected %s",
			cfg.SyncInterval,
			30*time.Second,
		)
	}
}

func TestLoadRejectsMissingRequiredEnvironmentVariables(t *testing.T) {
	clearEnvironment(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() returned nil error, expected missing-variable error")
	}

	expectedVariables := []string{
		"PROXMOX_URL",
		"PROXMOX_TOKEN_ID",
		"PROXMOX_TOKEN_SECRET",
		"ADGUARD_URL",
		"ADGUARD_USERNAME",
		"ADGUARD_PASSWORD",
	}

	for _, variable := range expectedVariables {
		if !strings.Contains(err.Error(), variable) {
			t.Errorf(
				"error %q does not mention missing variable %s",
				err,
				variable,
			)
		}
	}
}

func TestLoadRejectsInvalidSyncInterval(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("SYNC_INTERVAL_SECONDS", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() returned nil error, expected parsing error")
	}

	if !strings.Contains(err.Error(), "parse SYNC_INTERVAL_SECONDS") {
		t.Errorf(
			"error = %q, expected it to mention SYNC_INTERVAL_SECONDS parsing",
			err,
		)
	}
}

func TestLoadRejectsNonPositiveSyncInterval(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{
			name:  "zero",
			value: "0",
		},
		{
			name:  "negative",
			value: "-10",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv("SYNC_INTERVAL_SECONDS", testCase.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() returned nil error, expected validation error")
			}

			if !strings.Contains(
				err.Error(),
				"SYNC_INTERVAL_SECONDS must be greater than zero",
			) {
				t.Errorf(
					"error = %q, expected non-positive interval error",
					err,
				)
			}
		})
	}
}

func setValidEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("PROXMOX_URL", "https://proxmox.example.com:8006")
	t.Setenv("PROXMOX_TOKEN_ID", "root@pam!sync")
	t.Setenv("PROXMOX_TOKEN_SECRET", "proxmox-secret")
	t.Setenv("ADGUARD_URL", "http://adguard.example.com")
	t.Setenv("ADGUARD_USERNAME", "admin")
	t.Setenv("ADGUARD_PASSWORD", "adguard-secret")
	t.Setenv("SYNC_INTERVAL_SECONDS", "")
}

func clearEnvironment(t *testing.T) {
	t.Helper()

	t.Setenv("PROXMOX_URL", "")
	t.Setenv("PROXMOX_TOKEN_ID", "")
	t.Setenv("PROXMOX_TOKEN_SECRET", "")
	t.Setenv("ADGUARD_URL", "")
	t.Setenv("ADGUARD_USERNAME", "")
	t.Setenv("ADGUARD_PASSWORD", "")
	t.Setenv("SYNC_INTERVAL_SECONDS", "")
}
