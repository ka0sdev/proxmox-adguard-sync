package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
)

func TestRunValidationSuccess(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig
	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{}, nil
	}

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

	var output bytes.Buffer

	err := runValidation(
		context.Background(),
		&output,
	)
	if err != nil {
		t.Fatalf(
			"runValidation() error = %v",
			err,
		)
	}

	expectedMessages := []string{
		"Validating configuration and connectivity",
		"Configuration is valid",
		"Proxmox connection succeeded: 9.2.4",
		"AdGuard Home connection succeeded: 9 rewrites found",
		"Validation completed successfully",
	}

	for _, expected := range expectedMessages {
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
}

func TestRunValidationConfigurationFailure(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{},
			errors.New("missing configuration")
	}

	var output bytes.Buffer

	err := runValidation(
		context.Background(),
		&output,
	)
	if err == nil {
		t.Fatal(
			"runValidation() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"load configuration",
	) {
		t.Errorf(
			"error = %q, expected configuration error",
			err,
		)
	}

	if strings.Contains(
		output.String(),
		"Configuration is valid",
	) {
		t.Errorf(
			"output = %q, configuration should not be valid",
			output.String(),
		)
	}
}

func TestRunValidationProxmoxFailure(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig
	originalValidateProxmox :=
		validateProxmoxConnection

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
			validateProxmoxConnection =
				originalValidateProxmox
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{}, nil
	}

	validateProxmoxConnection = func(
		context.Context,
		config.Config,
	) (string, error) {
		return "",
			errors.New("authentication failed")
	}

	var output bytes.Buffer

	err := runValidation(
		context.Background(),
		&output,
	)
	if err == nil {
		t.Fatal(
			"runValidation() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"validate Proxmox connection",
	) {
		t.Errorf(
			"error = %q, expected Proxmox error",
			err,
		)
	}

	if strings.Contains(
		output.String(),
		"AdGuard Home connection succeeded",
	) {
		t.Errorf(
			"output = %q, AdGuard validation should not run",
			output.String(),
		)
	}
}

func TestRunValidationAdGuardFailure(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig
	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{}, nil
	}

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

	var output bytes.Buffer

	err := runValidation(
		context.Background(),
		&output,
	)
	if err == nil {
		t.Fatal(
			"runValidation() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"validate AdGuard Home connection",
	) {
		t.Errorf(
			"error = %q, expected AdGuard error",
			err,
		)
	}
}
