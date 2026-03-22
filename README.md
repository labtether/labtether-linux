# LabTether Linux Agent

The Linux agent for [LabTether](https://labtether.com) — reports telemetry, executes actions, and enables remote access for your Linux machines.

## Install

Download the latest binary from [Releases](https://github.com/labtether/labtether-linux/releases/latest):

```bash
curl -fsSL https://github.com/labtether/labtether-linux/releases/latest/download/labtether-agent-linux-amd64 -o /usr/local/bin/labtether-agent
chmod +x /usr/local/bin/labtether-agent
```

Then enroll with your hub:

```bash
labtether-agent --hub wss://your-hub:8443/ws/agent --enrollment-token YOUR_TOKEN
```

For systemd service setup and full configuration, see the [agent setup guide](https://labtether.com/docs/wiki/agents/linux).

## What It Does

- **System telemetry** — CPU, memory, disk, network, and temperature reported to your hub.
- **Remote access** — Terminal and desktop sessions from the LabTether console.
- **Service management** — Start, stop, and restart systemd services remotely.
- **Package updates** — View and apply package updates across your fleet.
- **Docker monitoring** — Container status, logs, and actions for Docker hosts.

## Build From Source

Requires Go 1.24+.

```bash
go build -o labtether-agent ./cmd/labtether-agent/
```

For most users, download the pre-built binary from [Releases](https://github.com/labtether/labtether-linux/releases/latest) instead.

## Links

- **LabTether Hub** — [github.com/labtether/labtether](https://github.com/labtether/labtether)
- **Documentation** — [labtether.com/docs](https://labtether.com/docs)
- **Website** — [labtether.com](https://labtether.com)

## License

Copyright 2026 LabTether. All rights reserved. See [LICENSE](LICENSE).
