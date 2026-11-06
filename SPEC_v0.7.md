# Fabi
## Universal AI Control Plane

**Version:** 0.7 — The Consolidated Spec

**思考法:** First-Principles Thinking Method
**状態:** Phase 0–1 shipped. All numbers measured `[M]` unless tagged.

0.3 added numbers. 0.4 subtracted. 0.5 derived the atom.
0.6 measured everything. **0.7 is what we actually built.**

---

# Part 1 — What Fabi Is

Fabi is a Computer Intelligence Layer.
Not an LLM. Not a launcher. Not an OS.

It scans your AI estate — tools, models, services, GPU, agents —
writes one append-only ledger, and answers with evidence.

```text
user → goal → tasks → capability → model → hardware → outcome
          ↑ everything below is fabi
```

The user names a purpose. Physics decides feasibility.
Fabi never deletes autonomously. Evidence first, action never automatic.

---

# Part 2 — The Atom (shipped, measured)

One binary, `fabi.exe` — 7.5 MB, Go, 0 config, offline core,
loopback-only network.

| Command | Job |
|---|---|
| `scan [--json]` | 6 probes → ledger → estate report |
| `watch [interval]` | periodic scan → timeseries |
| `doctor` | evidence-backed recommendations |
| `report` | statement — trend + reclaimable |
| `selftest` | falsification engine — claims → tests |
| `ledger` | query the single truth (stats/events/export) |
| `route <task>` | physics: tok/s = bandwidth ÷ resident bytes |
| `goal "<purpose>"` | goal → tasks → local feasibility verdict |
| `install` / `uninstall` | resident watch, dry-run, --yes required |

## Probes (6, failure-isolated, parallel, <3 s)

| Probe | Sees | Method |
|---|---|---|
| process | running AI tools + interpreter workloads | OS APIs + cmdline patterns |
| port | services + owner + HTTP identity | TCP dial + netstat + GET / |
| filesystem | installed-but-idle tools, WSL distros | known dirs + Lxss registry |
| package | installed runtimes/tools | uninstall registry / dpkg |
| gpu | VRAM/util/temp + per-process burn | nvidia-smi |
| models | model files, sizes, duplicates | manifests + ext walk |

## Measured (this PC, RTX A2000 12GB, 32GB RAM)

| Budget | Measured |
|---|---|
| ≤10 MB binary | 7.5 MB `[M]` |
| <3 s scan | 0.1–1.5 s `[M]` |
| ≤150 MB RSS | 10–15 MB `[M]` |
| 0 config | 0 `[M]` |
| offline core | loopback only `[M]` |

---

# Part 3 — The Ledger

One file: `~/.fabi/ledger.db` — append-only SQLite events.
Every state is a view over it. Second truth is deleted.

Event kinds (real, in ledger):

```text
scan.started        probe.completed      resource.found
measurement.sampled report.generated     report.viewed
doctor.recommendation  falsification.*   route.evaluated
goal.evaluated      install.action       user.action
```

PII rule: paths stored for local use; ledger never leaves the device
unless the user exports (`fabi ledger export`).

---

# Part 4 — The Laws (code-verified)

```text
L1  Probe ships with its blind spot documented
L2  Missing provenance → inference note, never silence
L3  Normalize before report, after ledger
L4  Findings unlock deeper probes
L5  Wall-clock measured from process entry
L6  Detection patterns are data files — the first plugin
L7  Coverage deltas are not file deltas
L8  Recommendations carry evidence_ids or don't exist
L9  Deletion is always manual — no --apply exists, by design
L10 Stable recommendations don't re-emit; drifted evidence does
L11 Every claim gets a test; UNVERIFIED stays UNVERIFIED
```

Doctor rules shipped: duplicate content (evidence: identical name+size),
installed-never-running (needs ≥2 scans, 0 running), hidden compute
(owner-invisible services), dead WSL installs, GPU-burn suspects.

---

# Part 5 — Patterns as Plugins

`patterns.json` (embedded) + `~/.fabi/patterns.json` (user merge).
114 entries: exe/cmd/scriptHost patterns, ports, scanDirs,
modelRoots, package matchers.

