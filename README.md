# Proxmox → AdGuard Home DNS Sync

> [!NOTE]
> This repository now uses the Go implementation by default.  
> The original JavaScript version is preserved on the [`legacy-javascript`](https://github.com/ka0sdev/proxmox-adguard-sync/tree/legacy-javascript) branch.

# Proxmox → AdGuard Home DNS Sync

A lightweight Go service that discovers Proxmox VE guests and synchronizes their IPv4 addresses into AdGuard Home DNS rewrites.

The service supports both QEMU/KVM virtual machines and LXC containers, configurable discovery strategies, filtering, safe ownership tracking, dry-run operation, and continuous reconciliation.

> This branch contains the Go rewrite of Proxmox AdGuard Sync. The original JavaScript implementation remains available on the `master` branch while the Go version is developed and validated.

## Features

* Supports QEMU/KVM virtual machines and LXC containers
* Discovers static and dynamic IPv4 addresses
* Configurable discovery order
* QEMU Guest Agent support
* Cloud-init address discovery
* LXC network configuration discovery
* Description-based IP and hostname overrides
* Guest filtering by type, state, name, and Proxmox tags
* AdGuard Home DNS rewrite creation, updates, and deletion
* Dry-run mode enabled by default
* Persistent ownership state
* Protects manually created AdGuard rewrites
* Continuous synchronization loop
* Graceful shutdown on `SIGINT` and `SIGTERM`
* Structured text or JSON logging
* No external Go dependencies

## How it works

Each synchronization cycle performs the following steps:

1. Retrieve the guest inventory from Proxmox VE.
2. Apply the configured guest filters.
3. Resolve an IPv4 address and DNS hostname for each selected guest.
4. Retrieve the existing DNS rewrites from AdGuard Home.
5. Load the application ownership state.
6. Build a reconciliation plan.
7. Add, update, or delete managed rewrites when dry-run mode is disabled.
8. Save the new ownership state after a successful reconciliation.

Example result:

```text
devbox-vm.internal      → 172.20.20.10
nextcloud-vm.internal   → 172.20.20.12
lxc-dns.internal        → 172.20.0.4
lxc-proxy-01.internal   → 172.20.0.8
```

## Safety model

The application does not assume ownership of every DNS rewrite in AdGuard Home.

Managed records are tracked in a local state file. A rewrite is eligible for automatic deletion only when it was previously recorded as owned by this application.

Manually created or otherwise unrelated AdGuard rewrites remain untouched.

Dry-run mode is enabled by default:

```env
DRY_RUN=true
```

When dry-run mode is enabled, the application retrieves data and builds the complete reconciliation plan without modifying AdGuard Home or saving ownership changes.

## Requirements

* Proxmox VE with API access
* AdGuard Home
* Go 1.24 or newer for local builds
* Network access to the Proxmox and AdGuard Home APIs
* QEMU Guest Agent inside virtual machines when dynamic VM address discovery is required

## Repository structure

```text
.
├── cmd/
│   └── proxmox-adguard-sync/
│       └── main.go
├── data/
│   └── .gitkeep
├── internal/
│   ├── adguard/
│   ├── config/
│   ├── discovery/
│   ├── logging/
│   ├── proxmox/
│   ├── reconcile/
│   ├── selection/
│   └── state/
├── .env.example
├── go.mod
├── LICENSE
└── README.md
```

## Installation

### Clone the Go branch

```bash
git clone \
  --branch go-rewrite \
  --single-branch \
  https://github.com/ka0sdev/proxmox-adguard-sync.git

cd proxmox-adguard-sync
```

### Configure the environment

Copy the example configuration:

```bash
cp .env.example .env.local
```

Edit it:

```bash
nano .env.local
```

At minimum, configure:

```env
PROXMOX_BASE_URL=https://proxmox.example.internal:8006/api2/json
PROXMOX_TOKEN_ID=dns-sync@pve!adguard-sync
PROXMOX_TOKEN_SECRET=replace-me
PROXMOX_VERIFY_TLS=true

ADGUARD_BASE_URL=http://adguard.example.internal
ADGUARD_USERNAME=admin
ADGUARD_PASSWORD=replace-me

DNS_SUFFIX=internal
DRY_RUN=true
STATE_FILE=./data/state.json
```

### Build

```bash
go build \
  -o bin/proxmox-adguard-sync \
  ./cmd/proxmox-adguard-sync
```

### Load the environment

The application reads its configuration from environment variables. It does not automatically parse `.env.local`.

For Bash:

```bash
set +H
set -a
source .env.local
set +a
```

`set +H` disables Bash history expansion, which is useful when the Proxmox token ID contains `!`.

### Run

```bash
./bin/proxmox-adguard-sync
```

The first synchronization starts immediately. Further synchronization cycles run according to `SYNC_INTERVAL_SECONDS`.

Stop the process with `Ctrl+C`.

## Development

Format the source:

```bash
gofmt -w .
```

Run static analysis:

```bash
go vet ./...
```

Run the tests:

```bash
go test ./...
```

Build the executable:

```bash
go build \
  -o bin/proxmox-adguard-sync \
  ./cmd/proxmox-adguard-sync
```

Run all validation steps together:

```bash
gofmt -w .
go vet ./...
go test ./...
go build \
  -o bin/proxmox-adguard-sync \
  ./cmd/proxmox-adguard-sync
```

## Configuration

### Runtime

| Variable                |             Default | Description                                     |
| ----------------------- | ------------------: | ----------------------------------------------- |
| `SYNC_INTERVAL_SECONDS` |                `60` | Time between synchronization cycles             |
| `DRY_RUN`               |              `true` | Build and log the plan without applying changes |
| `STATE_FILE`            | `./data/state.json` | Persistent ownership state path                 |

### Logging

| Variable     | Default | Description                                                       |
| ------------ | ------: | ----------------------------------------------------------------- |
| `LOG_LEVEL`  |  `info` | `debug`, `info`, `warn`, or `error`                               |
| `LOG_FORMAT` |  `text` | `text` or `json`                                                  |
| `LOG_JSON`   | `false` | Legacy alias that enables JSON logging when `LOG_FORMAT` is unset |

### DNS

| Variable     |    Default | Description                                 |
| ------------ | ---------: | ------------------------------------------- |
| `DNS_SUFFIX` | `internal` | DNS suffix appended to resolved guest names |

The generated domain follows this format:

```text
<hostname>.<DNS_SUFFIX>
```

Example:

```text
lxc-dns.internal
```

### Proxmox VE

| Variable               | Required | Description                                            |
| ---------------------- | -------: | ------------------------------------------------------ |
| `PROXMOX_BASE_URL`     |      Yes | Proxmox API URL ending in `/api2/json`                 |
| `PROXMOX_TOKEN_ID`     |      Yes | Proxmox API token identifier                           |
| `PROXMOX_TOKEN_SECRET` |      Yes | Proxmox API token secret                               |
| `PROXMOX_VERIFY_TLS`   |       No | Verify the Proxmox TLS certificate; defaults to `true` |

`PROXMOX_URL` is supported as an alias for `PROXMOX_BASE_URL`.

### AdGuard Home

| Variable           | Required | Description           |
| ------------------ | -------: | --------------------- |
| `ADGUARD_BASE_URL` |      Yes | AdGuard Home URL      |
| `ADGUARD_USERNAME` |      Yes | AdGuard Home username |
| `ADGUARD_PASSWORD` |      Yes | AdGuard Home password |

`ADGUARD_URL` is supported as an alias for `ADGUARD_BASE_URL`.

The base URL may include `/control`, but it is not required:

```env
ADGUARD_BASE_URL=http://127.0.0.1:80
```

or:

```env
ADGUARD_BASE_URL=http://127.0.0.1:80/control
```

### Guest filters

| Variable                 |    Default | Description                         |
| ------------------------ | ---------: | ----------------------------------- |
| `FILTER_INCLUDE_TYPES`   | `qemu,lxc` | Guest types to include              |
| `FILTER_REQUIRE_RUNNING` |    `false` | Include only running guests         |
| `FILTER_INCLUDE_TAGS`    |      Empty | Include guests matching these tags  |
| `FILTER_EXCLUDE_TAGS`    |      Empty | Exclude guests matching these tags  |
| `FILTER_INCLUDE_NAMES`   |      Empty | Include guests matching these names |
| `FILTER_EXCLUDE_NAMES`   |      Empty | Exclude guests matching these names |

Comma-separated values are supported:

```env
FILTER_INCLUDE_TYPES=qemu,lxc
FILTER_EXCLUDE_TAGS=no-monitor,disabled
FILTER_REQUIRE_RUNNING=true
```

### Discovery

| Variable                |                             Default | Description                                     |
| ----------------------- | ----------------------------------: | ----------------------------------------------- |
| `DISCOVERY_VM_ORDER`    | `guest-agent,description,cloudinit` | QEMU discovery order                            |
| `DISCOVERY_LXC_ORDER`   |                `config,description` | LXC discovery order                             |
| `DESCRIPTION_IP_KEYS`   |                         `dns_ip,ip` | Description keys accepted as address overrides  |
| `DESCRIPTION_NAME_KEYS` |                     `dns_name,name` | Description keys accepted as hostname overrides |

Legacy aliases are also supported:

```env
VM_DISCOVERY_ORDER=guest-agent,description,cloudinit
LXC_DISCOVERY_ORDER=config,description
```

## Address discovery

### QEMU/KVM virtual machines

The default discovery order is:

1. QEMU Guest Agent
2. Description metadata
3. Cloud-init configuration

```env
DISCOVERY_VM_ORDER=guest-agent,description,cloudinit
```

#### QEMU Guest Agent

The application retrieves network interfaces from the running guest and selects the first usable IPv4 address.

It ignores addresses that are:

* Loopback
* Unspecified
* Multicast
* Link-local
* Invalid
* IPv6

#### Description metadata

A static address can be supplied in the Proxmox guest description:

```text
dns_ip=172.20.20.10
dns_name=devbox
```

#### Cloud-init

The application examines `ipconfig0`, `ipconfig1`, and subsequent cloud-init network configuration fields in numerical order.

Dynamic values such as `dhcp`, `auto`, and `manual` are skipped because they do not provide a fixed address.

### LXC containers

The default discovery order is:

1. LXC network configuration
2. Description metadata

```env
DISCOVERY_LXC_ORDER=config,description
```

The application examines `net0`, `net1`, and subsequent network configuration entries.

Example:

```text
name=eth0,bridge=vmbr0,ip=172.20.0.4/16,gw=172.20.0.1
```

DHCP-only LXC configurations require a description override because the Proxmox LXC configuration does not contain the dynamically assigned address.

## Naming and overrides

By default, the Proxmox guest name becomes the DNS hostname:

```text
lxc-proxy-01 → lxc-proxy-01.internal
```

Description metadata can override the hostname:

```text
dns_name=proxy
dns_ip=172.20.0.8
```

Result:

```text
proxy.internal → 172.20.0.8
```

Hostnames are normalized to lowercase DNS-compatible labels.

## Reconciliation behavior

The reconciliation plan classifies records into five groups:

| Classification | Behavior                                          |
| -------------- | ------------------------------------------------- |
| Add            | Desired rewrite does not exist                    |
| Update         | Desired rewrite exists with a different answer    |
| Delete         | Previously managed rewrite is no longer desired   |
| Unchanged      | Desired rewrite already matches                   |
| Unmanaged      | Existing rewrite is not owned by this application |

Additions are applied first, followed by updates. Deletions are applied last.

If an operation fails, execution stops and the ownership state is not replaced.

## Ownership state

The default state path is:

```text
./data/state.json
```

Example:

```json
{
  "version": 1,
  "updated_at": "2026-07-22T15:58:50Z",
  "records": [
    {
      "domain": "lxc-dns.internal",
      "answer": "172.20.0.4"
    }
  ]
}
```

The state file should be stored persistently and must not be committed to Git.

A missing state file is treated as an empty initial state.

After the first successful live synchronization, the complete desired rewrite set is recorded as managed.

## First live synchronization

Start with:

```env
DRY_RUN=true
```

Review the reconciliation summary:

```text
DNS reconciliation plan complete
```

When the plan is correct, change:

```env
DRY_RUN=false
```

Reload the environment and run the application again.

After a successful live synchronization, confirm that the ownership state was written:

```bash
cat data/state.json
```

Dry-run mode may then be re-enabled while testing configuration changes.

## Proxmox API permissions

Create a dedicated Proxmox user and API token, for example:

```text
User:  dns-sync@pve
Token: adguard-sync
```

The token requires permission to:

* Read cluster guest inventory
* Read QEMU configuration
* Read LXC configuration
* Query the QEMU Guest Agent

For Proxmox VE 9, QEMU Guest Agent access requires the granular Guest Agent audit privilege.

Use the least-privileged role that supports the selected discovery methods.

## Enabling QEMU Guest Agent

Enable the Guest Agent in the Proxmox VM options.

On Debian or Ubuntu guests:

```bash
sudo apt update
sudo apt install -y qemu-guest-agent
sudo systemctl enable --now qemu-guest-agent
```

The VM must also have QEMU Guest Agent enabled in Proxmox.

## Testing DNS

Query a generated record directly against AdGuard Home:

```bash
nslookup lxc-dns.internal 127.0.0.1
```

Using `dig`:

```bash
dig @127.0.0.1 lxc-dns.internal
```

## Graceful shutdown

The application listens for:

* `SIGINT`
* `SIGTERM`

Pressing `Ctrl+C` produces a clean shutdown:

```text
Shutdown signal received
Application stopped
```

## Troubleshooting

### Missing required environment variables

Example:

```text
missing required environment variables
```

Confirm that the environment has been loaded:

```bash
set +H
set -a
source .env.local
set +a
```

### Proxmox authentication failure

Confirm:

* The API URL ends with `/api2/json`
* The token ID includes both the user and token name
* The token secret is correct
* Bash history expansion is disabled when the token contains `!`

### QEMU Guest Agent unavailable

Confirm that:

* QEMU Guest Agent is enabled in Proxmox
* The service is installed and running inside the VM
* The API token has the required Guest Agent audit permission

The resolver will continue to description and cloud-init fallbacks when configured.

### LXC address not discovered

Confirm that the LXC has a static IPv4 address in its Proxmox configuration.

For DHCP-managed containers, add metadata to the description:

```text
dns_ip=172.20.0.10
dns_name=service
```

### AdGuard authentication failure

Confirm:

* The base URL is reachable
* The username and password are correct
* The application can access `/control/rewrite/list`

### Existing records shown as unmanaged

This is expected when a record exists in AdGuard but is absent from the ownership state.

Unmanaged records are deliberately protected from deletion.

### State file permission failure

Confirm that the process can create and write the configured state directory:

```bash
mkdir -p data
chmod 750 data
```

State files are written with restrictive permissions.

## Security

* Use a dedicated Proxmox API token.
* Grant only the permissions required for inventory and discovery.
* Prefer valid TLS certificates.
* Keep `PROXMOX_VERIFY_TLS=true` in production.
* Protect the AdGuard password and Proxmox token secret.
* Do not commit `.env.local` or `data/state.json`.
* Run the application as an unprivileged user.
* Keep dry-run enabled while testing filter or discovery changes.

## Current limitations

* IPv4 only
* LXC DHCP addresses cannot be discovered directly from static Proxmox configuration
* Dynamic QEMU discovery depends on QEMU Guest Agent
* AdGuard Persistent Clients are not currently managed
* Docker packaging and native service definitions are not yet included in the Go branch
* Multiple Proxmox clusters are not currently supported

## JavaScript version

The original Node.js implementation remains available on the `master` branch.

The Go rewrite is being developed on the `go-rewrite` branch and is intended to replace the JavaScript implementation after deployment packaging and final validation are complete.

## License

THE BEER-WARE LICENSE (Revision 42)

As long as you retain this notice, you can do whatever you want with this stuff. If we meet someday, and you think this stuff is worth it, you can buy me a beer in return.

— ka0s
