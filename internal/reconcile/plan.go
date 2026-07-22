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
	Delete    []adguard.Rewrite
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

	sortRewrites(rewrites)

	return rewrites, nil
}

func BuildPlan(
	desired []adguard.Rewrite,
	current []adguard.Rewrite,
	managed map[string]string,
) Plan {
	currentByDomain := indexRewrites(current)
	desiredByDomain := indexRewrites(desired)

	plan := Plan{}

	for _, wanted := range desiredByDomain {
		existing, exists := currentByDomain[wanted.Domain]
		if !exists {
			plan.Add = append(plan.Add, wanted)
			continue
		}

		if strings.TrimSpace(existing.Answer) ==
			strings.TrimSpace(wanted.Answer) {
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

	for domain, existing := range currentByDomain {
		if _, desiredNow := desiredByDomain[domain]; desiredNow {
			continue
		}

		if _, owned := managed[domain]; owned {
			plan.Delete = append(
				plan.Delete,
				existing,
			)
			continue
		}

		plan.Unmanaged = append(
			plan.Unmanaged,
			existing,
		)
	}

	sortRewrites(plan.Add)
	sortRewrites(plan.Delete)
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

func indexRewrites(
	rewrites []adguard.Rewrite,
) map[string]adguard.Rewrite {
	indexed := make(
		map[string]adguard.Rewrite,
		len(rewrites),
	)

	for _, rewrite := range rewrites {
		domain := normalizeDomain(rewrite.Domain)
		if domain == "" {
			continue
		}

		rewrite.Domain = domain
		rewrite.Answer = strings.TrimSpace(rewrite.Answer)

		if _, exists := indexed[domain]; !exists {
			indexed[domain] = rewrite
		}
	}

	return indexed
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
