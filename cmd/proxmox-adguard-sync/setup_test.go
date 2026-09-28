package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
)

func mockSuccessfulSetupConnections(
	t *testing.T,
) {
	t.Helper()

	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	validateProxmoxConnection = func(
		context.Context,
		config.Config,
	) (string, error) {
		return "9.2.4", nil
	}

	validateAdGuardConnection = func(
		context.Context,
		config.Config,
	) (int, error) {
		return 9, nil
	}
}

func TestRunSetupWritesEnvironmentFile(
	t *testing.T,
) {
	mockSuccessfulSetupConnections(t)

	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	input := strings.Join(
		[]string{
			configurationPath,
			"https://proxmox.example:8006/api2/json",
			"adguard-sync@pve!token",
			"proxmox-secret",
			"n",
			"http://adguard.example",
			"admin",
			"adguard-secret",
			"",
			"y",
			"no-monitor, TESTING",
			"120",
			"./data/custom-state.json",
			"y",
			"",
		},
		"\n",
	)

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)
	if err != nil {
		t.Fatalf(
			"runSetup() error = %v",
			err,
		)
	}

	contents, err := os.ReadFile(
		configurationPath,
	)
	if err != nil {
		t.Fatalf(
			"read generated configuration: %v",
			err,
		)
	}

	generated := string(contents)

	expectedValues := []string{
		"PROXMOX_BASE_URL='https://proxmox.example:8006/api2/json'",
		"PROXMOX_TOKEN_ID='adguard-sync@pve!token'",
		"PROXMOX_TOKEN_SECRET='proxmox-secret'",
		"PROXMOX_VERIFY_TLS='false'",
		"ADGUARD_BASE_URL='http://adguard.example'",
		"ADGUARD_USERNAME='admin'",
		"ADGUARD_PASSWORD='adguard-secret'",
		"DNS_SUFFIX='internal'",
		"FILTER_REQUIRE_RUNNING='true'",
		"FILTER_EXCLUDE_TAGS='no-monitor,testing'",
		"SYNC_INTERVAL_SECONDS='120'",
		"STATE_FILE='./data/custom-state.json'",
		"DRY_RUN='true'",
	}

	for _, expected := range expectedValues {
		if !strings.Contains(
			generated,
			expected,
		) {
			t.Errorf(
				"generated configuration missing %q\n%s",
				expected,
				generated,
			)
		}
	}

	expectedOutput := []string{
		"Testing service connectivity",
		"Proxmox connection succeeded: 9.2.4",
		"AdGuard Home connection succeeded: 9 rewrites found",
		"Connectivity tests completed successfully",
		"Configuration written",
	}

	for _, expected := range expectedOutput {
		if !strings.Contains(
			output.String(),
			expected,
		) {
			t.Errorf(
				"output = %q, expected %q",
				output.String(),
				expected,
			)
		}
	}

	if strings.Contains(
		output.String(),
		"proxmox-secret",
	) {
		t.Error(
			"output contains Proxmox secret",
		)
	}

	if strings.Contains(
		output.String(),
		"adguard-secret",
	) {
		t.Error(
			"output contains AdGuard password",
		)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(configurationPath)
		if err != nil {
			t.Fatalf(
				"stat configuration file: %v",
				err,
			)
		}

		if permissions := info.Mode().Perm(); permissions != 0o600 {
			t.Errorf(
				"permissions = %o, expected 600",
				permissions,
			)
		}
	}

	if !strings.Contains(
		errorOutput.String(),
		"TLS certificate verification is disabled",
	) {
		t.Errorf(
			"error output = %q, expected TLS warning",
			errorOutput.String(),
		)
	}
}

