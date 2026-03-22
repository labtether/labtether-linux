# LabTether Linux Agent

Telemetry, remote access, and actions for your Linux machines — reported back to your [LabTether](https://labtether.com) hub.

## Install

```bash
curl -fsSL https://github.com/labtether/labtether-linux/releases/latest/download/labtether-agent-linux-amd64 \
  -o /usr/local/bin/labtether-agent && chmod +x /usr/local/bin/labtether-agent
```

Enroll with your hub:

```bash
labtether-agent --hub wss://your-hub:8443/ws/agent --enrollment-token YOUR_TOKEN
```

For systemd service setup, see the [full guide](https://labtether.com/docs/wiki/agents/linux).

## What It Does

- **System telemetry** — CPU, memory, disk, network, and temperature. Reported every heartbeat.
- **Remote terminal & desktop** — Open a shell or desktop session from the LabTether console. No SSH keys needed.
- **Service management** — Start, stop, restart systemd services from the dashboard.
- **Package updates** — See what's outdated. Apply updates across your fleet.
- **Docker monitoring** — Container status, logs, and lifecycle actions for Docker hosts.

## Build From Source

```bash
# Requires Go 1.24+
go build -o labtether-agent ./cmd/labtether-agent/
```

Most users should grab the pre-built binary from [Releases](https://github.com/labtether/labtether-linux/releases/latest).

## Links

| | |
|---|---|
| **LabTether Hub** | [github.com/labtether/labtether](https://github.com/labtether/labtether) |
| **Docs** | [labtether.com/docs](https://labtether.com/docs) |
| **Website** | [labtether.com](https://labtether.com) |

## License

Copyright 2026 LabTether. All rights reserved. See [LICENSE](LICENSE).
