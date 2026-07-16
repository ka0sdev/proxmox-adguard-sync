package discovery

import (
	"net/netip"
	"strings"
)

type DescriptionResult struct {
	Address netip.Addr
	Name    string

	AddressKey string
	NameKey    string

	HasAddress bool
	HasName    bool
}

func ParseDescription(
	description string,
	ipKeys []string,
	nameKeys []string,
) DescriptionResult {
	values := parseDescriptionValues(description)

	result := DescriptionResult{}

	for _, key := range ipKeys {
		normalizedKey := normalizeMetadataKey(key)
		if normalizedKey == "" {
			continue
		}

		rawValue, exists := values[normalizedKey]
		if !exists {
			continue
		}

		address, err := parseIPv4OrPrefix(
			strings.TrimSpace(rawValue),
		)
		if err != nil {
			continue
		}

		result.Address = address
		result.AddressKey = normalizedKey
		result.HasAddress = true

		break
	}

	for _, key := range nameKeys {
		normalizedKey := normalizeMetadataKey(key)
		if normalizedKey == "" {
			continue
		}

		rawValue, exists := values[normalizedKey]
		if !exists {
			continue
		}

		name := strings.TrimSpace(rawValue)
		if name == "" {
			continue
		}

		result.Name = name
		result.NameKey = normalizedKey
		result.HasName = true

		break
	}

	return result
}

func parseDescriptionValues(
	description string,
) map[string]string {
	values := make(map[string]string)

	description = strings.ReplaceAll(
		description,
		"\r\n",
		"\n",
	)
	description = strings.ReplaceAll(
		description,
		"\r",
		"\n",
	)

	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = normalizeMetadataKey(key)
		if key == "" {
			continue
		}

		values[key] = strings.TrimSpace(value)
	}

	return values
}

func normalizeMetadataKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
