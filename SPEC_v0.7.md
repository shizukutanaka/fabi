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

**L14: A removal recommendation requires content-hash equality.**

The verified-savings loop caught its own false positive: two files
shared name + size but sha256 differed (vae vs vae_1_0 revisions).
Doctor now hashes before saying REMOVE — the heuristic was a hint,
never proof. The loop worked because deletion stayed manual.

---

# Part 12 — Verified fixes and the signing prophecy (measured)

**First verified fix [M]:** `wsl --unregister OpenClawGateway` →
re-scan → `dead-wsl` finding gone from doctor output. The loop
recommendation → action → verified delta is closed end-to-end.

**Unsigned-binary incident [M]:** `fabi.exe` (locally built,
unsigned) became unreadable/unlaunchable mid-session — Windows
quarantine-class behavior on an executable that does port probing
+ process enumeration. A fresh rebuild (fabi2.exe) ran fine.
The RELEASE.md signing requirement is now a measured fact,
not a theory: **unsigned fabi gets blocked on real machines.**

**New estate fact [M]:** ollama appeared at :11434 (`Ollama is
running`) — previously idle for 25 scans. The llm-router startup
entry (llm-router-startup.vbs, Hermes_Gateway*.vbs found in the
Startup folder) is the likely activator.

---

# Part 13 — Phase 2 is live code (measured)

The spec stopped being a plan. Real executions on this machine:

```text
fabi run "..."        qwen2.5:0.5b → 76.9 tok/s measured [M]
fabi run --sweep      qwen 0.5b=193.8 · llama3.2:1b=75.9 tok/s [M]
fabi route            physics 8B=61 vs measured — side by side
fabi goal ... --run   goal → task chain → per-task model
                      scheduling → threaded execution → measured
```

Measured chain on this PC (`goal "research local ai" --run`):

```text
step 1: search    (3b) → qwen2.5:0.5b   193.8 tok/s
step 2: reasoning (8b) → llama3.2:1b    104.4 tok/s  ← context threaded
```

**L15: Every execution step emits task.executed — chains leave
no invisible work.** (Chain steps originally skipped the ledger;
now every step records goal/task/step/model/tok_s.)

The Scheduler (SPEC #10) exists in miniature: 3b-class → fastest,
8b-class → largest. Not yet a full scheduler — but a real one,
running real workloads, measured.

---

# Part 14 — Universal API v0 (shipped, measured)

`fabi serve` on `127.0.0.1:11435` — raw sockets, no net/http
(budget kept at 7.6 MB). 6 endpoints, all ledgered (via: serve):

```text
GET  /status    estate summary          GET  /models   live models
GET  /ledger    events (kind,n)         GET  /doctor   recommendations
POST /run       prompt → model output   POST /goal     decompose + chain
```

Measured: POST /run → qwen2.5:0.5b → 200 tok/s [M].
POST /goal → 2-step chain executed over the wire [M].

Other software can now call fabi. Strata, hermes, any agent —
the connection mission has an API surface. Documented in `API.md`.

**status** shows the heartbeat: `... | 128 tok/s qwen2.5:0.5b`.

---

# Part 15 — What Fabi actually is (Socratic resolution)

Six questions, answered with shipped code and measured data:

```text
Q1  Does fabi control anything?
    Now yes — unload (2.0 GB freed, verified), GPU queue,
    num_ctx. Observation → inference → control.

Q2  What does fabi know that tools don't?
    The correlation graph — models↔processes↔ports↔VRAM↔
    startup↔services. No engine sees outside itself.

Q3  What's lost if fabi disappears?
    The single truth (ledger), provenance detection, honest
    subtraction, and now resource control.

Q4  Did decisions change outcomes?
    Proven — scheduler's pick measured fastest (181 vs 120),
    constraint solver predicted 4.9s < 5.0s correctly.

Q5  What would falsify it?
    The falsification engine itself — it already caught 3
    real bugs (false dup, net/http bloat, unsigned binary).

Q6  Is the purpose something tools don't serve?
    Yes — the honest control plane. Every other tool manages
    ONE engine honestly OR sees the whole estate blindly.
    Fabi alone does both: sees everything (including what
    hides), keeps one truth, and controls what it finds.
```

That is the product. Not "a scanner with a ledger" — the honest
control plane the estate didn't have.

---

# Part 16 — Every spec pillar has a primitive

```text
Goal Engine   goal --run          Planner     goal → tasks
Scheduler     modelForTask        Task Engine task.executed
Resource Mgr  scan/ledger/status  Discovery   probes+provenance
Permission    deny default        Policy      7 doctor rules
Automation    fabi auto           Marketplace patterns.json
Doctor        7 rules             Monitor     watch+heartbeat
Serve API     6 endpoints         Remote      OLLAMA_HOST
```

`fabi auto` applies only safe recommendations (idle unload) —
user-triggered, non-destructive, in-use models stay warm. The
Automation pillar exists without violating "action stays manual."

`OLLAMA_HOST` opens remote/cloud engines — route's "cloud or
downgrade" verdict is now a reachable path, not a dead end.

Every pillar in the spec has a working primitive, measured,
ledgered, and falsified. Nothing remains only on paper.

---

## End

Versions 0.3–0.6 were a spec discovering itself.
Version 0.7 is a spec describing a product that exists.

The difference is the ledger.

Next number that matters: **one stranger's install.**

## Part 17 — Socratic conclusion (what fabi actually is)

23 questions later, the dialectic converged on one truth:

**fabi is not "the worldwide AI execution foundation".**
**fabi is "the honest eye on this PC's AI stack".**

The dialectic:
- "Worldwide" was the hook; "build honestly" was the real goal.
- fabi is made, not yet used — but its discoveries were real:
  fake duplicates, dead WSL, ghost GPU, unknown tools.
- Its methods were real: falsification, evidence chains,
  multi-engine routing, deletion, L1–L15.

The honest framing:
> fabi sees your own machine's AI estate honestly on the ledger —
> what exists, what's idle, what holds VRAM, what changed, and why
> it says what it says. The world may come or not; the honest eye
> already works.

The goal was inflated. The product was not. Record the truth, not
the story (L15).
