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
	defaultSyncInterval   = 60 * time.Second
	defaultLogLevel       = "info"
	defaultLogFormat      = "text"
	defaultProxmoxTLSMode = true
)

type Config struct {
	Proxmox   ProxmoxConfig
	AdGuard   AdGuardConfig
	Logging   LoggingConfig
	Filters   FilterConfig
	Discovery DiscoveryConfig

	SyncInterval time.Duration
}

type ProxmoxConfig struct {
	BaseURL        string
	APITokenID     string
	APITokenSecret string
	VerifyTLS      bool
}

type AdGuardConfig struct {
	BaseURL  string
	Username string
	Password string
}

type LoggingConfig struct {
	Level  string
	Format string
}

type FilterConfig struct {
	IncludeTypes   []string
	RequireRunning bool
	IncludeTags    []string
	ExcludeTags    []string
	IncludeNames   []string
	ExcludeNames   []string
}

type DiscoveryConfig struct {
	QEMUOrder           []string
	LXCOrder            []string
	DescriptionIPKeys   []string
	DescriptionNameKeys []string
}

func Load() (Config, error) {
	proxmoxVerifyTLS, err := environmentBool(
		"PROXMOX_VERIFY_TLS",
		defaultProxmoxTLSMode,
	)
	if err != nil {
		return Config{}, err
	}

	logFormat, err := loadLogFormat()
	if err != nil {
		return Config{}, err
	}

	requireRunning, err := environmentBool(
		"FILTER_REQUIRE_RUNNING",
		false,
	)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Proxmox: ProxmoxConfig{
			BaseURL: environmentFirst(
				"PROXMOX_BASE_URL",
				"PROXMOX_URL",
			),
			APITokenID: strings.TrimSpace(
				os.Getenv("PROXMOX_TOKEN_ID"),
			),
			APITokenSecret: strings.TrimSpace(
				os.Getenv("PROXMOX_TOKEN_SECRET"),
			),
			VerifyTLS: proxmoxVerifyTLS,
		},
		AdGuard: AdGuardConfig{
			BaseURL: environmentFirst(
				"ADGUARD_BASE_URL",
				"ADGUARD_URL",
			),
			Username: strings.TrimSpace(
				os.Getenv("ADGUARD_USERNAME"),
			),
			Password: strings.TrimSpace(
				os.Getenv("ADGUARD_PASSWORD"),
			),
		},
		Logging: LoggingConfig{
			Level: environmentOrDefault(
				"LOG_LEVEL",
				defaultLogLevel,
			),
			Format: logFormat,
		},
		Filters: FilterConfig{
			IncludeTypes: environmentCSV(
				"FILTER_INCLUDE_TYPES",
				[]string{"qemu", "lxc"},
			),
			RequireRunning: requireRunning,
			IncludeTags: environmentCSV(
				"FILTER_INCLUDE_TAGS",
				nil,
			),
			ExcludeTags: environmentCSV(
				"FILTER_EXCLUDE_TAGS",
				nil,
			),
			IncludeNames: environmentCSV(
				"FILTER_INCLUDE_NAMES",
				nil,
			),
			ExcludeNames: environmentCSV(
				"FILTER_EXCLUDE_NAMES",
				nil,
			),
		},
		Discovery: DiscoveryConfig{
			QEMUOrder: environmentCSVFirst(
				[]string{
					"DISCOVERY_VM_ORDER",
					"VM_DISCOVERY_ORDER",
				},
				[]string{
					"guest-agent",
					"description",
					"cloudinit",
				},
			),
			LXCOrder: environmentCSVFirst(
				[]string{
					"DISCOVERY_LXC_ORDER",
					"LXC_DISCOVERY_ORDER",
				},
				[]string{
					"config",
					"description",
				},
			),
			DescriptionIPKeys: environmentCSV(
				"DESCRIPTION_IP_KEYS",
				[]string{"dns_ip", "ip"},
			),
			DescriptionNameKeys: environmentCSV(
				"DESCRIPTION_NAME_KEYS",
				[]string{"dns_name", "name"},
			),
		},
		SyncInterval: defaultSyncInterval,
	}

	if value := strings.TrimSpace(
		os.Getenv("SYNC_INTERVAL_SECONDS"),
	); value != "" {
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
		missing = append(missing, "PROXMOX_BASE_URL")
	}

	if c.Proxmox.APITokenID == "" {
		missing = append(missing, "PROXMOX_TOKEN_ID")
	}

	if c.Proxmox.APITokenSecret == "" {
		missing = append(missing, "PROXMOX_TOKEN_SECRET")
	}

	if c.AdGuard.BaseURL == "" {
		missing = append(missing, "ADGUARD_BASE_URL")
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

	if !isSupportedLogLevel(c.Logging.Level) {
		return errors.New(
			"LOG_LEVEL must be one of: debug, info, warn, error",
		)
	}

	if !isSupportedLogFormat(c.Logging.Format) {
		return errors.New(
			"LOG_FORMAT must be one of: text, json",
		)
	}

	for _, guestType := range c.Filters.IncludeTypes {
		switch guestType {
		case "qemu", "lxc":
		default:
			return fmt.Errorf(
				"FILTER_INCLUDE_TYPES contains unsupported type %q",
				guestType,
			)
		}
	}

	for _, source := range c.Discovery.QEMUOrder {
		switch source {
		case "guest-agent", "description", "cloudinit":
		default:
			return fmt.Errorf(
				"DISCOVERY_VM_ORDER contains unsupported source %q",
				source,
			)
		}
	}

	for _, source := range c.Discovery.LXCOrder {
		switch source {
		case "config", "description":
		default:
			return fmt.Errorf(
				"DISCOVERY_LXC_ORDER contains unsupported source %q",
				source,
			)
		}
	}

	return nil
}

func loadLogFormat() (string, error) {
	if value := strings.TrimSpace(os.Getenv("LOG_FORMAT")); value != "" {
		return strings.ToLower(value), nil
	}

	logJSON, err := environmentBool("LOG_JSON", false)
	if err != nil {
		return "", err
	}

	if logJSON {
		return "json", nil
	}

	return defaultLogFormat, nil
}

func environmentFirst(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return strings.TrimRight(value, "/")
		}
	}

	return ""
}

func environmentOrDefault(name, defaultValue string) string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if value == "" {
		return defaultValue
	}

	return value
}

func environmentBool(name string, defaultValue bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func environmentCSV(name string, defaultValue []string) []string {
	return environmentCSVFirst(
		[]string{name},
		defaultValue,
	)
}

func environmentCSVFirst(
	names []string,
	defaultValue []string,
) []string {
	var value string

	for _, name := range names {
		value = strings.TrimSpace(os.Getenv(name))
		if value != "" {
			break
		}
	}

	if value == "" {
		return append([]string(nil), defaultValue...)
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		normalized := strings.ToLower(
			strings.TrimSpace(part),
		)

		if normalized != "" {
			values = append(values, normalized)
		}
	}

	return values
}

func isSupportedLogLevel(value string) bool {
	switch value {
	case "debug", "info", "warn", "warning", "error":
		return true
	default:
		return false
	}
}

func isSupportedLogFormat(value string) bool {
	switch value {
	case "text", "json":
		return true
	default:
		return false
	}
}