func TestRunSetupUsesSafeDefaults(
	t *testing.T,
) {
	mockSuccessfulSetupConnections(t)

	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	input := strings.Join(
		[]string{
			configurationPath,
			"https://proxmox.example:8006/api2/json",
			"sync@pve!token",
			"token-secret",
			"",
			"http://127.0.0.1:80",
			"admin",
			"password",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		},
		"\n",
	)

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)
	if err != nil {
		t.Fatalf(
			"runSetup() error = %v",
			err,
		)
	}

	contents, err := os.ReadFile(
		configurationPath,
	)
	if err != nil {
		t.Fatalf(
			"read generated configuration: %v",
			err,
		)
	}

	generated := string(contents)

	expectedValues := []string{
		"PROXMOX_VERIFY_TLS='true'",
		"DNS_SUFFIX='internal'",
		"STATE_FILE='./data/state.json'",
		"DRY_RUN='true'",
		"FILTER_REQUIRE_RUNNING='false'",
		"SYNC_INTERVAL_SECONDS='60'",
		"LOG_LEVEL='info'",
		"LOG_FORMAT='text'",
	}

	for _, expected := range expectedValues {
		if !strings.Contains(
			generated,
			expected,
		) {
			t.Errorf(
				"generated configuration missing %q",
				expected,
			)
		}
	}
}

func TestRunSetupDoesNotOverwriteWithoutConfirmation(
	t *testing.T,
) {
	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	originalContents := []byte(
		"EXISTING='true'\n",
	)

	if err := os.WriteFile(
		configurationPath,
		originalContents,
		0o600,
	); err != nil {
		t.Fatalf(
			"write existing file: %v",
			err,
		)
	}

	input := configurationPath + "\nn\n"

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)

	if !errors.Is(err, errSetupCancelled) {
		t.Fatalf(
			"runSetup() error = %v, expected errSetupCancelled",
			err,
		)
	}

	contents, err := os.ReadFile(
		configurationPath,
	)
	if err != nil {
		t.Fatalf(
			"read existing file: %v",
			err,
		)
	}

	if !bytes.Equal(
		contents,
		originalContents,
	) {
		t.Errorf(
			"existing configuration was modified: %q",
			contents,
		)
	}
}

func TestRunSetupCanOverwriteExistingFile(
	t *testing.T,
) {
	mockSuccessfulSetupConnections(t)

	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	if err := os.WriteFile(
		configurationPath,
		[]byte("OLD='value'\n"),
		0o600,
	); err != nil {
		t.Fatalf(
			"write existing file: %v",
			err,
		)
	}

	input := strings.Join(
		[]string{
			configurationPath,
			"y",
			"https://proxmox.example:8006/api2/json",
			"sync@pve!token",
			"token-secret",
			"",
			"http://127.0.0.1:80",
			"admin",
			"password",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		},
		"\n",
	)

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)
	if err != nil {
		t.Fatalf(
			"runSetup() error = %v",
			err,
		)
	}

	contents, err := os.ReadFile(
		configurationPath,
	)
	if err != nil {
		t.Fatalf(
			"read replaced file: %v",
			err,
		)
	}

	if strings.Contains(
		string(contents),
		"OLD='value'",
	) {
		t.Error(
			"existing configuration was not replaced",
		)
	}
}

func TestRunSetupProxmoxConnectionFailure(
	t *testing.T,
) {
	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	validateProxmoxConnection = func(
		context.Context,
		config.Config,
	) (string, error) {
		return "",
			errors.New("authentication failed")
	}

	adGuardCalled := false

	validateAdGuardConnection = func(
		context.Context,
		config.Config,
	) (int, error) {
		adGuardCalled = true
		return 0, nil
	}

	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	input := validSetupInput(
		configurationPath,
	)

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)
	if err == nil {
		t.Fatal(
			"runSetup() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"test Proxmox connection",
	) {
		t.Errorf(
			"error = %q, expected Proxmox connection error",
			err,
		)
	}

	if adGuardCalled {
		t.Error(
			"AdGuard validation ran after Proxmox validation failed",
		)
	}

	if _, statErr := os.Stat(configurationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf(
			"configuration file exists after failed Proxmox test",
		)
	}
}

