package reconcile

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/discovery"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

func TestBuildDesiredRewrites(t *testing.T) {
	guests := []discovery.ResolvedGuest{
		{
			Guest: proxmox.Guest{
				VMID: 202,
				Name: "lxc-dns",
			},
			Hostname: "lxc-dns",
			Address:  netip.MustParseAddr("172.20.0.4"),
		},
		{
			Guest: proxmox.Guest{
				VMID: 208,
				Name: "lxc-proxy-01",
			},
			Hostname: "Proxy_01",
			Address:  netip.MustParseAddr("172.20.0.8"),
		},
	}

	rewrites, err := BuildDesiredRewrites(
		guests,
		".Internal.",
	)
	if err != nil {
		t.Fatalf(
			"BuildDesiredRewrites() returned an unexpected error: %v",
			err,
		)
	}

	if len(rewrites) != 2 {
		t.Fatalf(
			"len(rewrites) = %d, expected 2",
			len(rewrites),
		)
	}

	if rewrites[0].Domain != "lxc-dns.internal" {
		t.Errorf(
			"rewrites[0].Domain = %q",
			rewrites[0].Domain,
		)
	}

	if rewrites[1].Domain != "proxy-01.internal" {
		t.Errorf(
			"rewrites[1].Domain = %q",
			rewrites[1].Domain,
		)
	}
}

func TestBuildDesiredRewritesRejectsDuplicates(
	t *testing.T,
) {
	guests := []discovery.ResolvedGuest{
		{
			Guest:    proxmox.Guest{VMID: 201},
			Hostname: "duplicate",
			Address:  netip.MustParseAddr("172.20.0.3"),
		},
		{
			Guest:    proxmox.Guest{VMID: 202},
			Hostname: "duplicate",
			Address:  netip.MustParseAddr("172.20.0.4"),
		},
	}

	_, err := BuildDesiredRewrites(
		guests,
		"internal",
	)
	if err == nil {
		t.Fatal(
			"BuildDesiredRewrites() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"duplicate desired DNS domain",
	) {
		t.Errorf(
			"error = %q, expected duplicate-domain error",
			err,
		)
	}
}

func TestBuildPlan(t *testing.T) {
	desired := []adguard.Rewrite{
		{
			Domain: "new.internal",
			Answer: "172.20.0.10",
		},
		{
			Domain: "changed.internal",
			Answer: "172.20.0.20",
		},
		{
			Domain: "same.internal",
			Answer: "172.20.0.30",
		},
	}

	current := []adguard.Rewrite{
		{
			Domain: "changed.internal",
			Answer: "172.20.0.99",
		},
		{
			Domain: "same.internal",
			Answer: "172.20.0.30",
		},
		{
			Domain: "manual.internal",
			Answer: "172.20.0.40",
		},
	}

	plan := BuildPlan(desired, current)

	if len(plan.Add) != 1 {
		t.Errorf(
			"len(Add) = %d, expected 1",
			len(plan.Add),
		)
	}

	if len(plan.Update) != 1 {
		t.Errorf(
			"len(Update) = %d, expected 1",
			len(plan.Update),
		)
	}

	if len(plan.Unchanged) != 1 {
		t.Errorf(
			"len(Unchanged) = %d, expected 1",
			len(plan.Unchanged),
		)
	}

	if len(plan.Unmanaged) != 1 {
		t.Errorf(
			"len(Unmanaged) = %d, expected 1",
			len(plan.Unmanaged),
		)
	}

	if plan.Update[0].Current.Answer !=
		"172.20.0.99" {
		t.Errorf(
			"Current.Answer = %q",
			plan.Update[0].Current.Answer,
		)
	}

	if plan.Update[0].Desired.Answer !=
		"172.20.0.20" {
		t.Errorf(
			"Desired.Answer = %q",
			plan.Update[0].Desired.Answer,
		)
	}

	if plan.Unmanaged[0].Domain !=
		"manual.internal" {
		t.Errorf(
			"Unmanaged[0].Domain = %q",
			plan.Unmanaged[0].Domain,
		)
	}
}
