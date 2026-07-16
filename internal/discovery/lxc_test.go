package discovery

import (
	"encoding/json"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestParseNetworkConfigIPv4(t *testing.T) {
	testCases := []struct {
		name        string
		network     string
		expectedIP  string
		expectFound bool
	}{
		{
			name: "static address with CIDR",
			network: "name=eth0,bridge=vmbr2," +
				"ip=172.20.0.4/16,type=veth",
			expectedIP:  "172.20.0.4",
			expectFound: true,
		},
		{
			name: "static address without CIDR",
			network: "name=eth0,bridge=vmbr2," +
				"ip=172.20.0.4,type=veth",
			expectedIP:  "172.20.0.4",
			expectFound: true,
		},
		{
			name: "case insensitive IP field",
			network: "name=eth0,IP=172.20.0.4/16," +
				"bridge=vmbr2",
			expectedIP:  "172.20.0.4",
			expectFound: true,
		},
		{
			name:        "DHCP address",
			network:     "name=eth0,ip=dhcp,bridge=vmbr2",
			expectFound: false,
		},
		{
			name:        "automatic address",
			network:     "name=eth0,ip=auto,bridge=vmbr2",
			expectFound: false,
		},
		{
			name:        "missing IP field",
			network:     "name=eth0,bridge=vmbr2,type=veth",
			expectFound: false,
		},
		{
			name:        "invalid address",
			network:     "name=eth0,ip=999.20.0.4/16",
			expectFound: false,
		},
		{
			name:        "IPv6 address",
			network:     "name=eth0,ip=2001:db8::1/64",
			expectFound: false,
		},
		{
			name:        "empty configuration",
			network:     "",
			expectFound: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			address, found := ParseNetworkConfigIPv4(
				testCase.network,
			)

			if found != testCase.expectFound {
				t.Fatalf(
					"found = %t, expected %t",
					found,
					testCase.expectFound,
				)
			}

			if !testCase.expectFound {
				return
			}

			if address.String() != testCase.expectedIP {
				t.Errorf(
					"address = %q, expected %q",
					address,
					testCase.expectedIP,
				)
			}
		})
	}
}

func TestDiscoverLXCConfigIPv4UsesLowestInterfaceIndex(
	t *testing.T,
) {
	config := proxmox.LXCConfig{
		"net10": json.RawMessage(
			`"name=eth10,ip=172.20.10.10/16"`,
		),
		"net2": json.RawMessage(
			`"name=eth2,ip=172.20.10.2/16"`,
		),
		"net0": json.RawMessage(
			`"name=eth0,ip=172.20.10.1/16"`,
		),
	}

	result, found := DiscoverLXCConfigIPv4(config)
	if !found {
		t.Fatal(
			"DiscoverLXCConfigIPv4() found no address",
		)
	}

	if result.Address.String() != "172.20.10.1" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.10.1",
		)
	}

	if result.InterfaceName != "net0" {
		t.Errorf(
			"InterfaceName = %q, expected %q",
			result.InterfaceName,
			"net0",
		)
	}
}

func TestDiscoverLXCConfigIPv4SkipsDynamicInterface(
	t *testing.T,
) {
	config := proxmox.LXCConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=dhcp,bridge=vmbr0"`,
		),
		"net1": json.RawMessage(
			`"name=eth1,ip=172.20.0.8/16,bridge=vmbr2"`,
		),
	}

	result, found := DiscoverLXCConfigIPv4(config)
	if !found {
		t.Fatal(
			"DiscoverLXCConfigIPv4() found no address",
		)
	}

	if result.Address.String() != "172.20.0.8" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.8",
		)
	}

	if result.InterfaceName != "net1" {
		t.Errorf(
			"InterfaceName = %q, expected %q",
			result.InterfaceName,
			"net1",
		)
	}
}

func TestDiscoverLXCConfigIPv4IgnoresUnrelatedFields(
	t *testing.T,
) {
	config := proxmox.LXCConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.0.9"`,
		),
		"hostname": json.RawMessage(
			`"lxc-test"`,
		),
		"memory": json.RawMessage(`2048`),
	}

	_, found := DiscoverLXCConfigIPv4(config)
	if found {
		t.Fatal(
			"DiscoverLXCConfigIPv4() found unexpected address",
		)
	}
}
