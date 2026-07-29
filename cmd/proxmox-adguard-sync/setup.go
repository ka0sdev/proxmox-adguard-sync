package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	defaultSetupFile         = ".env.local"
	defaultDNSSuffix         = "internal"
	defaultStateFile         = "./data/state.json"
	defaultSetupSyncInterval = 60
)

var errSetupCancelled = errors.New(
	"setup cancelled",
)

type setupAnswers struct {
	OutputFile string

	ProxmoxBaseURL     string
	ProxmoxTokenID     string
	ProxmoxTokenSecret string
	ProxmoxVerifyTLS   bool

	AdGuardBaseURL  string
	AdGuardUsername string
	AdGuardPassword string

	DNSSuffix string

	RequireRunning bool
	ExcludeTags    string

	SyncIntervalSeconds int
	StateFile           string
	DryRun              bool
}

type setupWizard struct {
	reader      *bufio.Reader
	input       io.Reader
	output      io.Writer
	errorOutput io.Writer
}

func runSetup(
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
) error {
	wizard := setupWizard{
		reader:      bufio.NewReader(input),
		input:       input,
		output:      output,
		errorOutput: errorOutput,
	}

	answers, err := wizard.collectAnswers()
	if err != nil {
		return err
	}

	wizard.printSummary(answers)

	confirmed, err := wizard.promptBool(
		"Write configuration?",
		true,
	)
	if err != nil {
		return err
	}

	if !confirmed {
		return errSetupCancelled
	}

	if err := writeSetupEnvironmentFile(
		answers.OutputFile,
		answers,
	); err != nil {
		return fmt.Errorf(
			"write configuration file: %w",
			err,
		)
	}

	_, _ = fmt.Fprintf(
		output,
		"\n✓ Configuration written to %s\n",
		answers.OutputFile,
	)

	_, _ = fmt.Fprintln(
		output,
		"✓ File permissions set to 0600",
	)

	_, _ = fmt.Fprintln(
		output,
		"✓ Dry-run mode is enabled",
	)

	_, _ = fmt.Fprintln(
		output,
		"\nLoad the configuration before running the service:",
	)

	_, _ = fmt.Fprintf(
		output,
		"  set -a && source %s && set +a\n",
		shellDisplayPath(answers.OutputFile),
	)

	_, _ = fmt.Fprintln(
		output,
		"\nThen validate it:",
	)

	_, _ = fmt.Fprintln(
		output,
		"  ./bin/proxmox-adguard-sync validate",
	)

	return nil
}

