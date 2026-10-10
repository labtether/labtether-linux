# LabTether Linux Agent Compatibility Source

Telemetry, remote access, and actions for your Linux machines — reported back to your [LabTether](https://labtether.com) hub.

This repository is independently tested compatibility source. Official Linux
agent binaries are built and published by the canonical
[`labtether-agent`](https://github.com/labtether/labtether-agent) repository;
this repository does not publish a second competing release line.

## Install

Use **Add Device** in the Hub console and copy its Linux install command. That
command selects the right Hub address, sets up certificate trust, and installs
the service. For a manual binary install, choose an exact version and follow
the checksum and build-attestation steps in the [canonical agent guide](https://github.com/labtether/labtether-agent#linux).

To run the installed agent in the foreground with a Hub certificate already
trusted by this machine:

```bash
umask 077
token_file="$(mktemp)"
trap 'rm -f "$token_file"' EXIT
read -r -s -p 'Enrollment token: ' enrollment_token
printf '\n'
printf '%s\n' "$enrollment_token" > "$token_file"
unset enrollment_token
sudo env \
  LABTETHER_WS_URL=wss://your-hub:8443/ws/agent \
  LABTETHER_ENROLLMENT_TOKEN_FILE="$token_file" \
  /usr/local/bin/labtether-agent
```

For a private Hub CA, use the Hub's generated install command or set
`LABTETHER_TLS_CA_FILE` to a trusted local CA file. For systemd setup, see the
[full agent guide](https://labtether.com/docs/install-upgrade/agent-install-commands-by-os).

On later starts, the agent restores the Hub addresses saved at enrollment. To
move it to a new Hub address, set `LABTETHER_WS_URL` or
`LABTETHER_API_BASE_URL`; the other address is derived from that same origin.
If your WebSocket and API use different origins, set both variables together.
An explicit `LABTETHER_TLS_CA_FILE` remains in use after enrollment.

## What It Does

- **System telemetry** — CPU, memory, disk, network, and temperature. Reported every heartbeat.
- **Remote terminal & desktop** — Open a shell or desktop session from the LabTether console. No SSH keys needed.
- **Service management** — Start, stop, restart systemd services from the dashboard.
- **Package updates** — See what's outdated. Apply updates across your fleet.
- **Docker monitoring** — Container status, logs, and lifecycle actions for Docker hosts.

## Build From Source

```bash
# Requires Go 1.27.2+ (see go.mod)
go build -o labtether-agent ./cmd/labtether-agent/
```

Most users should grab the pre-built binary from the canonical
[`labtether-agent` releases](https://github.com/labtether/labtether-agent/releases/latest).

## Links

| | |
|---|---|
| **LabTether Hub** | [github.com/labtether/labtether](https://github.com/labtether/labtether) |
| **Docs** | [labtether.com/docs](https://labtether.com/docs) |
| **Website** | [labtether.com](https://labtether.com) |

## License

Copyright 2026 LabTether. All rights reserved. See [LICENSE](LICENSE).
