# Updates for compatibility builds

Self-updates require `LABTETHER_AUTO_UPDATE_TRUSTED_PUBLIC_KEY` and signed
metadata. A missing key stops the update. Signatures use the canonical
agent's payload (version, OS, architecture, SHA-256 and size), so canonical
release signatures verify here. The release public key is published in the
Hub repository at `scripts/release/agent-release-public-key.b64` and in the
canonical agent README. `LABTETHER_AUTO_UPDATE_ACCEPT_UNSIGNED=true` is
an explicit opt-in for development only and does not bypass a configured key.

Update metadata is limited to 1 MiB. Releases must match the running OS and
architecture and use a stable version. Older versions are rejected even for a
forced update. Hub tokens are sent only to the configured Hub origin, never to
a separate update mirror or a redirect on another origin.
