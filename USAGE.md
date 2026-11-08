# fabi — User Guide

The honest eye on your PC's AI estate. This is how to actually use it.

## First 60 seconds

```sh
fabi            # shows the one-line status (or an intro on first run)
fabi scan       # detect every AI tool, model, service, GPU on this PC
fabi doctor     # evidence-based recommendations — what to remove/idle
fabi status     # one-line health, anytime
```

That's the whole loop. Everything below is optional depth.

## Daily workflows

**"What's on this machine?"**
`fabi scan` → `fabi report` — estate inventory + measured trends.

**"What's changed since yesterday?"**
`fabi diff` — what appeared / what vanished between scans.
`fabi diff 5` — compare against 5 scans back.

**"What should I get rid of?"**
`fabi doctor` — recommendations tagged remove / idle / investigate,
each with a confidence note and evidence.

**"Why does it say that?"**
`fabi why <target>` — the ledger events that justify a recommendation.
Every rec traces to real evidence; `fabi why` shows the chain.

**"Run something on a model"**
`fabi run "prompt"` — real measured inference (tok/s, TTFT).
`fabi run --sweep` — benchmark every live model.
`fabi goal "summarize this" --run` — decompose a purpose into tasks
and execute what's feasible locally.

**"Free up VRAM"**
`fabi unload` — unload loaded models.
`fabi auto` — applies only safe, reversible actions (idle unload).
Records `auto.skipped` when it chooses not to act.

## Machine-readable

Every major command takes `--json` — for scripts and monitoring:

```sh
fabi status --json     # health as an object
fabi doctor --json     # recommendations as an array
fabi diff --json       # estate delta as lists
fabi scan --json       # full findings as JSON
fabi report --json     # the statement as numbers
fabi ledger stats --json
fabi why X --json      # evidence chain
fabi update --json     # version delta
fabi goal "..." --json # task decomposition + verdicts
fabi route X --json    # physics verdict
fabi run "..." --json  # measured inference
```

## The Universal API

`fabi serve` exposes the same surface on `127.0.0.1:11435` —
loopback only by design:

```
GET  /status /models /ledger /doctor /diff /why /pattern /update /route
POST /run /goal /unload /scan /pattern /import
```

## Multi-PC estate

```sh
# on the other machine:
fabi ledger export > estate.jsonl
# move the file, then on this machine:
fabi ledger import estate.jsonl        # or POST /import
fabi ledger stats                      # "scans by os" shows origins
```

Foreign scans are stored verbatim (provenance kept) and excluded
from local recommendations — merging never mixes estates silently.

## Updating

```sh
fabi update     # compares your version to the latest GitHub release
```

fabi never auto-downloads — it reports the delta and the URL;
you replace the binary. Deny-by-default holds even for itself.

## Language

Output comes in 10 languages: en/ja/zh/es/de/fr/ko/pt/ru/it.
Detection order: `FABI_LANG` > `LANG` > `LC_ALL` > OS UI language > en.

```sh
fabi lang   # shows detected language, its source, supported list
            # and how to override (set FABI_LANG=<code>)
```

A new language is just data — add `lang_<code>.go` with the
message map and detection works automatically.

## Troubleshooting

| Symptom | Honest answer |
|---|---|
| `fabi run` says "no live generative model" | start `ollama serve` or load a model in LM Studio |
| `fabi auto` says "nothing to do" | either no engine running, or all loaded models are in use — both are correct states |
| SmartScreen warning on first run | binary is unsigned — see README for verification (SHA256SUMS); signing is planned |
| `fabi update` can't reach GitHub | you're offline or behind a proxy — everything else still works |
| `fabi why X` shows "no recommendation" | fabi hasn't recommended anything about X — it shows raw observations instead |

## Philosophy, in one line

fabi never guesses and never hides: it records what exists, what's
idle, what holds VRAM, what changed, and why it says what it says —
and says `UNVERIFIED` when it can't prove something.
