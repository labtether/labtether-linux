# AGENTS.md — LabTether Linux Compatibility Source

## Shared rules

For Astra and Opus 5.5: edit `AGENTS.md`; keep `CLAUDE.md -> AGENTS.md`.
Read `../AGENTS.md` once if available, or from a worktree use
`/Users/michael/Development/LabTether/AGENTS.md`. Read only task-relevant docs.
Finish scoped work with focused checks; make routine reversible choices yourself.
Use current manifests, preserve unrelated dirty work, and reply in short plain words.

- Reuse this repo; do not clone or copy it. Worktrees belong under
  `/Users/michael/.codex/worktrees/LabTether/`; temp files in task `work/` or
  `mktemp -d`. Create no repos, worktrees, caches or temp folders directly in
  `/Users/michael` or `/Users/michael/Development`. Clean only your own temp files.
- One broad build/suite at a time; require 100 GB free for heavy builds. Reuse
  caches. Removing user files, dirty work, repos, branches or worktrees needs
  an exact preview and explicit approval. Preview generated-cache cleanup.
- VM 102 / `UntrustedVM` is excluded and untouched: no enumeration, queries,
  inspection, backup, operations or QA evidence.
- Signing material stays local outside repos: never expose, list, copy, stage or
  upload it. Release signing/notarization needs explicit authorization; preserve owner
  signing pauses. Read workspace release rules; publish only verified distributables.
- Prefer scoped disposable credentials or an existing session; never rotate the
  owner password if either is available. Before temporary auth changes, install
  restore/cleanup traps, save the exact state without logging secrets, then
  verify restoration of its hash/timestamp and session baseline.
- In `zsh`, use `rc` or `exit_code`, never reserved `status`.

## File size and checks

- Hard limit: 500 code lines per handwritten source/test/script or executable
  CI/build/config file, using pinned `cloc 2.10` (excludes blanks/comment-only
  lines). Only genuine generated/vendor code is exempt. No legacy exceptions,
  minifying, numbered chunks or moving code into data; split by responsibility.
- Run `python3 scripts/ci/check-line-limit.py` here. Existing violations remain
  open until it passes. Use focused tests and required CI; broaden for shared
  behavior or unresolved failures. Builds do not prove live behavior; backup or
  verification does not prove restore. Report what actually passed.

## Repo guide

This is the separately maintained Linux compatibility source. The canonical
cross-platform agent is the sibling `labtether-agent/` repo.

- Runtime: `internal/agentcore/`; platform code: `internal/agentplatform/`.
- Shared compatibility types/helpers: `pkg/agentmgr/`, `pkg/securityruntime/`,
  and `pkg/servicehttp/`.
- Use `go.mod` for the Go version; older README version text can lag it.
- Run tests for the package changed, for example `go test ./pkg/agentmgr` or
  `go test ./internal/agentcore`. Use a Linux host for Linux-only behavior.
- Keep transport, enrollment, and trust behavior compatible with Hub and the
  canonical agent. Do not weaken TLS checks to make a connection pass.
- This repo has its own CI. It is not a coordinated release prerequisite and
  must not publish a competing canonical agent release.