Measured: add `{"scanDirs": ["~/.fabi"]}` → next scan finds it.
Zero code change = detection is marketplace content already.

---

# Part 6 — Physics Routing (Phase-2 seed, shipped)

```text
tok/s ≈ memory bandwidth ÷ resident model bytes
```

| Task | Verdict on A2000 |
|---|---|
| reasoning (200 tok/s) | 61 tok/s → TOO SLOW `[M]` |
| code_review | POSSIBLE (2.7× slow) `[M]` |
| image_gen | VRAM FIT `[M]` |

`route.evaluated` / `goal.evaluated` events recorded —
the ledger now stores decisions, not only observations.

---

# Part 7 — What This Machine Taught Us

Real estate, real findings (all `[M]`):

- 160 MB dup was a FALSE POSITIVE — same name+size, sha256
  differed (vae vs vae_1_0 revisions). Caught before deletion;
  L14 now requires content-hash equality for `remove` recs.
- 3 idle tools: ollama, lm studio, comfyui — 8/8 scans, never running
- 2 hidden services: port :1234 live JSON API, :8080 dead gateway
- 1 dead WSL distro: OpenClawGateway — registered, vhdx gone
- 10.5 GB unattributed VRAM (WDDM/VM gap) with named suspects
- Unknown tools caught by generic patterns: Strata, hermes-agent
- GPU pulsing 100%→23%→100% — visible only in timeseries

---

# Part 8 — What Remains

**Distribution (external dependency):**
GitHub release → code signing → winget manifest (files ready in
`packaging/`) → `winget install fabi`. Steps in `RELEASE.md`.

**Phase 2 (when numbers demand it):**
- Real routing: heuristic → model pick → run
- Idle/reclaim from timeseries (`was it ever used?` → query)
- Cloud fallback when physics says TOO SLOW
- Resurrection conditions govern what ships — see SPEC v0.4 #82

**Honest UNVERIFIED `[G]`:**
- macOS/Linux runtime (builds clean `[B]`, never ran on hw)
- AMD/Intel/Apple GPU probes
- 150M installs — binding metric is weekly install velocity

---

# Part 9 — The One Number

Install count. Everything else is a means.

```text
0 config → 60 s install → numbers on screen → stays
```

If it runs and asks for nothing, distribution is physics.

---

# Part 10 — Session end-state `[M]`

10 commands shipped: scan, watch, doctor, report, selftest,
ledger, route, goal, install/uninstall, bench.

`bench` measured the machine's first real inference:
the :1234 hidden service identified as a live **LM Studio server**
serving `text-embedding-nomic-embed-text-v1.5` — while `lms ps`
reports nothing loaded. The invisible-owner mystery resolved:
an LM Studio instance outside this host's process view.

Commits: `9c51df0 → c33e6ef → a33ce8e` (8 total).
Next: GitHub release → signing → winget → stranger's install.

---

# Part 11 — Regression lessons, encoded `[M]`

Selftest caught two regressions in production; the fixes followed
the laws they proved:

```text
binary 10.6 MB > budget    FAIL → net/http deleted (raw TCP, 58 lines)
                                 10.6 → 7.6 MB, PASS. Delete beats defer.
nvidia-smi stall 36 s      FAIL → per-probe 1500 ms cap.
                                 Environment stalls; the scan doesn't.
```

**L12: When a dependency costs more than the thing it saves,
the dependency is deleted.**

net/http cost 1.5 MB per call site for a 30-byte GET. A raw socket
does the same job in 58 lines for free. The correct fix was not
optimization — it was deletion.

**L13: Any external process that can stall gets a budget smaller
than the scan's.**

A driver ioctl hung under load; the scan's timeout was theoretical
protection that didn't help against a kernel-stuck child.
Each probe now owns a cap tighter than the global budget.

---

## End

Versions 0.3–0.6 were a spec discovering itself.
Version 0.7 is a spec describing a product that exists.

The difference is the ledger.

Next number that matters: **one stranger's install.**
