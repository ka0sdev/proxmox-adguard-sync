package discovery

import (
	"encoding/json"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestParseCloudInitIPv4(t *testing.T) {
	testCases := []struct {
		name        string
		config      string
		expectedIP  string
		expectFound bool
	}{
		{
			name:        "static IPv4 with gateway",
			config:      "ip=172.20.20.10/24,gw=172.20.20.1",
			expectedIP:  "172.20.20.10",
			expectFound: true,
		},
		{
			name:        "static IPv4 without prefix",
			config:      "ip=172.20.20.10,gw=172.20.20.1",
			expectedIP:  "172.20.20.10",
			expectFound: true,
		},
		{
			name:        "gateway before address",
			config:      "gw=172.20.20.1,ip=172.20.20.10/24",
			expectedIP:  "172.20.20.10",
			expectFound: true,
		},
		{
			name:        "case insensitive key",
			config:      "IP=172.20.20.10/24",
			expectedIP:  "172.20.20.10",
			expectFound: true,
		},
		{
			name:        "DHCP",
			config:      "ip=dhcp",
			expectFound: false,
		},
		{
			name:        "automatic",
			config:      "ip=auto",
			expectFound: false,
		},
		{
			name:        "manual",
			config:      "ip=manual",
			expectFound: false,
		},
		{
			name:        "IPv6",
			config:      "ip=2001:db8::10/64",
			expectFound: false,
		},
		{
			name:        "loopback",
			config:      "ip=127.0.0.1/8",
			expectFound: false,
		},
		{
			name:        "link local",
			config:      "ip=169.254.10.20/16",
			expectFound: false,
		},
		{
			name:        "invalid address",
			config:      "ip=999.20.20.10/24",
			expectFound: false,
		},
		{
			name:        "missing address",
			config:      "gw=172.20.20.1",
			expectFound: false,
		},
		{
			name:        "empty configuration",
			config:      "",
			expectFound: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			address, found := ParseCloudInitIPv4(
				testCase.config,
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

func TestDiscoverCloudInitIPv4UsesLowestIndex(
	t *testing.T,
) {
	guestConfig := proxmox.GuestConfig{
		"ipconfig10": json.RawMessage(
			`"ip=172.20.20.30/24"`,
		),
		"ipconfig2": json.RawMessage(
			`"ip=172.20.20.20/24"`,
		),
		"ipconfig0": json.RawMessage(
			`"ip=172.20.20.10/24"`,
		),
	}

	result, found := DiscoverCloudInitIPv4(guestConfig)
	if !found {
		t.Fatal(
			"DiscoverCloudInitIPv4() found no address",
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.ConfigName != "ipconfig0" {
		t.Errorf(
			"ConfigName = %q, expected %q",
			result.ConfigName,
			"ipconfig0",
		)
	}
}

func TestDiscoverCloudInitIPv4SkipsDynamicConfig(
	t *testing.T,
) {
	guestConfig := proxmox.GuestConfig{
		"ipconfig0": json.RawMessage(
			`"ip=dhcp"`,
		),
		"ipconfig1": json.RawMessage(
			`"ip=172.20.20.11/24,gw=172.20.20.1"`,
		),
	}

	result, found := DiscoverCloudInitIPv4(guestConfig)
	if !found {
		t.Fatal(
			"DiscoverCloudInitIPv4() found no address",
		)
	}

	if result.Address.String() != "172.20.20.11" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.11",
		)
	}

	if result.ConfigName != "ipconfig1" {
		t.Errorf(
			"ConfigName = %q, expected %q",
			result.ConfigName,
			"ipconfig1",
		)
	}
}

func TestDiscoverCloudInitIPv4IgnoresUnrelatedFields(
	t *testing.T,
) {
	guestConfig := proxmox.GuestConfig{
		"name": json.RawMessage(
			`"devbox-vm"`,
		),
		"description": json.RawMessage(
			`"ip=172.20.20.99"`,
		),
		"net0": json.RawMessage(
			`"virtio=BC:24:11:AA:BB:CC,bridge=vmbr0"`,
		),
	}

	_, found := DiscoverCloudInitIPv4(guestConfig)
	if found {
		t.Fatal(
			"DiscoverCloudInitIPv4() found unexpected address",
		)
	}
}
