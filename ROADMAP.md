# Roadmap

This document outlines the planned development of Proxmox AdGuard Sync.

The roadmap is directional rather than a fixed commitment. Priorities may change based on testing, feedback, upstream API changes, and operational requirements.

## Current status

The Go implementation is currently pre-release software and should not yet be considered production-ready.

The primary focus is validating synchronization safety, improving installation and configuration, and documenting reliable upgrade and recovery procedures.

## v0.1.0 — Initial Go release

### Core synchronization

- [x] Retrieve Proxmox guest inventory
- [x] Support QEMU virtual machines
- [x] Support LXC containers
- [x] Filter guests by type, name, status, and tags
- [x] Discover LXC IPv4 addresses from guest configuration
- [x] Discover QEMU addresses using the Guest Agent
- [x] Support QEMU cloud-init address fallback
- [x] Support description-based DNS metadata
- [x] Generate desired AdGuard Home DNS rewrites
- [x] Detect additions, updates, deletions, unchanged records, and unmanaged records
- [x] Preserve manually managed AdGuard Home records
- [x] Persist ownership state
- [x] Support dry-run mode
- [x] Run synchronization on a recurring interval
- [x] Handle graceful shutdown

### Release engineering

- [x] Build Linux AMD64 releases
- [x] Build Linux ARM64 releases
- [x] Generate SHA-256 checksums
- [x] Embed version, commit, and build metadata
- [x] Validate published release assets
- [x] Validate binary architectures
- [x] Validate release version metadata
- [ ] Add automated validation summary to GitHub Releases
- [ ] Publish installation and upgrade documentation
- [ ] Document rollback procedures

### Configuration and installation

- [ ] Add an interactive configuration wizard
- [ ] Add a non-interactive configuration validation command
- [ ] Test Proxmox connectivity during setup
- [ ] Test AdGuard Home connectivity during setup
- [ ] Generate configuration files with restrictive permissions
- [ ] Add a systemd service example
- [ ] Add a guided native installation process
- [ ] Add a minimal production container image
- [ ] Document Docker Compose deployment

### Reliability and safety

- [ ] Add atomic state-file writes
- [ ] Add state-file corruption detection
- [ ] Add automatic state backups
- [ ] Add recovery documentation
- [ ] Add additional protection against accidental mass deletion
- [ ] Add configurable deletion thresholds
- [ ] Add synchronization locking to prevent overlapping runs
- [ ] Improve handling of temporary Proxmox API failures
- [ ] Improve handling of temporary AdGuard Home API failures
- [ ] Add retry and backoff behavior
- [ ] Add integration tests for synchronization planning
- [ ] Add integration tests for AdGuard mutations
- [ ] Test interrupted synchronization recovery
- [ ] Test upgrade and rollback behavior

### Documentation

- [ ] Add complete configuration reference
- [ ] Add Proxmox API token setup guide
- [ ] Add AdGuard Home credential setup guide
- [ ] Add discovery-order examples
- [ ] Add metadata examples for guest descriptions
- [ ] Add troubleshooting documentation
- [ ] Add migration guide from the legacy JavaScript version
- [ ] Add example configurations

## Future versions

### Observability

- [ ] Add structured synchronization summaries
- [ ] Add optional metrics endpoint
- [ ] Add Prometheus-compatible metrics
- [ ] Add health and readiness checks
- [ ] Add configurable log output destinations
- [ ] Add synchronization duration metrics
- [ ] Add counters for additions, updates, deletions, and failures

### Configuration improvements

- [ ] Support configuration files in addition to environment variables
- [ ] Support explicit per-guest DNS overrides
- [ ] Support multiple DNS suffix rules
- [ ] Support tag-based DNS suffix selection
- [ ] Support custom hostname templates
- [ ] Add configuration schema validation
- [ ] Add configuration migration support

### Proxmox support

- [ ] Test multi-node Proxmox clusters
- [ ] Improve duplicate guest-name handling
- [ ] Support additional network configuration formats
- [ ] Improve IPv6 handling
- [ ] Support configurable network-interface selection
- [ ] Support multiple address-selection strategies

### AdGuard Home support

- [ ] Validate compatibility across supported AdGuard Home versions
- [ ] Improve conflict reporting for manually managed records
- [ ] Add optional record adoption
- [ ] Add export and restore tools for managed records
- [ ] Add support for additional AdGuard Home DNS features where appropriate

### Operations

- [ ] Add service installation command
- [ ] Add service removal command
- [ ] Add upgrade command
- [ ] Add release checksum verification during installation
- [ ] Add shell completion
- [ ] Add package formats where maintainable
- [ ] Evaluate automated update notifications

## Stable release criteria

A stable release will require:

- Successful live synchronization testing across representative QEMU and LXC guests
- Verified protection of manually managed DNS entries
- Verified state recovery after interruption or failure
- Documented installation, upgrade, migration, and rollback procedures
- Reliable release artifact validation
- Configuration validation and connectivity testing
- Sufficient unit and integration test coverage
- No known high-risk deletion or ownership-state issues