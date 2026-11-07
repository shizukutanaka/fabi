# Contributing to fabi

Small surface, honest claims. That's the whole style.

## Before you change code

- Run `fabi selftest` — every claim has a falsification check.
  If your change adds a claim, add a check.
- Keep budgets: binary ≤10 MB, scan <3 s, RSS ≤150 MB,
  zero config, offline core.
- Read `AGENTS.md` — the laws (ledger-is-truth, patterns-are-data,
  deny-default, missing-provenance-reported) are enforced in code.

## What fabi will not accept

- Code that deletes user resources autonomously — deletion is
  always manual (L9).
- Features that can't be tested by `fabi selftest` — a claim that
  can't be falsified isn't a claim (L11).
- Silent failures — missing provenance is reported, not hidden (L2).
- Marketing that outruns evidence — UNVERIFIED claims may not
  appear in marketing (L15).

## How to contribute

1. Fork and open a PR — describe what the change does AND what it
   falsifies (the selftest claim it adds or affects).
2. Keep it small — fabi's style is the smallest useful implementation.
3. Match existing code: compact, no comments unless needed, Go stdlib.

## Reporting bugs

Open an issue with: `fabi version`, OS, the command you ran, and
`fabi why <thing>` output if it involves a recommendation.
fabi's own ledger is the best bug report — paste the relevant
`fabi ledger events` lines.