func (w *setupWizard) collectAnswers() (
	setupAnswers,
	error,
) {
	_, _ = fmt.Fprintln(
		w.output,
		"Proxmox AdGuard Sync Setup",
	)

	_, _ = fmt.Fprintln(
		w.output,
		"\nThis wizard creates an environment configuration file.",
	)

	_, _ = fmt.Fprintln(
		w.output,
		"Dry-run mode will be enabled by default.",
	)

	outputFile, err := w.promptString(
		"\nConfiguration file",
		defaultSetupFile,
		false,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	outputFile = filepath.Clean(
		expandHomeDirectory(outputFile),
	)

	if err := w.confirmOverwrite(outputFile); err != nil {
		return setupAnswers{}, err
	}

	proxmoxURL, err := w.promptString(
		"\nProxmox API URL",
		"",
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	tokenID, err := w.promptString(
		"Proxmox token ID",
		"",
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	tokenSecret, err := w.promptSecret(
		"Proxmox token secret",
	)
	if err != nil {
		return setupAnswers{}, err
	}

	verifyTLS, err := w.promptBool(
		"Verify Proxmox TLS certificates?",
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	if !verifyTLS {
		_, _ = fmt.Fprintln(
			w.errorOutput,
			"Warning: Proxmox TLS certificate verification is disabled.",
		)
	}

	adGuardURL, err := w.promptString(
		"\nAdGuard Home URL",
		"",
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	adGuardUsername, err := w.promptString(
		"AdGuard Home username",
		"",
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	adGuardPassword, err := w.promptSecret(
		"AdGuard Home password",
	)
	if err != nil {
		return setupAnswers{}, err
	}

	dnsSuffix, err := w.promptString(
		"\nDNS suffix",
		defaultDNSSuffix,
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	requireRunning, err := w.promptBool(
		"Only include running guests?",
		false,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	excludeTags, err := w.promptString(
		"Excluded tags, comma separated",
		"",
		false,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	syncInterval, err := w.promptPositiveInteger(
		"Synchronization interval in seconds",
		defaultSetupSyncInterval,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	stateFile, err := w.promptString(
		"State file",
		defaultStateFile,
		true,
	)
	if err != nil {
		return setupAnswers{}, err
	}

	return setupAnswers{
		OutputFile: outputFile,

		ProxmoxBaseURL: strings.TrimRight(
			strings.TrimSpace(proxmoxURL),
			"/",
		),
		ProxmoxTokenID: strings.TrimSpace(
			tokenID,
		),
		ProxmoxTokenSecret: tokenSecret,
		ProxmoxVerifyTLS:   verifyTLS,

		AdGuardBaseURL: strings.TrimRight(
			strings.TrimSpace(adGuardURL),
			"/",
		),
		AdGuardUsername: strings.TrimSpace(
			adGuardUsername,
		),
		AdGuardPassword: adGuardPassword,

		DNSSuffix: strings.ToLower(
			strings.TrimSpace(dnsSuffix),
		),

		RequireRunning: requireRunning,
		ExcludeTags: normalizeCSVInput(
			excludeTags,
		),

		SyncIntervalSeconds: syncInterval,
		StateFile: strings.TrimSpace(
			stateFile,
		),
		DryRun: true,
	}, nil
}

func (w *setupWizard) confirmOverwrite(
	path string,
) error {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		confirmed, promptErr := w.promptBool(
			fmt.Sprintf(
				"%s already exists. Overwrite it?",
				path,
			),
			false,
		)
		if promptErr != nil {
			return promptErr
		}

		if !confirmed {
			return errSetupCancelled
		}

		return nil

	case errors.Is(err, os.ErrNotExist):
		return nil

	default:
		return fmt.Errorf(
			"inspect configuration file: %w",
			err,
		)
	}
}

func (w *setupWizard) promptString(
	label string,
	defaultValue string,
	required bool,
) (string, error) {
	for {
		if defaultValue == "" {
			_, _ = fmt.Fprintf(
				w.output,
				"%s: ",
				label,
			)
		} else {
			_, _ = fmt.Fprintf(
				w.output,
				"%s [%s]: ",
				label,
				defaultValue,
			)
		}

		value, err := w.readLine()
		if err != nil {
			return "", fmt.Errorf(
				"read %s: %w",
				label,
				err,
			)
		}

		value = strings.TrimSpace(value)

		if value == "" {
			value = defaultValue
		}

		if required && value == "" {
			_, _ = fmt.Fprintln(
				w.errorOutput,
				"A value is required.",
			)

			continue
		}

		return value, nil
	}
}

func (w *setupWizard) promptSecret(
	label string,
) (string, error) {
	for {
		_, _ = fmt.Fprintf(
			w.output,
			"%s: ",
			label,
		)

		value, err := w.readSecret()
		if err != nil {
			return "", fmt.Errorf(
				"read %s: %w",
				label,
				err,
			)
		}

		value = strings.TrimSpace(value)

		if value == "" {
			_, _ = fmt.Fprintln(
				w.errorOutput,
				"A value is required.",
			)

			continue
		}

		return value, nil
	}
}

func (w *setupWizard) promptBool(
	label string,
	defaultValue bool,
) (bool, error) {
	defaultPrompt := "y/N"
	if defaultValue {
		defaultPrompt = "Y/n"
	}

	for {
		_, _ = fmt.Fprintf(
			w.output,
			"%s [%s]: ",
			label,
			defaultPrompt,
		)

		value, err := w.readLine()
		if err != nil {
			return false, fmt.Errorf(
				"read %s: %w",
				label,
				err,
			)
		}

		switch strings.ToLower(
			strings.TrimSpace(value),
		) {
		case "":
			return defaultValue, nil

		case "y", "yes", "true", "1":
			return true, nil

		case "n", "no", "false", "0":
			return false, nil

		default:
			_, _ = fmt.Fprintln(
				w.errorOutput,
				"Enter yes or no.",
			)
		}
	}
}

func (w *setupWizard) promptPositiveInteger(
	label string,
	defaultValue int,
) (int, error) {
	for {
		_, _ = fmt.Fprintf(
			w.output,
			"%s [%d]: ",
			label,
			defaultValue,
		)

		value, err := w.readLine()
		if err != nil {
			return 0, fmt.Errorf(
				"read %s: %w",
				label,
				err,
			)
		}

		value = strings.TrimSpace(value)
		if value == "" {
			return defaultValue, nil
		}

		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			_, _ = fmt.Fprintln(
				w.errorOutput,
				"Enter a whole number greater than zero.",
			)

			continue
		}

		return parsed, nil
	}
}

func (w *setupWizard) readLine() (
	string,
	error,
) {
	value, err := w.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if errors.Is(err, io.EOF) && value == "" {
		return "", io.EOF
	}

	return strings.TrimRight(
		value,
		"\r\n",
	), nil
}

func (w *setupWizard) readSecret() (
	string,
	error,
) {
	inputFile, isFile := w.input.(*os.File)
	if !isFile ||
		!term.IsTerminal(int(inputFile.Fd())) {
		return w.readLine()
	}

	secret, err := term.ReadPassword(
		int(inputFile.Fd()),
	)

	_, _ = fmt.Fprintln(w.output)

	if err != nil {
		return "", err
	}

	return string(secret), nil
}

func (w *setupWizard) printSummary(
	answers setupAnswers,
) {
	excludedTags := answers.ExcludeTags
	if excludedTags == "" {
		excludedTags = "(none)"
	}

	_, _ = fmt.Fprintln(
		w.output,
		"\nConfiguration summary",
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Configuration file:  %s\n",
		answers.OutputFile,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Proxmox URL:         %s\n",
		answers.ProxmoxBaseURL,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Verify TLS:          %t\n",
		answers.ProxmoxVerifyTLS,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  AdGuard URL:         %s\n",
		answers.AdGuardBaseURL,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  DNS suffix:          %s\n",
		answers.DNSSuffix,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Require running:     %t\n",
		answers.RequireRunning,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Excluded tags:       %s\n",
		excludedTags,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Sync interval:       %d seconds\n",
		answers.SyncIntervalSeconds,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  State file:          %s\n",
		answers.StateFile,
	)

	_, _ = fmt.Fprintf(
		w.output,
		"  Dry run:             %t\n\n",
		answers.DryRun,
	)
}

func writeSetupEnvironmentFile(
	path string,
	answers setupAnswers,
) error {
	parentDirectory := filepath.Dir(path)

	if parentDirectory != "." {
		if err := os.MkdirAll(
			parentDirectory,
			0o700,
		); err != nil {
			return fmt.Errorf(
				"create parent directory: %w",
				err,
			)
		}
	}

	temporaryFile, err := os.CreateTemp(
		parentDirectory,
		".proxmox-adguard-sync-*.tmp",
	)
	if err != nil {
		return fmt.Errorf(
			"create temporary file: %w",
			err,
		)
	}

	temporaryPath := temporaryFile.Name()
	removeTemporaryFile := true

	defer func() {
		_ = temporaryFile.Close()

		if removeTemporaryFile {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporaryFile.Chmod(0o600); err != nil {
		return fmt.Errorf(
			"set temporary file permissions: %w",
			err,
		)
	}

	if _, err := io.WriteString(
		temporaryFile,
		renderSetupEnvironmentFile(answers),
	); err != nil {
		return fmt.Errorf(
			"write temporary file: %w",
			err,
		)
	}

	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf(
			"synchronize temporary file: %w",
			err,
		)
	}

	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf(
			"close temporary file: %w",
			err,
		)
	}

	if err := os.Rename(
		temporaryPath,
		path,
	); err != nil {
		return fmt.Errorf(
			"replace configuration file: %w",
			err,
		)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf(
			"set configuration file permissions: %w",
			err,
		)
	}

	removeTemporaryFile = false

	return nil
}

func renderSetupEnvironmentFile(
	answers setupAnswers,
) string {
	var builder strings.Builder

	builder.WriteString(
		"# Generated by proxmox-adguard-sync setup\n",
	)

	builder.WriteString(
		"# Review this file before disabling dry-run mode.\n\n",
	)

	writeEnvironmentValue(
		&builder,
		"PROXMOX_BASE_URL",
		answers.ProxmoxBaseURL,
	)

	writeEnvironmentValue(
		&builder,
		"PROXMOX_TOKEN_ID",
		answers.ProxmoxTokenID,
	)

	writeEnvironmentValue(
		&builder,
		"PROXMOX_TOKEN_SECRET",
		answers.ProxmoxTokenSecret,
	)

	writeEnvironmentBool(
		&builder,
		"PROXMOX_VERIFY_TLS",
		answers.ProxmoxVerifyTLS,
	)

	builder.WriteString("\n")

	writeEnvironmentValue(
		&builder,
		"ADGUARD_BASE_URL",
		answers.AdGuardBaseURL,
	)

	writeEnvironmentValue(
		&builder,
		"ADGUARD_USERNAME",
		answers.AdGuardUsername,
	)

	writeEnvironmentValue(
		&builder,
		"ADGUARD_PASSWORD",
		answers.AdGuardPassword,
	)

	builder.WriteString("\n")

	writeEnvironmentValue(
		&builder,
		"DNS_SUFFIX",
		answers.DNSSuffix,
	)

	writeEnvironmentValue(
		&builder,
		"STATE_FILE",
		answers.StateFile,
	)

	writeEnvironmentBool(
		&builder,
		"DRY_RUN",
		answers.DryRun,
	)

	builder.WriteString("\n")

	writeEnvironmentValue(
		&builder,
		"FILTER_INCLUDE_TYPES",
		"qemu,lxc",
	)

	writeEnvironmentBool(
		&builder,
		"FILTER_REQUIRE_RUNNING",
		answers.RequireRunning,
	)

	writeEnvironmentValue(
		&builder,
		"FILTER_INCLUDE_TAGS",
		"",
	)

	writeEnvironmentValue(
		&builder,
		"FILTER_EXCLUDE_TAGS",
		answers.ExcludeTags,
	)

	writeEnvironmentValue(
		&builder,
		"FILTER_INCLUDE_NAMES",
		"",
	)

	writeEnvironmentValue(
		&builder,
		"FILTER_EXCLUDE_NAMES",
		"",
	)

	builder.WriteString("\n")

	writeEnvironmentValue(
		&builder,
		"DISCOVERY_VM_ORDER",
		"guest-agent,description,cloudinit",
	)

	writeEnvironmentValue(
		&builder,
		"DISCOVERY_LXC_ORDER",
		"config,description",
	)

	writeEnvironmentValue(
		&builder,
		"DESCRIPTION_IP_KEYS",
		"dns_ip,ip",
	)

	writeEnvironmentValue(
		&builder,
		"DESCRIPTION_NAME_KEYS",
		"dns_name,name",
	)

	builder.WriteString("\n")

	writeEnvironmentInteger(
		&builder,
		"SYNC_INTERVAL_SECONDS",
		answers.SyncIntervalSeconds,
	)

	writeEnvironmentValue(
		&builder,
		"LOG_LEVEL",
		"info",
	)

	writeEnvironmentValue(
		&builder,
		"LOG_FORMAT",
		"text",
	)

	return builder.String()
}

func writeEnvironmentValue(
	builder *strings.Builder,
	name string,
	value string,
) {
	_, _ = fmt.Fprintf(
		builder,
		"%s=%s\n",
		name,
		shellQuote(value),
	)
}

func writeEnvironmentBool(
	builder *strings.Builder,
	name string,
	value bool,
) {
	writeEnvironmentValue(
		builder,
		name,
		strconv.FormatBool(value),
	)
}

func writeEnvironmentInteger(
	builder *strings.Builder,
	name string,
	value int,
) {
	writeEnvironmentValue(
		builder,
		name,
		strconv.Itoa(value),
	)
}

func shellQuote(value string) string {
	return "'" +
		strings.ReplaceAll(
			value,
			"'",
			`'\''`,
		) +
		"'"
}

func shellDisplayPath(path string) string {
	if strings.ContainsAny(
		path,
		" \t'\"",
	) {
		return shellQuote(path)
	}

	return path
}

func normalizeCSVInput(value string) string {
	parts := strings.Split(value, ",")
	normalized := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		part = strings.ToLower(
			strings.TrimSpace(part),
		)

		if part != "" {
			normalized = append(
				normalized,
				part,
			)
		}
	}

	return strings.Join(normalized, ",")
}

func expandHomeDirectory(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(
				home,
				strings.TrimPrefix(path, "~/"),
			)
		}
	}

	return path
}
