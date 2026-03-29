# LAN Dash

A real-time LAN device discovery and monitoring dashboard built in Go. Discover devices on your network, track their status, and monitor open ports — all from a web browser or SSH terminal.

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)

## Features

- **Automatic device discovery** — ARP table monitoring + subnet sweep on startup
- **mDNS hostname resolution** — resolves `.local` names (printers, smart TVs, Home Assistant)
- **Port scanning** — detects common services (HTTP, SSH, Plex, MQTT, Home Assistant)
- **Vendor lookup** — identifies manufacturers via embedded OUI database
- **Real-time web dashboard** — Datastar SSE for live DOM updates, zero JavaScript written
- **SSH terminal UI** — Bubble Tea TUI accessible from any machine on your LAN
- **Scan history** — SQLite persistence with 24-hour RTT trend API
- **Graceful shutdown** — clean signal handling, no orphaned connections

## Quick Start

```bash
# Clone and run
git clone https://github.com/emm5317/lan-dash.git
cd lan-dash
go run ./cmd/server/

# Open in browser
open http://localhost:3000

# Or connect via SSH TUI
ssh -p 2223 localhost
```

## Architecture

```
                    ┌──────────────┐
                    │   main.go    │
                    │ signal ctx   │
                    └──────┬───────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼─────┐     ┌─────▼──────┐    ┌──────▼──────┐
   │ Scanner  │     │  HTTP :3000 │    │  SSH :2223  │
   │ ARP poll │     │  Datastar   │    │  Bubble Tea │
   │ mDNS     │     │  REST API   │    │  Lip Gloss  │
   │ Sweep    │     └─────▲──────┘    └──────▲──────┘
   └────┬─────┘           │                  │
        │           ┌─────┴──────────────────┘
        │           │
   ┌────▼───────────▼────┐     ┌──────────┐
   │   Store (in-memory) │────►│  SQLite   │
   │   pub/sub events    │     │  history  │
   └─────────────────────┘     └──────────┘
```

**Data flow:** Scanner detects ARP changes → enriches devices (ping, ports, vendor, DNS) → upserts to Store → Store broadcasts to subscribers → Web UI morphs DOM via SSE, TUI re-renders table.

## Web Dashboard

The web UI uses [Datastar](https://data-star.dev/) for reactive server-sent event updates. The table auto-populates on page load and updates in real-time as devices come online or go offline.

- Filter devices by IP, vendor, or hostname
- Trigger manual scans with the "Scan now" button
- RTT color-coding: green (fast), amber (medium), coral (slow)

## SSH TUI

Connect from any machine on your LAN:

```bash
ssh -p 2223 <server-ip>
```

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `/` | Enter filter mode |
| `s` | Trigger scan |
| `q` | Quit |
| `Esc` | Clear filter |

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/devices` | All discovered devices (JSON) |
| `GET` | `/api/events` | SSE stream (Datastar fragments) |
| `POST` | `/api/scan` | Trigger device enrichment |
| `GET` | `/api/history?ip=10.0.0.1` | 24-hour RTT history (JSON) |

## Scan History

Every device update is recorded to SQLite (`lan-dash.db`). Query the RTT trend:

```bash
curl "http://localhost:3000/api/history?ip=10.0.0.1"
```

```json
[
  {"ip": "10.0.0.1", "rtt_ms": 3.3, "alive": true, "open_ports": [80,443], "timestamp": 1774756780}
]
```

### Litestream Backup

Replicate the SQLite WAL to Backblaze B2 (or any S3-compatible store) with zero code changes:

```bash
# Edit litestream.yml with your B2 credentials, then:
litestream replicate -config litestream.yml
```

## Ports Scanned

The scanner checks these common service ports on each discovered device:

| Port | Service |
|------|---------|
| 22 | SSH |
| 80 | HTTP |
| 443 | HTTPS |
| 3000 | Dev servers |
| 5000 | Flask / Docker registry |
| 8080 | HTTP alt |
| 8443 | HTTPS alt |
| 8123 | Home Assistant |
| 9000 | Portainer |
| 1883 | MQTT |
| 5353 | mDNS |
| 32400 | Plex |

## Project Structure

```
lan-dash/
├── cmd/server/main.go          # Entry point, signal handling
├── internal/
│   ├── api/
│   │   ├── handlers.go         # HTTP routes
│   │   └── datastar.go         # SSE stream handler
│   ├── scanner/
│   │   ├── sweep.go            # Subnet ARP population
│   │   ├── netlink.go          # ARP polling + enrichment
│   │   ├── ping.go             # TCP ping
│   │   ├── ports.go            # Port scanner
│   │   ├── mdns.go             # mDNS discovery
│   │   └── oui.json            # Vendor database
│   ├── store/
│   │   └── store.go            # In-memory store + pub/sub
│   ├── history/
│   │   └── history.go          # SQLite persistence
│   └── tui/
│       ├── wish.go             # SSH server
│       ├── model.go            # TUI logic
│       └── table.go            # TUI rendering
├── web/
│   └── index.html              # Datastar web UI
├── litestream.yml              # Backup config
└── go.mod
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Terminal UI framework |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | TUI styling |
| [Wish](https://github.com/charmbracelet/wish) | SSH server middleware |
| [Datastar](https://data-star.dev/) | Reactive SSE web UI |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Pure-Go SQLite (no CGO) |
| [hashicorp/mdns](https://github.com/hashicorp/mdns) | mDNS service discovery |

## License

Apache 2.0
