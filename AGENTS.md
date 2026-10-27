# AGENTS.md — LabTether Linux

Public Linux compatibility source for the agent.

For Astra and Opus 5.5: edit `AGENTS.md`; keep `CLAUDE.md -> AGENTS.md`.
Read `../AGENTS.md` once if available, or from a worktree use
`$HOME/Development/LabTether/AGENTS.md`. Read only task-relevant docs.
Finish scoped work with focused checks; make routine reversible choices yourself.
Use current manifests, preserve unrelated dirty work, and reply in short plain words.

- Reuse this repo; do not clone or copy it. Worktrees belong under
  `$HOME/.codex/worktrees/LabTether/`; temp files in task `work/` or
  `mktemp -d`. Create no repos, worktrees, caches or temp folders directly in
  `$HOME` or `$HOME/Development`. Clean only your own temp files.
- One broad build/suite at a time; require 100 GB free for heavy builds. Reuse
  caches. Removing user files, dirty work, repos, branches or worktrees needs
  an exact preview and explicit approval. Preview generated-cache cleanup.
- Follow the workspace no-touch target list: no enumeration, queries,
  inspection, backup, operations or QA evidence for those targets.
- Signing material stays local outside repos: never expose, list, copy, stage or
  upload it. Release signing/notarization needs explicit authorization; preserve owner
  signing pauses. Read workspace release rules; publish only verified distributables.
- Prefer scoped disposable credentials or an existing session; never rotate the
  owner password if either is available. Before temporary auth changes, install
  restore/cleanup traps, save the exact state without logging secrets, then
  verify restoration of its hash/timestamp and session baseline.
- In `zsh`, use `rc` or `exit_code`, never reserved `status`.

This file covers routine work in this repo. The workspace guide,
`../AGENTS.md` (from a worktree:
`$HOME/Development/LabTether/AGENTS.md`), has the full rules, lab
machines and release stages. Read it before release, signing, QA, lab or
cross-repo work. If the two disagree, the workspace guide wins; fix the stale
line here.

## How to work

- Carry the task to a checked result. Make routine, reversible choices
  yourself. Stop to ask only for a decision that is the owner's or a boundary
  below, and keep doing independent work while you wait.
- Check the branch and dirty files before editing; leave unrelated work alone.
  Reuse this checkout. Worktrees go under
  `$HOME/.codex/worktrees/LabTether/`; throwaway files go in the
  task's `work/` folder or `mktemp -d`, never directly in `$HOME` or
  `$HOME/Development`. Clean up only what you created.
- Run one broad build or suite at a time and reuse existing caches.
- Reply in short, plain words: what changed, whether it worked, what is next.

## Boundaries

- Run VMs, Docker stacks, simulator QA, installed-agent tests and long suites
  on the lab test machines named in the workspace guide, never on the
  development laptop. One test file, lint or one compile is fine locally.
- Machines the workspace guide marks as no-touch are out of scope: do not
  query, operate on or cite them.
- Signing material stays in its local-only folder outside every repository.
  Never list, copy, print, stage or upload any of it. Signing and notarization
  need the owner's go-ahead for each release.
- Removing repos, branches, worktrees, user files or dirty work needs an exact
  preview and the owner's approval.
- Prefer a scoped disposable token or an existing session to changing auth.
  Before any temporary auth change, set restore traps and confirm the original
  state afterwards. Never rotate the owner password when a safer route exists.
- In `zsh`, `status` is reserved; name exit codes `rc` or `exit_code`.
- Keep personal details out of files, test data and commit messages: real
  names, home folders, lab host names, home IP addresses, personal emails and
  tailnet names. Use neutral examples such as `/Users/demo`, `nas01` and
  `user@example.com`. `python3 scripts/ci/check-private-info.py` and CI block
  them; the workspace git hooks also block the exact private values.

## Checks

- Hard limit: 500 code lines per handwritten file (source, tests, scripts,
  executable CI/build/config), counted by pinned `cloc 2.10`. Run
  `python3 scripts/ci/check-line-limit.py` here. Split by responsibility; do
  not minify, number chunks or move code into data. Existing violations stay
  open until the checker passes.
- Add or keep a test only when it catches a real fault or guards a user
  promise, and name that risk. Skip getter-only, duplicate and
  implementation-mirror tests. Docs-only edits need path, link, diff and
  line-limit checks, not builds.
- Before starting or rerunning remote CI, check the remaining minutes and
  spend cap and name what the run proves. Do not rerun for another green.
- Report what actually passed. A build does not prove runtime behavior, a mock
  does not prove a live result, and a backup is not a restore.

## This repo

This is the separately maintained Linux compatibility source. The canonical
cross-platform agent is `../labtether-agent/`.

- Runtime: `internal/agentcore/`. Platform code: `internal/agentplatform/`.
- Shared compatibility types and helpers: `pkg/agentmgr/`,
  `pkg/securityruntime/` and `pkg/servicehttp/`.
- `go.mod` sets the Go version; README version text can lag it.
- Test the changed package, for example `go test ./pkg/agentmgr` or
  `go test ./internal/agentcore`. Linux-only behavior is checked on a Linux
  lab machine.
- Keep transport, enrollment and trust compatible with Hub and the canonical
  agent. Do not weaken TLS checks to make a connection pass.
- This repo has its own CI. It is not a release prerequisite and must not
  publish a competing canonical agent release.
