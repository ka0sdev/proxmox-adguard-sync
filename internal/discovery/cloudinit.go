package discovery

import (
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

type CloudInitResult struct {
	Address    netip.Addr
	ConfigName string
	RawConfig  string
}

func DiscoverCloudInitIPv4(
	guestConfig proxmox.GuestConfig,
) (CloudInitResult, bool) {
	configNames := sortedCloudInitConfigNames(guestConfig)

	for _, configName := range configNames {
		rawConfig := guestConfig.StringValue(configName)

		address, found := ParseCloudInitIPv4(rawConfig)
		if !found {
			continue
		}

		return CloudInitResult{
			Address:    address,
			ConfigName: configName,
			RawConfig:  rawConfig,
		}, true
	}

	return CloudInitResult{}, false
}

func ParseCloudInitIPv4(
	ipConfig string,
) (netip.Addr, bool) {
	ipConfig = strings.TrimSpace(ipConfig)
	if ipConfig == "" {
		return netip.Addr{}, false
	}

	for _, field := range strings.Split(ipConfig, ",") {
		key, value, found := strings.Cut(field, "=")
		if !found {
			continue
		}

		if !strings.EqualFold(
			strings.TrimSpace(key),
			"ip",
		) {
			continue
		}

		rawAddress := strings.TrimSpace(value)

		if strings.EqualFold(rawAddress, "dhcp") ||
			strings.EqualFold(rawAddress, "auto") ||
			strings.EqualFold(rawAddress, "manual") {
			return netip.Addr{}, false
		}

		address, err := parseIPv4OrPrefix(rawAddress)
		if err != nil {
			return netip.Addr{}, false
		}

		if !isUsableIPv4(address) {
			return netip.Addr{}, false
		}

		return address, true
	}

	return netip.Addr{}, false
}

func sortedCloudInitConfigNames(
	guestConfig proxmox.GuestConfig,
) []string {
	type indexedConfig struct {
		name  string
		index int
	}

	configs := make([]indexedConfig, 0)

	for key := range guestConfig {
		index, valid := cloudInitConfigIndex(key)
		if !valid {
			continue
		}

		configs = append(
			configs,
			indexedConfig{
				name:  key,
				index: index,
			},
		)
	}

	sort.Slice(
		configs,
		func(first, second int) bool {
			return configs[first].index < configs[second].index
		},
	)

	names := make([]string, 0, len(configs))

	for _, config := range configs {
		names = append(names, config.name)
	}

	return names
}

func cloudInitConfigIndex(value string) (int, bool) {
	value = strings.ToLower(strings.TrimSpace(value))

	if !strings.HasPrefix(value, "ipconfig") {
		return 0, false
	}

	indexText := strings.TrimPrefix(value, "ipconfig")
	if indexText == "" {
		return 0, false
	}

	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return 0, false
	}

	return index, true
}
