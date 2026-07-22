package discovery

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestResolverPrefersLXCConfig(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		LXCOrder:            []string{"config", "description"},
		DescriptionIPKeys:   []string{"dns_ip", "ip"},
		DescriptionNameKeys: []string{"dns_name", "name"},
	})

	guest := proxmox.Guest{
		VMID: 202,
		Name: "lxc-dns",
		Node: "pm",
		Type: proxmox.GuestTypeLXC,
	}

	guestConfig := proxmox.GuestConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=172.20.0.4/16"`,
		),
		"description": json.RawMessage(
			`"dns_ip=172.20.0.99\ndns_name=dns-server"`,
		),
	}

	result, err := resolver.Resolve(guest, guestConfig)
	if err != nil {
		t.Fatalf(
			"Resolve() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.0.4" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.4",
		)
	}

	if result.Hostname != "dns-server" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"dns-server",
		)
	}

	if result.Source != SourceLXCConfig {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceLXCConfig,
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

func TestResolverUsesLXCDescriptionFallback(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		LXCOrder:            []string{"config", "description"},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 208,
		Name: "lxc-proxy-01",
		Node: "pm",
		Type: proxmox.GuestTypeLXC,
	}

	guestConfig := proxmox.GuestConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=dhcp"`,
		),
		"description": json.RawMessage(
			`"dns_ip=172.20.0.8\ndns_name=proxy"`,
		),
	}

	result, err := resolver.Resolve(guest, guestConfig)
	if err != nil {
		t.Fatalf(
			"Resolve() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.0.8" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.8",
		)
	}

	if result.Hostname != "proxy" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"proxy",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}

	if result.InterfaceName != "" {
		t.Errorf(
			"InterfaceName = %q, expected empty string",
			result.InterfaceName,
		)
	}
}

func TestResolverSupportsLXCDescriptionFirst(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		LXCOrder:            []string{"description", "config"},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 202,
		Name: "lxc-dns",
		Node: "pm",
		Type: proxmox.GuestTypeLXC,
	}

	guestConfig := proxmox.GuestConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=172.20.0.4/16"`,
		),
		"description": json.RawMessage(
			`"dns_ip=172.20.0.99"`,
		),
	}

	result, err := resolver.Resolve(guest, guestConfig)
	if err != nil {
		t.Fatalf(
			"Resolve() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.0.99" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.99",
		)
	}

	if result.Hostname != "lxc-dns" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"lxc-dns",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}
}

func TestResolverUsesDescriptionForQEMU(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.10\ndns_name=devbox"`,
		),
	}

	result, err := resolver.Resolve(guest, guestConfig)
	if err != nil {
		t.Fatalf(
			"Resolve() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.Hostname != "devbox" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"devbox",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}
}

func TestResolverUsesQEMUAgentFirst(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.99\ndns_name=devbox"`,
		),
	}

	interfaces := []proxmox.QEMUAgentInterface{
		{
			Name: "ens18",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "ipv4",
					Address: "172.20.20.10",
					Prefix:  24,
				},
			},
		},
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		interfaces,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.Hostname != "devbox" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"devbox",
		)
	}

	if result.Source != SourceQEMUAgent {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceQEMUAgent,
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

func TestResolverUsesDescriptionWhenAgentHasNoAddress(
	t *testing.T,
) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.10"`,
		),
	}

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
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		interfaces,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}
}

func TestResolverSupportsDescriptionBeforeAgent(
	t *testing.T,
) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"description",
			"guest-agent",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.99"`,
		),
	}

	interfaces := []proxmox.QEMUAgentInterface{
		{
			Name: "ens18",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "ipv4",
					Address: "172.20.20.10",
					Prefix:  24,
				},
			},
		},
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		interfaces,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.99" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.99",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}

	if result.InterfaceName != "" {
		t.Errorf(
			"InterfaceName = %q, expected empty string",
			result.InterfaceName,
		)
	}
}

func TestResolverNormalizesHostname(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase",
			input:    "LXC-DNS",
			expected: "lxc-dns",
		},
		{
			name:     "underscore",
			input:    "Database_Server",
			expected: "database-server",
		},
		{
			name:     "surrounding whitespace",
			input:    "  Proxy-01  ",
			expected: "proxy-01",
		},
		{
			name:     "repeated hyphens",
			input:    "--Invalid--Name--",
			expected: "invalid-name",
		},
		{
			name:     "unsupported characters",
			input:    "My App!",
			expected: "myapp",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := normalizeHostname(testCase.input)

			if actual != testCase.expected {
				t.Errorf(
					"normalizeHostname(%q) = %q, expected %q",
					testCase.input,
					actual,
					testCase.expected,
				)
			}
		})
	}
}

