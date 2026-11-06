# Fabi
## Universal AI Control Plane

**Version:** 0.5

**思考法:** First-Principles Thinking Method — Physics of the Atom

**前版:** 0.4（引き算の版）
**今版:** 0.5（Atomを物理から導出する版）

---

# 82. Resurrection Table

0.4 deleted components.

Deletion without a way back is not engineering. It is giving up.

Every deleted thing gets a resurrection condition:

```text
Resurrect ⟺ measured metric crosses threshold
```

Opinion cannot resurrect. Only the Ledger can.

| Component | Killed in | Resurrects when | Measured by |
|---|---|---|---|
| LLM Planner | 0.4 | Heuristic routing regret >15% over 90 days | Ledger `[M]` |
| Goal Engine | 0.4 | >20% of runs are user-typed goals unprompted | Ledger funnel `[M]` |
| Task Engine | 0.4 | >10% of scans spawn multi-step work | Ledger `[M]` |
| Knowledge Graph | 0.4 | Cross-object queries >1K/day AND view rebuild >10 min | Query log `[M]` |
| GUI | 0.4 | CLI daily-active >50% of installs for 90 days | Opt-in telemetry `[M]` |
| SDK | 0.4 | 10K installs AND ≥3 external plugin requests/week | Support log `[M]` |
| Marketplace | 0.4 | 100K installs | Install count `[M]` |
| Multi-PC cluster | 0.4 | Single-PC physics budget saturated on >80% of machines | Monitor `[M]` |
| PAIR / exo / GPUStack | 0.2 | Cluster resurrected AND heterogeneous-hardware demand proven | Ledger `[M]` |
| Cloud routing | deferred | Local tok/s < requirement AND user opts in | Monitor + consent `[M]` |

Rule:

```text
Resurrection condition met → build it.
Resurrection condition unmet → it stays dead.
Dead is a state, not a grave.
```

---

# 83. Beachhead Physics

The market is countable.

Fabi's own probes enumerate it.

| Segment | Population | Source | Tag |
|---|---|---|---|
| Ollama users | ~5M active | ecosystem estimate | `[A]` external |
| LM Studio users | ~2M | ecosystem estimate | `[A]` external |
| GPT4All users | ~0.7M | ecosystem estimate | `[A]` external |
| Jan users | ~0.5M | ecosystem estimate | `[A]` external |
| Adjacent (ComfyUI, Dify, Flowise…) | est. 20–50M | downloads, not users | `[G]` |

Beachhead = local-AI tool users ≈ 8M `[A]`.

Phase 1 exit = 10K installs = ~0.1% of beachhead.

That is a conversion problem, not a market problem.

