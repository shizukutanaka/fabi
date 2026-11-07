# Security Policy

## What fabi touches — and what it never does

fabi inspects your local AI estate. By design:

- **Loopback only** — probes hit `127.0.0.1` services. The single
  outbound call is `fabi update` (GitHub releases), and only when
  the user runs it. Nothing is sent anywhere else.
- **Read-mostly** — scans and probes never modify your system. The
  only writes are `fabi install` (a Startup launcher, dry-run
  without `--yes`) and `fabi unload` (model VRAM release via the
  engine's own API).
- **Deny by default** — install/uninstall are dry-run unless
  `--yes` is passed. Nothing deletes your data; `fabi uninstall`
  keeps the ledger.
- **Append-only ledger** — `~/.fabi/ledger.db` records events; it
  is never mutated in place and never leaves the machine unless you
  `export`/`import` it yourself.
- **Patterns are data** — detection rules are JSON, never executed
  code, so a Marketplace pattern can't run anything on your machine.

## Known limits (honest)

- The `fabi serve` API binds `127.0.0.1` only — it trusts whatever
  local process can reach it. Don't expose the port.
- `OLLAMA_HOST` routes execution to a remote engine you configure —
  fabi trusts it as configured, but prompts still leave your machine
  in that case. Set it deliberately.
- `fabi ledger import` merges foreign event data — events are stored
  verbatim (provenance kept), and derived views ignore imported
  ranges, but don't import ledgers you don't trust as *truth*.

## Reporting a vulnerability

If you find a security issue — anything that lets fabi read or write
beyond its described surface, or that makes it reach out beyond its
one update call:

- **Don't open a public issue for a live vulnerability.**
- Email the maintainer via the address on the GitHub profile of
  `shizukutanaka/fabi`, or use GitHub's private vulnerability
  reporting ("Security" → "Report a vulnerability") on the repo.

Include: fabi version (`fabi version`), OS, the command you ran, and
what it did vs what it should do. We'll acknowledge within a few
days and either fix or document it honestly — a security gap that
can't be fixed is documented as a limit, not hidden (L15).

## Out of scope

- Issues in the engines fabi talks to (Ollama, LM Studio, ComfyUI) —
  report those upstream.
- Findings fabi surfaces *about* your machine (that's its job).
