package discovery

import (
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestDiscoverQEMUAgentIPv4(t *testing.T) {
	interfaces := []proxmox.QEMUAgentInterface{
		{
			Name: "lo",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "ipv4",
					Address: "127.0.0.1",
					Prefix:  8,
				},
			},
		},
		{
			Name: "ens18",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "ipv6",
					Address: "2001:db8::10",
					Prefix:  64,
				},
				{
					Type:    "ipv4",
					Address: "172.20.20.10",
					Prefix:  24,
				},
			},
		},
	}

	result, found := DiscoverQEMUAgentIPv4(interfaces)
	if !found {
		t.Fatal(
			"DiscoverQEMUAgentIPv4() found no address",
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.InterfaceName != "ens18" {
		t.Errorf(
			"InterfaceName = %q, expected %q",
			result.InterfaceName,
			"ens18",
		)
	}
}

func TestDiscoverQEMUAgentIPv4SkipsUnusableAddresses(
	t *testing.T,
) {
	testCases := []struct {
		name    string
		address string
	}{
		{
			name:    "loopback",
			address: "127.0.0.1",
		},
		{
			name:    "unspecified",
			address: "0.0.0.0",
		},
		{
			name:    "link local",
			address: "169.254.10.20",
		},
		{
			name:    "multicast",
			address: "224.0.0.1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			interfaces := []proxmox.QEMUAgentInterface{
				{
					Name: "ens18",
					IPAddresses: []proxmox.QEMUAgentIPAddress{
						{
							Type:    "ipv4",
							Address: testCase.address,
						},
					},
				},
			}

			_, found := DiscoverQEMUAgentIPv4(interfaces)
			if found {
				t.Fatal(
					"DiscoverQEMUAgentIPv4() found unusable address",
				)
			}
		})
	}
}

func TestDiscoverQEMUAgentIPv4SupportsInetType(
	t *testing.T,
) {
	interfaces := []proxmox.QEMUAgentInterface{
		{
			Name: "eth0",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "inet",
					Address: "10.0.0.20",
				},
			},
		},
	}

	result, found := DiscoverQEMUAgentIPv4(interfaces)
	if !found {
		t.Fatal(
			"DiscoverQEMUAgentIPv4() found no address",
		)
	}

	if result.Address.String() != "10.0.0.20" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"10.0.0.20",
		)
	}
}

func TestDiscoverQEMUAgentIPv4ReturnsNoResult(
	t *testing.T,
) {
	_, found := DiscoverQEMUAgentIPv4(nil)
	if found {
		t.Fatal(
			"DiscoverQEMUAgentIPv4(nil) found an address",
		)
	}
}