Conversion is already budgeted: 60 s, 0 config (#59, #76).

1M installs = ~12% of beachhead.

That requires either beachhead expansion or a second segment:

```text
Segment 2 = cloud-only AI users
    ↓
can't be scanned (no local tools)
    ↓
needs import/extension → Phase 4 question, not Phase 1
```

150M by 2035 remains `[G]`.

The binding metric is not the target.

The binding metric is weekly install velocity `[M]`.

Velocity is measured by package managers, not by hope.

---

# 84. Language Physics

The budget eliminates languages before taste does.

| Candidate | Core size | Idle RAM | Cold start | Verdict |
|---|---|---|---|---|
| Electron | ~150 MB+ | ~300 MB+ | ~2 s | DEAD — Chromium alone breaks RAM budget |
| Python + PyInstaller | ~30–50 MB | ~50–80 MB | ~1 s | DEAD — packaging + RAM marginal |
| Node / Bun | ~80–100 MB | ~40–60 MB | ~0.5 s | DEAD — runtime too big for 10 MB core |
| Deno | ~100 MB | ~50 MB | ~0.5 s | DEAD — same |
| .NET | ~20–60 MB | ~30–50 MB | ~0.3 s | MAYBE — NativeAOT only, verify `[B]` |
| Go | ~8–15 MB | ~10–30 MB | ~0.05 s | SURVIVES |
| Rust | ~3–8 MB | ~5–20 MB | ~0.02 s | SURVIVES |
| Zig / C | ~1–5 MB | ~2–10 MB | ~0.01 s | SURVIVES, highest cost |

Physics does not choose the winner.

The build-speed test does:

```text
prototype fabi scan in Go and in Rust
    ↓
whichever ships the 6 probes (#85) first wins
    ↓
loser is deleted, not kept "for later"
```

Spec picks constraints, not religions.

Constraints: single static binary, no runtime deps, cross-compile to 3 OSes, SQLite embedded, offline.

---

# 85. The Scan — `fabi scan` Specified

Cold budget: 3 s (#59).

Achieved by parallelism + per-probe timeouts + failure isolation.

```text
probe crash ≠ scan crash
probe timeout → finding: {probe, timeout} → ledger
```

| Probe | Method | Budget | Finds |
|---|---|---|---|
| process | OS API — Win32 snapshot / sysctl / procfs | <200 ms | Running AI tools |
| port | localhost well-known ports only | <300 ms | Services with no visible process |
| filesystem | known dirs per OS + user dirs | <500 ms | Installed-but-stopped tools, model files, sizes |
| package | registry uninstall keys / brew list / dpkg | <800 ms | Installed inventory |
| GPU | NVML / IOKit / sysfs / rocm-smi | <200 ms | VRAM, model fit floor |
| models | manifests in model dirs | <1 s | Model inventory, quant, size |

Well-known port list is data, not code:

```text
11434  ollama
1234   lm-studio
8188   comfyui
3000   flowise / misc
3001   anythingllm
7860   sd-webui
8080   open-webui / lm-studio-alt
```

Probe definitions live in a data file.

Data files are plugins later. Code is not.

Scan output = events, not a report:

```text
scan.started
probe.completed × N
resource.found × M
scan.completed {duration_ms, findings}
```

Report is generated from events. Events are truth. Report is a view (#73).

---

# 86. Ledger Schema v0

Append-only. SQLite. One file: `~/.fabi/ledger.db`.

User-readable. Exportable. No cloud sync in Phase 0.

| Event | Payload |
|---|---|
| `scan.started` | ts, host_id, fabi_version |
| `probe.completed` | probe, duration_ms, findings, error? |
| `resource.found` | kind, name, size_bytes, path_hash, source_probe |
| `measurement.sampled` | gpu_vram_used, ram_used, gpu_util |
| `doctor.recommendation` | kind, target, evidence_ids, confidence |
| `user.action` | action, target, accepted |
| `report.generated` | period, totals |

`host_id` = hash of machine GUID.

No username. No hostname. No file paths in cleartext — `path_hash` only.

Local-first means the ledger never leaves unless the user exports it.

PII in the ledger is a policy violation, not a bug.

---

# 87. The Report

Phase 0 report shows only what the Ledger knows:

```text
YOUR AI ESTATE — 2026-11-07

TOOLS FOUND        7
  running          3
  installed-idle   4

MODELS             23
  disk used        187 GB
  duplicates       3 (qwen2.5:7b ×3 quants)

GPU                RTX-class, 12 GB VRAM
  idle RAM cost    2 tools × ~1 GB = ~2 GB always-on

RECLAIMABLE        ~43 GB disk, ~2 GB RAM idle
```

No ¥ line yet.

Yen requires cost data, and cost data requires routing (Phase 2).

Claiming ¥ in Phase 0 = vaporware (#58).

The report earns the ¥ line when the Ledger earns the data.

---

# 88. Permission Defaults — Phase 0

Scan is read-only.

Read-only needs no permission prompt. That is the 0-config law.

| Action | Default |
|---|---|
| scan probes | allow, silent |
| doctor recommendations | suggest only |
| doctor apply | requires `fabi doctor --apply <id>` |
| writes | only inside `~/.fabi/` |
| telemetry | OFF. No telemetry in Phase 0–1 |
| network | OFF. Loopback probes only |

Install count is measured by package managers and opt-in counters later — not by phoning home.

A scanner that phones home is a spy with good marketing.

Spies are deleted (#58).

---

# 89. What v0.5 Still Doesn't Do

- Does not pick the language. The build-speed test picks it (#84).
- Does not write code. The spec ends where the compiler begins.
- Does not define the plugin ABI. An ABI for a product with one probe is fiction.
- Does not schedule Phase 2. Resurrection Table decides, not calendar.

v0.5 is the last spec-only version.

v0.6, if it exists, is written from Ledger data — not from thought.

Spec follows measurement. Measurement does not follow spec.

---

# 90. This Week, Re-stated in Physics

```text
build fabi scan
    ↓
Go or Rust — whichever probes finish first
    ↓
6 probes, <3 s cold, <150 MB RAM
    ↓
ledger.db writes events
    ↓
report prints numbers
    ↓
run it on this PC
```

This PC is the first test subject.

The spec's author is the first user.

Dogfooding is not a virtue. It is the only honest test.

If `fabi scan` on the author's own machine takes >3 s or >150 MB, the physics budget is broken before a single user installs it.

Broken budget found in-house costs nothing.

Broken budget found by a user costs the launch.

---

## End

0.3 put numbers on words.

0.4 subtracted the spec.

0.5 subtracted the mystery.

Every deferred component now has a resurrection metric.
Every probe has a time budget.
Every event has a schema.
The market is countable.
The language is constrained.

What remains is not speculation.

What remains is physics.

Build the physics.
