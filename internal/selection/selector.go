package selection

import (
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

type ExclusionReason string

const (
	ReasonUnsupportedType ExclusionReason = "unsupported_type"
	ReasonTemplate        ExclusionReason = "template"
	ReasonMissingIdentity ExclusionReason = "missing_identity"
	ReasonNotRunning      ExclusionReason = "not_running"
	ReasonNameNotIncluded ExclusionReason = "name_not_included"
	ReasonNameExcluded    ExclusionReason = "name_excluded"
	ReasonTagNotIncluded  ExclusionReason = "tag_not_included"
	ReasonTagExcluded     ExclusionReason = "tag_excluded"
)

type Result struct {
	Guest     proxmox.Guest
	Included  bool
	Reason    ExclusionReason
	GuestTags []string
}

type Selector struct {
	filters config.FilterConfig
}

func New(filters config.FilterConfig) Selector {
	return Selector{
		filters: filters,
	}
}

func (s Selector) Evaluate(guest proxmox.Guest) Result {
	tags := guest.ParsedTags()

	result := Result{
		Guest:     guest,
		Included:  false,
		GuestTags: tags,
	}

	if !containsFold(
		s.filters.IncludeTypes,
		string(guest.Type),
	) {
		result.Reason = ReasonUnsupportedType
		return result
	}

	if guest.IsTemplate() {
		result.Reason = ReasonTemplate
		return result
	}

	if guest.VMID <= 0 ||
		strings.TrimSpace(guest.Name) == "" ||
		strings.TrimSpace(guest.Node) == "" {
		result.Reason = ReasonMissingIdentity
		return result
	}

	if s.filters.RequireRunning && !guest.IsRunning() {
		result.Reason = ReasonNotRunning
		return result
	}

	if len(s.filters.IncludeNames) > 0 &&
		!matchesAnySubstring(guest.Name, s.filters.IncludeNames) {
		result.Reason = ReasonNameNotIncluded
		return result
	}

	if len(s.filters.ExcludeNames) > 0 &&
		matchesAnySubstring(guest.Name, s.filters.ExcludeNames) {
		result.Reason = ReasonNameExcluded
		return result
	}

	if len(s.filters.IncludeTags) > 0 &&
		!tagsContainAny(tags, s.filters.IncludeTags) {
		result.Reason = ReasonTagNotIncluded
		return result
	}

	if len(s.filters.ExcludeTags) > 0 &&
		tagsContainAny(tags, s.filters.ExcludeTags) {
		result.Reason = ReasonTagExcluded
		return result
	}

	result.Included = true
	result.Reason = ""

	return result
}

func (s Selector) Filter(
	guests []proxmox.Guest,
) ([]proxmox.Guest, []Result) {
	included := make([]proxmox.Guest, 0, len(guests))
	excluded := make([]Result, 0)

	for _, guest := range guests {
		result := s.Evaluate(guest)

		if result.Included {
			included = append(included, guest)
			continue
		}

		excluded = append(excluded, result)
	}

	return included, excluded
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(
			strings.TrimSpace(value),
			strings.TrimSpace(wanted),
		) {
			return true
		}
	}

	return false
}

func matchesAnySubstring(value string, patterns []string) bool {
	normalizedValue := strings.ToLower(value)

	for _, pattern := range patterns {
		normalizedPattern := strings.ToLower(
			strings.TrimSpace(pattern),
		)

		if normalizedPattern != "" &&
			strings.Contains(normalizedValue, normalizedPattern) {
			return true
		}
	}

	return false
}

func tagsContainAny(tags, wantedTags []string) bool {
	for _, tag := range tags {
		if containsFold(wantedTags, tag) {
			return true
		}
	}

	return false
}
