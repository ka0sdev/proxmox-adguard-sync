package discovery

import (
	"net/netip"
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

type QEMUAgentResult struct {
	Address       netip.Addr
	InterfaceName string
}

func DiscoverQEMUAgentIPv4(
	interfaces []proxmox.QEMUAgentInterface,
) (QEMUAgentResult, bool) {
	for _, networkInterface := range interfaces {
		if strings.EqualFold(
			strings.TrimSpace(networkInterface.Name),
			"lo",
		) {
			continue
		}

		for _, candidate := range networkInterface.IPAddresses {
			if !isQEMUAgentIPv4Type(candidate.Type) {
				continue
			}

			address, err := netip.ParseAddr(
				strings.TrimSpace(candidate.Address),
			)
			if err != nil {
				continue
			}

			if !isUsableIPv4(address) {
				continue
			}

			return QEMUAgentResult{
				Address:       address,
				InterfaceName: networkInterface.Name,
			}, true
		}
	}

	return QEMUAgentResult{}, false
}

func isQEMUAgentIPv4Type(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ipv4", "inet":
		return true
	default:
		return false
	}
}

func isUsableIPv4(address netip.Addr) bool {
	if !address.IsValid() || !address.Is4() {
		return false
	}

	if address.IsLoopback() ||
		address.IsUnspecified() ||
		address.IsMulticast() ||
		address.IsLinkLocalUnicast() {
		return false
	}

	return true
}
