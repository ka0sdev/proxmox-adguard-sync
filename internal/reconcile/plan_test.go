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
			"rewrites[0].Domain = %q, expected %q",
			rewrites[0].Domain,
			"lxc-dns.internal",
		)
	}

	if rewrites[0].Answer != "172.20.0.4" {
		t.Errorf(
			"rewrites[0].Answer = %q, expected %q",
			rewrites[0].Answer,
			"172.20.0.4",
		)
	}

	if rewrites[1].Domain != "proxy-01.internal" {
		t.Errorf(
			"rewrites[1].Domain = %q, expected %q",
			rewrites[1].Domain,
			"proxy-01.internal",
		)
	}

	if rewrites[1].Answer != "172.20.0.8" {
		t.Errorf(
			"rewrites[1].Answer = %q, expected %q",
			rewrites[1].Answer,
			"172.20.0.8",
		)
	}
}

func TestBuildDesiredRewritesRejectsDuplicates(
	t *testing.T,
) {
	guests := []discovery.ResolvedGuest{
		{
			Guest: proxmox.Guest{
				VMID: 201,
			},
			Hostname: "duplicate",
			Address:  netip.MustParseAddr("172.20.0.3"),
		},
		{
			Guest: proxmox.Guest{
				VMID: 202,
			},
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

func TestBuildDesiredRewritesRejectsEmptySuffix(
	t *testing.T,
) {
	_, err := BuildDesiredRewrites(
		nil,
		"...",
	)
	if err == nil {
		t.Fatal(
			"BuildDesiredRewrites() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"DNS suffix resolves to an empty value",
	) {
		t.Errorf(
			"error = %q, expected empty-suffix error",
			err,
		)
	}
}

func TestBuildDesiredRewritesRejectsEmptyHostname(
	t *testing.T,
) {
	guests := []discovery.ResolvedGuest{
		{
			Guest: proxmox.Guest{
				VMID: 202,
			},
			Hostname: "!!!",
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
		"VMID 202 has an empty DNS hostname",
	) {
		t.Errorf(
			"error = %q, expected empty-hostname error",
			err,
		)
	}
}

func TestBuildDesiredRewritesRejectsInvalidAddress(
	t *testing.T,
) {
	guests := []discovery.ResolvedGuest{
		{
			Guest: proxmox.Guest{
				VMID: 202,
			},
			Hostname: "lxc-dns",
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
		"VMID 202 has invalid IPv4 address",
	) {
		t.Errorf(
			"error = %q, expected invalid-address error",
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
			Domain: "stale.internal",
			Answer: "172.20.0.50",
		},
		{
			Domain: "manual.internal",
			Answer: "172.20.0.40",
		},
	}

	managed := map[string]string{
		"stale.internal": "172.20.0.50",
	}

	plan := BuildPlan(
		desired,
		current,
		managed,
	)

	if len(plan.Add) != 1 {
		t.Errorf(
			"len(Add) = %d, expected 1",
			len(plan.Add),
		)
	}

	if plan.Add[0].Domain != "new.internal" {
		t.Errorf(
			"Add[0].Domain = %q, expected %q",
			plan.Add[0].Domain,
			"new.internal",
		)
	}

	if len(plan.Update) != 1 {
		t.Errorf(
			"len(Update) = %d, expected 1",
			len(plan.Update),
		)
	}

	if plan.Update[0].Current.Answer !=
		"172.20.0.99" {
		t.Errorf(
			"Current.Answer = %q, expected %q",
			plan.Update[0].Current.Answer,
			"172.20.0.99",
		)
	}

	if plan.Update[0].Desired.Answer !=
		"172.20.0.20" {
		t.Errorf(
			"Desired.Answer = %q, expected %q",
			plan.Update[0].Desired.Answer,
			"172.20.0.20",
		)
	}

	if len(plan.Delete) != 1 {
		t.Errorf(
			"len(Delete) = %d, expected 1",
			len(plan.Delete),
		)
	}

	if plan.Delete[0].Domain != "stale.internal" {
		t.Errorf(
			"Delete[0].Domain = %q, expected %q",
			plan.Delete[0].Domain,
			"stale.internal",
		)
	}

	if len(plan.Unchanged) != 1 {
		t.Errorf(
			"len(Unchanged) = %d, expected 1",
			len(plan.Unchanged),
		)
	}

	if plan.Unchanged[0].Domain != "same.internal" {
		t.Errorf(
			"Unchanged[0].Domain = %q, expected %q",
			plan.Unchanged[0].Domain,
			"same.internal",
		)
	}

	if len(plan.Unmanaged) != 1 {
		t.Errorf(
			"len(Unmanaged) = %d, expected 1",
			len(plan.Unmanaged),
		)
	}

	if plan.Unmanaged[0].Domain !=
		"manual.internal" {
		t.Errorf(
			"Unmanaged[0].Domain = %q, expected %q",
			plan.Unmanaged[0].Domain,
			"manual.internal",
		)
	}
}

func TestBuildPlanDoesNotDeleteManagedRecordAlreadyAbsent(
	t *testing.T,
) {
	plan := BuildPlan(
		nil,
		nil,
		map[string]string{
			"missing.internal": "172.20.0.50",
		},
	)

	if len(plan.Delete) != 0 {
		t.Errorf(
			"len(Delete) = %d, expected 0",
			len(plan.Delete),
		)
	}
}

func TestBuildPlanTreatsExistingDesiredRecordAsManagedNow(
	t *testing.T,
) {
	desired := []adguard.Rewrite{
		{
			Domain: "service.internal",
			Answer: "172.20.0.10",
		},
	}

	current := []adguard.Rewrite{
		{
			Domain: "service.internal",
			Answer: "172.20.0.10",
		},
	}

	plan := BuildPlan(
		desired,
		current,
		map[string]string{},
	)

	if len(plan.Unchanged) != 1 {
		t.Errorf(
			"len(Unchanged) = %d, expected 1",
			len(plan.Unchanged),
		)
	}

	if len(plan.Unmanaged) != 0 {
		t.Errorf(
			"len(Unmanaged) = %d, expected 0",
			len(plan.Unmanaged),
		)
	}
}

func TestBuildPlanNormalizesDomains(t *testing.T) {
	desired := []adguard.Rewrite{
		{
			Domain: "SERVICE.INTERNAL.",
			Answer: "172.20.0.10",
		},
	}

	current := []adguard.Rewrite{
		{
			Domain: "service.internal",
			Answer: "172.20.0.10",
		},
	}

	plan := BuildPlan(
		desired,
		current,
		nil,
	)

	if len(plan.Unchanged) != 1 {
		t.Errorf(
			"len(Unchanged) = %d, expected 1",
			len(plan.Unchanged),
		)
	}

	if len(plan.Add) != 0 {
		t.Errorf(
			"len(Add) = %d, expected 0",
			len(plan.Add),
		)
	}
}