func TestResolverReturnsErrorWithoutLXCAddress(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		LXCOrder:            []string{"config", "description"},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 202,
		Name: "lxc-dns",
		Node: "pm",
		Type: proxmox.GuestTypeLXC,
	}

	_, err := resolver.Resolve(
		guest,
		proxmox.GuestConfig{},
	)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	if !strings.Contains(
		err.Error(),
		"no IPv4 address discovered for LXC VMID 202",
	) {
		t.Errorf(
			"error = %q, expected missing LXC address error",
			err,
		)
	}
}

func TestResolverReturnsErrorWithoutQEMUAddress(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	_, err := resolver.ResolveWithQEMUAgent(
		guest,
		proxmox.GuestConfig{},
		nil,
	)
	if err == nil {
		t.Fatal(
			"ResolveWithQEMUAgent() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"no IPv4 address discovered for QEMU VMID 101",
	) {
		t.Errorf(
			"error = %q, expected missing QEMU address error",
			err,
		)
	}
}

func TestResolverRejectsEmptyHostname(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		LXCOrder:          []string{"config"},
		DescriptionIPKeys: []string{"dns_ip"},
	})

	guest := proxmox.Guest{
		VMID: 202,
		Name: "!!!",
		Node: "pm",
		Type: proxmox.GuestTypeLXC,
	}

	guestConfig := proxmox.GuestConfig{
		"net0": json.RawMessage(
			`"name=eth0,ip=172.20.0.4/16"`,
		),
	}

	_, err := resolver.Resolve(guest, guestConfig)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	if !strings.Contains(
		err.Error(),
		ErrEmptyHostname.Error(),
	) {
		t.Errorf(
			"error = %q, expected empty-hostname error",
			err,
		)
	}
}

func TestResolverRejectsUnsupportedGuestType(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{})

	guest := proxmox.Guest{
		VMID: 999,
		Name: "unknown",
		Node: "pm",
		Type: proxmox.GuestType("unknown"),
	}

	_, err := resolver.Resolve(
		guest,
		proxmox.GuestConfig{},
	)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	if !strings.Contains(
		err.Error(),
		"unsupported guest type",
	) {
		t.Errorf(
			"error = %q, expected unsupported type error",
			err,
		)
	}
}

func TestResolverUsesCloudInitFallback(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"ipconfig0": json.RawMessage(
			`"ip=172.20.20.10/24,gw=172.20.20.1"`,
		),
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.Hostname != "devbox-vm" {
		t.Errorf(
			"Hostname = %q, expected %q",
			result.Hostname,
			"devbox-vm",
		)
	}

	if result.Source != SourceQEMUCloudInit {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceQEMUCloudInit,
		)
	}

	if result.InterfaceName != "ipconfig0" {
		t.Errorf(
			"InterfaceName = %q, expected %q",
			result.InterfaceName,
			"ipconfig0",
		)
	}
}

func TestResolverPrefersDescriptionOverCloudInit(
	t *testing.T,
) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"guest-agent",
			"description",
			"cloudinit",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.99"`,
		),
		"ipconfig0": json.RawMessage(
			`"ip=172.20.20.10/24"`,
		),
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.99" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.99",
		)
	}

	if result.Source != SourceDescription {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceDescription,
		)
	}
}

func TestResolverSupportsCloudInitFirst(t *testing.T) {
	resolver := NewResolver(config.DiscoveryConfig{
		QEMUOrder: []string{
			"cloudinit",
			"guest-agent",
			"description",
		},
		DescriptionIPKeys:   []string{"dns_ip"},
		DescriptionNameKeys: []string{"dns_name"},
	})

	guest := proxmox.Guest{
		VMID: 101,
		Name: "devbox-vm",
		Node: "pm",
		Type: proxmox.GuestTypeQEMU,
	}

	guestConfig := proxmox.GuestConfig{
		"description": json.RawMessage(
			`"dns_ip=172.20.20.99"`,
		),
		"ipconfig0": json.RawMessage(
			`"ip=172.20.20.10/24"`,
		),
	}

	interfaces := []proxmox.QEMUAgentInterface{
		{
			Name: "ens18",
			IPAddresses: []proxmox.QEMUAgentIPAddress{
				{
					Type:    "ipv4",
					Address: "172.20.20.50",
				},
			},
		},
	}

	result, err := resolver.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		interfaces,
	)
	if err != nil {
		t.Fatalf(
			"ResolveWithQEMUAgent() returned an unexpected error: %v",
			err,
		)
	}

	if result.Address.String() != "172.20.20.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.20.10",
		)
	}

	if result.Source != SourceQEMUCloudInit {
		t.Errorf(
			"Source = %q, expected %q",
			result.Source,
			SourceQEMUCloudInit,
		)
	}
}
