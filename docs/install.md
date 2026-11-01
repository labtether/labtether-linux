# Install and run by hand

Use **Add Device** in the Hub console and copy its Linux install command. That
command selects the right Hub address, sets up certificate trust, and installs
the service. For a manual binary install, choose an exact version and follow
the checksum and Ed25519 signature steps in the [canonical agent guide](https://github.com/labtether/labtether-agent/blob/main/docs/install.md#linux).
Release binaries are signed locally by the maintainers; they have no GitHub
build attestation.

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
[full agent guide](https://labtether.com/docs/guides/managing-nodes).

On later starts, the agent restores the Hub addresses saved at enrollment. To
move it to a new Hub address, set `LABTETHER_WS_URL` or
`LABTETHER_API_BASE_URL`; the other address is derived from that same origin.
If your WebSocket and API use different origins, set both variables together.
An explicit `LABTETHER_TLS_CA_FILE` remains in use after enrollment.