func TestRunSetupAdGuardConnectionFailure(
	t *testing.T,
) {
	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	validateProxmoxConnection = func(
		context.Context,
		config.Config,
	) (string, error) {
		return "9.2.4", nil
	}

	validateAdGuardConnection = func(
		context.Context,
		config.Config,
	) (int, error) {
		return 0,
			errors.New("authentication failed")
	}

	temporaryDirectory := t.TempDir()
	configurationPath := filepath.Join(
		temporaryDirectory,
		".env.local",
	)

	input := validSetupInput(
		configurationPath,
	)

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	err := runSetup(
		strings.NewReader(input),
		&output,
		&errorOutput,
	)
	if err == nil {
		t.Fatal(
			"runSetup() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"test AdGuard Home connection",
	) {
		t.Errorf(
			"error = %q, expected AdGuard connection error",
			err,
		)
	}

	if !strings.Contains(
		output.String(),
		"Proxmox connection succeeded",
	) {
		t.Errorf(
			"output = %q, expected successful Proxmox test",
			output.String(),
		)
	}

	if _, statErr := os.Stat(configurationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf(
			"configuration file exists after failed AdGuard test",
		)
	}
}

func TestSetupAnswersToConfig(
	t *testing.T,
) {
	answers := setupAnswers{
		ProxmoxBaseURL:      "https://proxmox.example/api2/json",
		ProxmoxTokenID:      "sync@pve!token",
		ProxmoxTokenSecret:  "secret",
		ProxmoxVerifyTLS:    true,
		AdGuardBaseURL:      "http://adguard.example",
		AdGuardUsername:     "admin",
		AdGuardPassword:     "password",
		DNSSuffix:           "internal",
		RequireRunning:      true,
		ExcludeTags:         "testing,no-monitor",
		SyncIntervalSeconds: 120,
		StateFile:           "./data/state.json",
		DryRun:              true,
	}

	cfg := setupAnswersToConfig(answers)

	if cfg.Proxmox.BaseURL != answers.ProxmoxBaseURL {
		t.Errorf(
			"Proxmox BaseURL = %q, expected %q",
			cfg.Proxmox.BaseURL,
			answers.ProxmoxBaseURL,
		)
	}

	if cfg.AdGuard.BaseURL != answers.AdGuardBaseURL {
		t.Errorf(
			"AdGuard BaseURL = %q, expected %q",
			cfg.AdGuard.BaseURL,
			answers.AdGuardBaseURL,
		)
	}

	if cfg.SyncInterval != 120*time.Second {
		t.Errorf(
			"SyncInterval = %s, expected 2m0s",
			cfg.SyncInterval,
		)
	}

	if len(cfg.Filters.ExcludeTags) != 2 {
		t.Errorf(
			"ExcludeTags = %#v, expected two values",
			cfg.Filters.ExcludeTags,
		)
	}
}

func TestShellQuote(
	t *testing.T,
) {
	actual := shellQuote(
		`value with ' quote and ! mark`,
	)

	expected := `'value with '\'' quote and ! mark'`

	if actual != expected {
		t.Errorf(
			"shellQuote() = %q, expected %q",
			actual,
			expected,
		)
	}
}

func TestNormalizeCSVInput(
	t *testing.T,
) {
	actual := normalizeCSVInput(
		" no-monitor, Testing, , UAT ",
	)

	expected := "no-monitor,testing,uat"

	if actual != expected {
		t.Errorf(
			"normalizeCSVInput() = %q, expected %q",
			actual,
			expected,
		)
	}
}

func validSetupInput(
	configurationPath string,
) string {
	return strings.Join(
		[]string{
			configurationPath,
			"https://proxmox.example:8006/api2/json",
			"sync@pve!token",
			"token-secret",
			"",
			"http://adguard.example",
			"admin",
			"password",
			"",
			"",
			"",
			"",
			"",
			"y",
			"",
		},
		"\n",
	)
}
