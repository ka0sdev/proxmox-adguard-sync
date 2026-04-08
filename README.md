# Proxmox → AdGuard Home DNS Sync

A configurable Node.js service that synchronizes Proxmox guests (LXC + VMs) into AdGuard Home DNS rewrites.

Designed to be environment-agnostic: configure via `.env`, run in Docker, and let it continuously reconcile DNS records.

---

## Table of Contents

* [Overview](#overview)
* [Features](#features)
* [Architecture](#architecture)
* [Requirements](#requirements)
* [Installation](#installation)
* [Configuration (.env)](#configuration-env)
* [How Discovery Works](#how-discovery-works)
* [Naming & Overrides](#naming--overrides)
* [Reconciliation Model](#reconciliation-model)
* [State Management](#state-management)
* [Running the Service](#running-the-service)
* [Testing DNS](#testing-dns)
* [Proxmox API Setup](#proxmox-api-setup)
* [Guest Agent Guidance (VMs)](#guest-agent-guidance-vms)
* [AdGuard Home Integration](#adguard-home-integration)
* [Troubleshooting](#troubleshooting)
* [Security Notes](#security-notes)
* [Recommended Defaults](#recommended-defaults)
* [Limitations](#limitations)
* [License](#license)

---

## Overview

The service:

* Reads Proxmox inventory via API
* Discovers each guest's IP and desired DNS name
* Syncs entries into AdGuard Home DNS rewrites
* Repeats continuously on a configurable interval

Example result:

```text
lxc-pulse.internal   -> 172.20.0.3
edge-vm.internal     -> 172.20.0.2
devbox-vm.internal   -> 172.20.20.10
```

---

## Features

* Supports **LXC** and **QEMU/KVM VMs**
* Works with **static** and **DHCP/dynamic (VM)** setups
* Config-driven behavior via `.env`
* Multiple discovery strategies with configurable order
* Metadata overrides via guest description
* Safe reconciliation (add/update/delete only managed records)
* Dockerized deployment
* Continuous sync loop with healthcheck

---

## Architecture

```text
Proxmox API
    ↓
Sync Service (Node.js)
    ↓
AdGuard Home API
    ↓
DNS Rewrites
```

---

## Requirements

* Proxmox VE with API access
* AdGuard Home
* Docker + Docker Compose plugin
* Network access from the container to Proxmox and AdGuard
* (Recommended) QEMU Guest Agent for VMs

---

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/<your-username>/proxmox-adguard-sync.git
cd proxmox-adguard-sync
```

### 2. Configure environment

```bash
cp .env.example .env
nano .env
```

Set at minimum:

```dotenv
PROXMOX_BASE_URL=...
PROXMOX_TOKEN_ID=...
PROXMOX_TOKEN_SECRET=...

ADGUARD_BASE_URL=...
ADGUARD_USERNAME=...
ADGUARD_PASSWORD=...
```

### 3. Start the service

```bash
docker compose up -d --build
```

View logs:

```bash
docker compose logs -f
```

---

## Configuration (.env)

### General

```dotenv
SYNC_INTERVAL_SECONDS=60
LOG_LEVEL=info
TZ=Europe/Copenhagen
```

### JSON Logging

```dotenv
LOG_JSON=false 
```

Example:

```json
{
  "ts": "2026-04-07T10:00:00Z",
  "level": "info",
  "service": "proxmox-adguard-sync",
  "msg": "Sync completed",
  "meta": { "adds": 5 }
}
```

### DNS

```dotenv
DNS_SUFFIX=internal
```

### Filters

```dotenv
FILTER_INCLUDE_TYPES=qemu,lxc
FILTER_REQUIRE_RUNNING=false
FILTER_INCLUDE_TAGS=
FILTER_EXCLUDE_TAGS=
FILTER_INCLUDE_NAMES=
FILTER_EXCLUDE_NAMES=
```

### Discovery Order

```dotenv
DISCOVERY_VM_ORDER=guest-agent,description,cloudinit
DISCOVERY_LXC_ORDER=config,description
```

### Description Keys

```dotenv
DESCRIPTION_IP_KEYS=dns_ip,ip
DESCRIPTION_NAME_KEYS=dns_name,name
```

### Proxmox

```dotenv
PROXMOX_BASE_URL=https://your-proxmox:8006/api2/json
PROXMOX_TOKEN_ID=...
PROXMOX_TOKEN_SECRET=...
PROXMOX_VERIFY_TLS=false
NODE_TLS_REJECT_UNAUTHORIZED=0
```

### AdGuard Home

```dotenv
ADGUARD_BASE_URL=http://127.0.0.1:3000
ADGUARD_USERNAME=admin
ADGUARD_PASSWORD=...
```

### State

```dotenv
STATE_FILE=/app/data/state.json
```

---

## How Discovery Works

### VMs (default order)

1. QEMU Guest Agent (best for DHCP)
2. Description metadata
3. Cloud-init config

### LXCs (default order)

1. LXC config (`net0`, `net1`, etc.)
2. Description metadata

---

## Naming & Overrides

Default behavior:

```text
<proxmox-name>.<DNS_SUFFIX>
```

Examples:

```text
edge-vm.internal
lxc-pulse.internal
```

### Override via description

```text
dns_name=app
dns_ip=10.0.0.10
```

Result:

```text
app.internal -> 10.0.0.10
```

---

## Reconciliation Model

Each cycle:

1. Fetch guests from Proxmox
2. Apply filters
3. Resolve IP + name
4. Compare with AdGuard rewrites
5. Apply changes

| Action | Behavior            |
| ------ | ------------------- |
| Add    | Missing entry       |
| Update | IP changed          |
| Delete | Stale managed entry |

---

## State Management

File:

```text
data/state.json
```

Used to:

* Track managed entries
* Avoid deleting manual DNS entries
* Ensure safe reconciliation

---

## Running the Service

```bash
docker compose up -d --build
```

Logs:

```bash
docker compose logs -f
```

Healthcheck:

```bash
node src/index.js --healthcheck
```

---

## Testing DNS

```bash
nslookup edge-vm.internal 127.0.0.1
nslookup lxc-pulse.internal 127.0.0.1
```

---

## Proxmox API Setup

Create:

* User: `dns-sync@pve`
* API Token

### Recommended Role

#### Proxmox VE 8

Create custom role with:

* `Sys.Audit`
* `VM.Audit`
* `VM.Monitor`

#### Proxmox VE 9+

Create custom role with:

* `Sys.Audit`
* `VM.Audit` *(deprecated — will be removed in a future release)*
* `VM.GuestAgent.Audit`

> `VM.Monitor` was removed in PVE 9 and replaced with more granular permissions.

### Assign Role

Assign to:

* User
* API Token

Path:

```text
/
```

---

## Guest Agent Guidance (VMs)

Enable in Proxmox:

```text
VM → Options → QEMU Guest Agent → Enabled
```

Inside VM:

```bash
sudo apt install -y qemu-guest-agent
sudo systemctl enable --now qemu-guest-agent
```

---

## AdGuard Home Integration

Uses API endpoints:

* `/control/rewrite/list`
* `/control/rewrite/add`
* `/control/rewrite/delete`

---

## Troubleshooting

### fetch failed

* Check TLS settings
* Verify connectivity

### Permission error

#### PVE 8

```text
VM.Monitor missing
```

Fix: ensure role includes `VM.Monitor`

#### PVE 9+

```text
VM.GuestAgent.Audit missing
```

Fix: ensure role includes:

* `Sys.Audit`
* `VM.Audit`
* `VM.GuestAgent.Audit`

### VM skipped

* Guest agent missing
* No metadata fallback

### LXC skipped

* Using DHCP without metadata

---

## Security Notes

* Use API tokens (not passwords)
* Limit privileges
* Protect `.env`
* Prefer trusted TLS over disabling verification

---

## Recommended Defaults

```dotenv
SYNC_INTERVAL_SECONDS=60
DNS_SUFFIX=internal
FILTER_INCLUDE_TYPES=qemu,lxc
FILTER_REQUIRE_RUNNING=true
DISCOVERY_VM_ORDER=guest-agent,description,cloudinit
DISCOVERY_LXC_ORDER=config,description
```

---

## Limitations

* DHCP IP discovery for LXCs is not universally available
* Requires guest agent for reliable dynamic VM detection

---

## License

THE BEER-WARE LICENSE (Revision 42)

As long as you retain this notice, you can do whatever you want with this stuff. If we meet someday, and you think this stuff is worth it, you can buy me a beer in return.

-ka0s