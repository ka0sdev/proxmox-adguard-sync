package discovery

import (
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

type LXCResult struct {
	Address       netip.Addr
	InterfaceName string
	RawConfig     string
}

func DiscoverLXCConfigIPv4(
	config proxmox.LXCConfig,
) (LXCResult, bool) {
	interfaceNames := sortedLXCInterfaceNames(config)

	for _, interfaceName := range interfaceNames {
		rawConfig := config.StringValue(interfaceName)

		address, found := ParseNetworkConfigIPv4(rawConfig)
		if !found {
			continue
		}

		return LXCResult{
			Address:       address,
			InterfaceName: interfaceName,
			RawConfig:     rawConfig,
		}, true
	}

	return LXCResult{}, false
}

func ParseNetworkConfigIPv4(
	networkConfig string,
) (netip.Addr, bool) {
	networkConfig = strings.TrimSpace(networkConfig)
	if networkConfig == "" {
		return netip.Addr{}, false
	}

	for _, field := range strings.Split(networkConfig, ",") {
		key, value, found := strings.Cut(field, "=")
		if !found {
			continue
		}

		if !strings.EqualFold(strings.TrimSpace(key), "ip") {
			continue
		}

		rawAddress := strings.TrimSpace(value)

		if strings.EqualFold(rawAddress, "dhcp") ||
			strings.EqualFold(rawAddress, "auto") {
			return netip.Addr{}, false
		}

		address, err := parseIPv4OrPrefix(rawAddress)
		if err != nil {
			return netip.Addr{}, false
		}

		return address, true
	}

	return netip.Addr{}, false
}

func parseIPv4OrPrefix(value string) (netip.Addr, error) {
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Addr{}, err
		}

		address := prefix.Addr()
		if !address.Is4() {
			return netip.Addr{}, &nonIPv4Error{}
		}

		return address, nil
	}

	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, err
	}

	if !address.Is4() {
		return netip.Addr{}, &nonIPv4Error{}
	}

	return address, nil
}

func sortedLXCInterfaceNames(
	config proxmox.LXCConfig,
) []string {
	type networkInterface struct {
		name  string
		index int
	}

	interfaces := make([]networkInterface, 0)

	for key := range config {
		index, valid := networkInterfaceIndex(key)
		if !valid {
			continue
		}

		interfaces = append(
			interfaces,
			networkInterface{
				name:  key,
				index: index,
			},
		)
	}

	sort.Slice(
		interfaces,
		func(first, second int) bool {
			return interfaces[first].index < interfaces[second].index
		},
	)

	names := make([]string, 0, len(interfaces))

	for _, networkInterface := range interfaces {
		names = append(names, networkInterface.name)
	}

	return names
}

func networkInterfaceIndex(value string) (int, bool) {
	value = strings.ToLower(strings.TrimSpace(value))

	if !strings.HasPrefix(value, "net") {
		return 0, false
	}

	indexText := strings.TrimPrefix(value, "net")
	if indexText == "" {
		return 0, false
	}

	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return 0, false
	}

	return index, true
}

type nonIPv4Error struct{}

func (*nonIPv4Error) Error() string {
	return "address is not IPv4"
}
