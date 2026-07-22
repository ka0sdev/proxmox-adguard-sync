package discovery

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
	"github.com/ka0sdev/proxmox-adguard-sync/internal/proxmox"
)

type Source string

const (
	SourceLXCConfig   Source = "lxc_config"
	SourceQEMUAgent   Source = "qemu_guest_agent"
	SourceDescription Source = "description"
)

var ErrEmptyHostname = errors.New(
	"resolved hostname is empty",
)

type ResolvedGuest struct {
	Guest         proxmox.Guest
	Address       netip.Addr
	Hostname      string
	Source        Source
	InterfaceName string
}

type Resolver struct {
	config config.DiscoveryConfig
}

func NewResolver(
	discoveryConfig config.DiscoveryConfig,
) Resolver {
	return Resolver{
		config: discoveryConfig,
	}
}

func (r Resolver) Resolve(
	guest proxmox.Guest,
	guestConfig proxmox.GuestConfig,
) (ResolvedGuest, error) {
	return r.ResolveWithQEMUAgent(
		guest,
		guestConfig,
		nil,
	)
}

func (r Resolver) ResolveWithQEMUAgent(
	guest proxmox.Guest,
	guestConfig proxmox.GuestConfig,
	interfaces []proxmox.QEMUAgentInterface,
) (ResolvedGuest, error) {
	switch guest.Type {
	case proxmox.GuestTypeLXC:
		return r.resolveLXC(
			guest,
			guestConfig,
		)

	case proxmox.GuestTypeQEMU:
		return r.resolveQEMU(
			guest,
			guestConfig,
			interfaces,
		)

	default:
		return ResolvedGuest{}, fmt.Errorf(
			"unsupported guest type %q",
			guest.Type,
		)
	}
}

func (r Resolver) resolveLXC(
	guest proxmox.Guest,
	guestConfig proxmox.GuestConfig,
) (ResolvedGuest, error) {
	description := ParseDescription(
		guestConfig.StringValue("description"),
		r.config.DescriptionIPKeys,
		r.config.DescriptionNameKeys,
	)

	hostname := resolveHostname(
		guest.Name,
		description,
	)

	if err := validateResolvedHostname(hostname); err != nil {
		return ResolvedGuest{}, fmt.Errorf(
			"resolve hostname for LXC VMID %d: %w",
			guest.VMID,
			err,
		)
	}

	for _, source := range r.config.LXCOrder {
		switch source {
		case "config":
			result, found :=
				DiscoverLXCConfigIPv4(guestConfig)

			if !found {
				continue
			}

			return ResolvedGuest{
				Guest:         guest,
				Address:       result.Address,
				Hostname:      hostname,
				Source:        SourceLXCConfig,
				InterfaceName: result.InterfaceName,
			}, nil

		case "description":
			if !description.HasAddress {
				continue
			}

			return ResolvedGuest{
				Guest:    guest,
				Address:  description.Address,
				Hostname: hostname,
				Source:   SourceDescription,
			}, nil
		}
	}

	return ResolvedGuest{}, fmt.Errorf(
		"no IPv4 address discovered for LXC VMID %d",
		guest.VMID,
	)
}

func (r Resolver) resolveQEMU(
	guest proxmox.Guest,
	guestConfig proxmox.GuestConfig,
	interfaces []proxmox.QEMUAgentInterface,
) (ResolvedGuest, error) {
	description := ParseDescription(
		guestConfig.StringValue("description"),
		r.config.DescriptionIPKeys,
		r.config.DescriptionNameKeys,
	)

	hostname := resolveHostname(
		guest.Name,
		description,
	)

	if err := validateResolvedHostname(hostname); err != nil {
		return ResolvedGuest{}, fmt.Errorf(
			"resolve hostname for QEMU VMID %d: %w",
			guest.VMID,
			err,
		)
	}

	for _, source := range r.config.QEMUOrder {
		switch source {
		case "guest-agent":
			result, found :=
				DiscoverQEMUAgentIPv4(interfaces)

			if !found {
				continue
			}

			return ResolvedGuest{
				Guest:         guest,
				Address:       result.Address,
				Hostname:      hostname,
				Source:        SourceQEMUAgent,
				InterfaceName: result.InterfaceName,
			}, nil

		case "description":
			if !description.HasAddress {
				continue
			}

			return ResolvedGuest{
				Guest:    guest,
				Address:  description.Address,
				Hostname: hostname,
				Source:   SourceDescription,
			}, nil

		case "cloudinit":
			// Cloud-init discovery is added in the next step.
			continue
		}
	}

	return ResolvedGuest{}, fmt.Errorf(
		"no IPv4 address discovered for QEMU VMID %d",
		guest.VMID,
	)
}

func resolveHostname(
	guestName string,
	description DescriptionResult,
) string {
	if description.HasName {
		return normalizeHostname(description.Name)
	}

	return normalizeHostname(guestName)
}

func validateResolvedHostname(hostname string) error {
	if hostname == "" {
		return ErrEmptyHostname
	}

	return nil
}

func normalizeHostname(value string) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	value = strings.ReplaceAll(
		value,
		"_",
		"-",
	)

	var builder strings.Builder
	builder.Grow(len(value))

	previousHyphen := false

	for _, character := range value {
		isLetter :=
			character >= 'a' &&
				character <= 'z'

		isNumber :=
			character >= '0' &&
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

	return strings.Trim(
		builder.String(),
		"-",
	)
}
