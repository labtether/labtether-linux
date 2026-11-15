# LabTether Linux Agent: archived

**This repository was archived on 2026-11-15 and is no longer updated.** It
holds an older copy of the Linux agent source and never published releases.

The LabTether agent is developed and released in
[labtether-agent](https://github.com/labtether/labtether-agent). To install
it, open your [LabTether](https://github.com/labtether/labtether) hub and click
**Add Device**, or get an official build from the
[labtether-agent releases](https://github.com/labtether/labtether-agent/releases/latest).
macOS uses the [menu bar app](https://github.com/labtether/labtether-mac).

The rest of this page describes the source as it was when archived.

## What the agent does

Install it on a Linux machine and your hub gets:

- live CPU, memory, disk, network and temperature
- a terminal and remote desktop in the browser, with no SSH keys to copy around
- systemd service control and package updates
- Docker container status, logs and actions

## Build from source

You need Go 1.27.3 or newer (see `go.mod`).

```bash
go build -o labtether-agent ./cmd/labtether-agent/
```

## More detail

- [Install and run by hand](docs/install.md), including a private hub CA and
  moving an agent to a new hub address
- [Self-updates for builds from this repository](docs/self-updates.md)
- [Remote desktop input on Linux (X11)](docs/linux-desktop.md)

## Contributing

If you work with an AI coding agent, point it at [AGENTS.md](AGENTS.md) first.
It covers the checks to run and how this repo relates to `labtether-agent`.

## Help

- [Docs](https://labtether.com/docs) and [GitHub Discussions](https://github.com/labtether/labtether/discussions)
- Security issues: report privately as described in [SECURITY.md](SECURITY.md)

## License

[Apache 2.0](LICENSE). Copyright 2026 LabTether.
