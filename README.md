# LabTether Linux Agent: compatibility source

**Looking for the agent?** Install it from your
[LabTether](https://github.com/labtether/labtether) hub (click **Add Device**)
or get an official build from
[labtether-agent](https://github.com/labtether/labtether-agent/releases/latest).

This repository holds a separately tested copy of the Linux agent source. It
does not publish its own releases. Official builds for Linux and Windows
(amd64, arm64) and the Linux container image come from
[labtether-agent](https://github.com/labtether/labtether-agent). macOS uses the
[menu bar app](https://github.com/labtether/labtether-mac). FreeBSD is not
shipped yet.

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
