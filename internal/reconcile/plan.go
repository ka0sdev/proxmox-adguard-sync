package reconcile

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/adguard"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/discovery"
)

type Change struct {
	Current adguard.Rewrite
	Desired adguard.Rewrite
}

type Plan struct {
	Add       []adguard.Rewrite
	Update    []Change
	Unchanged []adguard.Rewrite
	Unmanaged []adguard.Rewrite
}

func BuildDesiredRewrites(
	guests []discovery.ResolvedGuest,
	dnsSuffix string,
) ([]adguard.Rewrite, error) {
	suffix := normalizeDNSSuffix(dnsSuffix)
	if suffix == "" {
		return nil, errors.New(
			"DNS suffix resolves to an empty value",
		)
	}

	rewrites := make(
		[]adguard.Rewrite,
		0,
		len(guests),
	)

	seenDomains := make(map[string]struct{})

	for _, guest := range guests {
		hostname := normalizeDNSLabel(guest.Hostname)
		if hostname == "" {
			return nil, fmt.Errorf(
				"VMID %d has an empty DNS hostname",
				guest.Guest.VMID,
			)
		}

		if !guest.Address.IsValid() ||
			!guest.Address.Is4() {
			return nil, fmt.Errorf(
				"VMID %d has invalid IPv4 address",
				guest.Guest.VMID,
			)
		}

		domain := hostname + "." + suffix

		if _, exists := seenDomains[domain]; exists {
			return nil, fmt.Errorf(
				"duplicate desired DNS domain %q",
				domain,
			)
		}

		seenDomains[domain] = struct{}{}

		rewrites = append(
			rewrites,
			adguard.Rewrite{
				Domain: domain,
				Answer: guest.Address.String(),
			},
		)
	}

	sort.Slice(
		rewrites,
		func(first, second int) bool {
			return rewrites[first].Domain <
				rewrites[second].Domain
		},
	)

	return rewrites, nil
}

func BuildPlan(
	desired []adguard.Rewrite,
	current []adguard.Rewrite,
) Plan {
	currentByDomain := make(
		map[string]adguard.Rewrite,
		len(current),
	)

	desiredDomains := make(
		map[string]struct{},
		len(desired),
	)

	for _, rewrite := range current {
		domain := normalizeDomain(rewrite.Domain)

		if domain == "" {
			continue
		}

		rewrite.Domain = domain

		if _, exists := currentByDomain[domain]; !exists {
			currentByDomain[domain] = rewrite
		}
	}

	plan := Plan{}

	for _, wanted := range desired {
		wanted.Domain = normalizeDomain(wanted.Domain)
		wanted.Answer = strings.TrimSpace(wanted.Answer)

		desiredDomains[wanted.Domain] = struct{}{}

		existing, exists := currentByDomain[wanted.Domain]
		if !exists {
			plan.Add = append(plan.Add, wanted)
			continue
		}

		if strings.TrimSpace(existing.Answer) ==
			wanted.Answer {
			plan.Unchanged = append(
				plan.Unchanged,
				wanted,
			)

			continue
		}

		plan.Update = append(
			plan.Update,
			Change{
				Current: existing,
				Desired: wanted,
			},
		)
	}

	for _, rewrite := range current {
		domain := normalizeDomain(rewrite.Domain)

		if _, managedNow := desiredDomains[domain]; managedNow {
			continue
		}

		rewrite.Domain = domain

		plan.Unmanaged = append(
			plan.Unmanaged,
			rewrite,
		)
	}

	sortRewrites(plan.Add)
	sortRewrites(plan.Unchanged)
	sortRewrites(plan.Unmanaged)

	sort.Slice(
		plan.Update,
		func(first, second int) bool {
			return plan.Update[first].Desired.Domain <
				plan.Update[second].Desired.Domain
		},
	)

	return plan
}

func normalizeDNSSuffix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, ".")

	parts := strings.Split(value, ".")
	normalized := make([]string, 0, len(parts))

	for _, part := range parts {
		label := normalizeDNSLabel(part)
		if label != "" {
			normalized = append(normalized, label)
		}
	}

	return strings.Join(normalized, ".")
}

func normalizeDNSLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")

	var builder strings.Builder

	previousHyphen := false

	for _, character := range value {
		isLetter := character >= 'a' &&
			character <= 'z'

		isNumber := character >= '0' &&
			character <= '9'

		if isLetter || isNumber {
			builder.WriteRune(character)
			previousHyphen = false
			continue
		}

		if character == '-' && !previousHyphen {
			builder.WriteRune(character)
			previousHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func normalizeDomain(value string) string {
	return strings.ToLower(
		strings.Trim(
			strings.TrimSpace(value),
			".",
		),
	)
}

func sortRewrites(rewrites []adguard.Rewrite) {
	sort.Slice(
		rewrites,
		func(first, second int) bool {
			return rewrites[first].Domain <
				rewrites[second].Domain
		},
	)
}
