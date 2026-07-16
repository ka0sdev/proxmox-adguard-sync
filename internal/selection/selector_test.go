package selection

import (
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestSelectorIncludesValidGuest(t *testing.T) {
	selector := New(config.FilterConfig{
		IncludeTypes: []string{"qemu", "lxc"},
	})

	result := selector.Evaluate(proxmox.Guest{
		VMID:   202,
		Name:   "lxc-dns",
		Node:   "pm",
		Type:   proxmox.GuestTypeLXC,
		Status: "running",
		Tags:   "dns;infrastructure;lxc",
	})

	if !result.Included {
		t.Fatalf(
			"Evaluate() excluded guest with reason %q",
			result.Reason,
		)
	}
}

func TestSelectorExclusionRules(t *testing.T) {
	testCases := []struct {
		name     string
		filters  config.FilterConfig
		guest    proxmox.Guest
		expected ExclusionReason
	}{
		{
			name: "unsupported type",
			filters: config.FilterConfig{
				IncludeTypes: []string{"lxc"},
			},
			guest: proxmox.Guest{
				VMID:   101,
				Name:   "devbox-vm",
				Node:   "pm",
				Type:   proxmox.GuestTypeQEMU,
				Status: "running",
			},
			expected: ReasonUnsupportedType,
		},
		{
			name: "template",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
			},
			guest: proxmox.Guest{
				VMID:     9000,
				Name:     "ubuntu-template",
				Node:     "pm",
				Type:     proxmox.GuestTypeQEMU,
				Status:   "stopped",
				Template: 1,
			},
			expected: ReasonTemplate,
		},
		{
			name: "missing name",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
			},
			guest: proxmox.Guest{
				VMID:   202,
				Node:   "pm",
				Type:   proxmox.GuestTypeLXC,
				Status: "running",
			},
			expected: ReasonMissingIdentity,
		},
		{
			name: "not running",
			filters: config.FilterConfig{
				IncludeTypes:   []string{"qemu", "lxc"},
				RequireRunning: true,
			},
			guest: proxmox.Guest{
				VMID:   103,
				Name:   "workbench-vm",
				Node:   "pm",
				Type:   proxmox.GuestTypeQEMU,
				Status: "stopped",
			},
			expected: ReasonNotRunning,
		},
		{
			name: "name not included",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
				IncludeNames: []string{"dns"},
			},
			guest: proxmox.Guest{
				VMID:   201,
				Name:   "lxc-pulse",
				Node:   "pm",
				Type:   proxmox.GuestTypeLXC,
				Status: "running",
			},
			expected: ReasonNameNotIncluded,
		},
		{
			name: "name excluded",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
				ExcludeNames: []string{"workbench"},
			},
			guest: proxmox.Guest{
				VMID:   103,
				Name:   "workbench-vm",
				Node:   "pm",
				Type:   proxmox.GuestTypeQEMU,
				Status: "running",
			},
			expected: ReasonNameExcluded,
		},
		{
			name: "tag not included",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
				IncludeTags:  []string{"critical"},
			},
			guest: proxmox.Guest{
				VMID:   201,
				Name:   "lxc-pulse",
				Node:   "pm",
				Type:   proxmox.GuestTypeLXC,
				Status: "running",
				Tags:   "infrastructure;monitoring",
			},
			expected: ReasonTagNotIncluded,
		},
		{
			name: "tag excluded",
			filters: config.FilterConfig{
				IncludeTypes: []string{"qemu", "lxc"},
				ExcludeTags:  []string{"no-monitor"},
			},
			guest: proxmox.Guest{
				VMID:   205,
				Name:   "lxc-apps-01",
				Node:   "pm",
				Type:   proxmox.GuestTypeLXC,
				Status: "running",
				Tags:   "lxc;no-monitor",
			},
			expected: ReasonTagExcluded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			selector := New(testCase.filters)

			result := selector.Evaluate(testCase.guest)

			if result.Included {
				t.Fatal(
					"Evaluate() included guest, expected exclusion",
				)
			}

			if result.Reason != testCase.expected {
				t.Errorf(
					"Reason = %q, expected %q",
					result.Reason,
					testCase.expected,
				)
			}
		})
	}
}

func TestSelectorUsesCaseInsensitiveNameSubstringMatching(
	t *testing.T,
) {
	selector := New(config.FilterConfig{
		IncludeTypes: []string{"qemu", "lxc"},
		IncludeNames: []string{"DNS"},
	})

	result := selector.Evaluate(proxmox.Guest{
		VMID:   202,
		Name:   "lxc-dns",
		Node:   "pm",
		Type:   proxmox.GuestTypeLXC,
		Status: "running",
	})

	if !result.Included {
		t.Fatalf(
			"Evaluate() excluded guest with reason %q",
			result.Reason,
		)
	}
}

func TestSelectorUsesCaseInsensitiveExactTagMatching(
	t *testing.T,
) {
	selector := New(config.FilterConfig{
		IncludeTypes: []string{"qemu", "lxc"},
		IncludeTags:  []string{"INFRASTRUCTURE"},
	})

	result := selector.Evaluate(proxmox.Guest{
		VMID:   202,
		Name:   "lxc-dns",
		Node:   "pm",
		Type:   proxmox.GuestTypeLXC,
		Status: "running",
		Tags:   "dns;infrastructure;lxc",
	})

	if !result.Included {
		t.Fatalf(
			"Evaluate() excluded guest with reason %q",
			result.Reason,
		)
	}
}

func TestSelectorDoesNotUsePartialTagMatching(t *testing.T) {
	selector := New(config.FilterConfig{
		IncludeTypes: []string{"qemu", "lxc"},
		IncludeTags:  []string{"infra"},
	})

	result := selector.Evaluate(proxmox.Guest{
		VMID:   202,
		Name:   "lxc-dns",
		Node:   "pm",
		Type:   proxmox.GuestTypeLXC,
		Status: "running",
		Tags:   "infrastructure",
	})

	if result.Included {
		t.Fatal(
			"Evaluate() included partial tag match, expected exclusion",
		)
	}

	if result.Reason != ReasonTagNotIncluded {
		t.Errorf(
			"Reason = %q, expected %q",
			result.Reason,
			ReasonTagNotIncluded,
		)
	}
}
