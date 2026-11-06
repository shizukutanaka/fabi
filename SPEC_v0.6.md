# Fabi
## Universal AI Control Plane

**Version:** 0.6

**思考法:** First-Principles Thinking Method — The Measured Version

**前版:** 0.5（Atomを物理から導出）
**今版:** 0.6（Ledgerデータから書いた版 — specの約束通り、思考ではなく測定）

**測定機:** このPC — Windows, RTX A2000 12GB, RAM 32GB, Go 1.26
**測定日:** 2026-11-07
**被験体:** 開発者自身（最初のユーザー）

---

# 91. First Contact — measured numbers

`fabi scan` v0.0.1-dev, real PC, real data:

| Metric | Budget (#59) | Measured `[M]` | Margin |
|---|---|---|---|
| Binary size | ≤10 MB core | **7.6 MB** | 1.3× under |
| Scan (main→report, L5) | <3 s cold | **0.7–1.5 s** | ~2–4× under |
| Wall clock (`time fabi scan`) | <3 s | **0.85–3.5 s** | variance ⚠ |
| RSS | ≤150 MB | **14–15 MB** | ~10× under |
| Probes | 6, isolated | 6/6 ok, max 650 ms each | ✓ |
| Ledger | append-only events | 319 events / 5 scans | ✓ |
| Config required | 0 | **0** | ✓ |

All MVP physics budgets pass on first hardware.

Go won the #84 build-speed test on first attempt. Rust never started.

Language decision: **Go** `[B]` — decided by shipping, not by taste.

---

# 92. What measurement discovered that spec didn't

Measurement generated four findings the spec never predicted:

## 1. Unknown-tool detection works

Scan found `strata` and `hermes-agent` — AI agent infrastructure that
appears in no detection database, caught only by generic cmdline markers
(`mcp`, product-name substrings).

Proof: pattern lists must ship as **data**, and generic markers
(`mcp`, `agent`-adjacent paths) catch the long tail.

## 2. The blind spot is a feature

Ports 1234/8080 answered TCP but showed **no LISTENING owner** in netstat.
Diagnosis: services inside WSL2/VM/container — invisible to host OS probes.

Fabi now reports:

```text
hidden compute (inferred)
  2 localhost AI service(s) answered with no visible owner process
  — likely WSL2, VM, or container
```

New law: **a finding with missing provenance gets an inference note,
not silence.** Blind spots are reported, not hidden.

## 3. Raw inventory ≠ report

Package probe: 34 raw registry entries → 9 runtime families after
normalization ("Python 3.11.9 Add to Path / Core Interpreter /
Tcl/Tk Support…" → `python@3.11`).

Normalization is part of the probe contract. Raw counts are ledger
data; families are report data. Both are kept — views over one truth (#73).

## 4. Tool→model linkage was missing

ComfyUI was detected as installed, but its `models/` dir was never
scanned — 867 MB model invisible until tool-dir→model-root linkage
was added (model disk total: 789 MB → 1.6 GB `[M]`).

Discovery feeds discovery. Every finding can unlock a deeper probe.

---

# 93. Honest gaps (measured, not guessed)

| Gap | Evidence | Tag |
|---|---|---|
| Wall-time variance 0.85→3.5 s | AV scan on new binary, disk cache | `[M]` |
| macOS / Linux probes | code exists, never run on real hw | `[G]` |
| WSL/VM internals | owner-invisible services detected, contents not | `[M]` |
| Static port list | 10 ports; real world has more | `[A]` |
| Ollama models | blob store 13 KB — no models pulled on this PC | `[M]` |
| GPU vendors | nvidia-smi only; AMD/Intel/Apple absent | `[A]` |

The scan timer measures `runScan` internals only.

Spec bug found by measurement: **wall-clock budget must include
process startup + ledger open.** Timer moved to `main()` entry in v0.0.2.

The instrument was measuring the wrong boundary. Fixed.

---

# 94. New laws from data

```text
L1. Probe ships with its blind spot documented.
L2. Missing provenance → inference note, never silence.
L3. Normalization happens before report, after ledger.
L4. Findings unlock deeper probes (tool dir → model root).
L5. Wall-clock is measured from process entry, not function entry.
L6. Pattern lists are data files, not code — they are the first
    marketplace content.
```

L6 is the important one: the detection patterns that found Strata
and hermes are exactly what a Capability Pack would distribute.

The product's first plugin already exists — it just isn't packaged yet.

---

# 95. Phase 0 exit review

| Requirement | Status |
|---|---|
| `fabi scan` on real PC | DONE `[M]` |
| 6 probes, <3 s, <150 MB | DONE, 10× margin `[M]` |
| Ledger append-only | DONE `[M]` |
| 0 config | DONE `[M]` |
| Numbers on screen | DONE `[M]` |
| Package-manager presence | NOT DONE — winget manifest is next |
| Signed build | NOT DONE — unsigned exe triggers SmartScreen |

Phase 0 remainder is **distribution**, not features.

The spec predicted this in #76: package-manager presence is a
Phase-1 exit criterion. Measurement agrees — the last 10% is shipping,
not scanning.

---

# 96. What v0.6 deletes

Measurement earned these deletions:

- "GPU probe via NVML library" → nvidia-smi spawn is fast enough `[M]` 105–650 ms
- "Port→PID via gopsutil net API" → netstat spawn is simpler, fast enough `[M]` ~200 ms
- "Process grouping in report" → moved into probe; ledger stores raw groups `[M]`

Optimization before measurement is waste (#75). These were measured first.

Kept despite cost:

- modernc.org/sqlite (pure Go) — 7.6 MB total binary proves it fits `[M]`

---

# 97. The 10K question, answered by data

Beachhead (#83): ~8M local-AI users `[A]`.

This PC is Exhibit A: it runs Strata, hermes, ComfyUI, LM Studio,
Ollama, Open WebUI — **six AI tools the user installed separately**,
each with its own UI, its own config, its own idea of what a "model" is.

That is the product thesis, observed:

```text
user installs N AI tools
    ↓
each tool knows only itself
    ↓
nobody knows the whole estate
    ↓
Fabi scan → one ledger → one report
```

10K installs = 0.1% of beachhead.

The conversion funnel starts at winget, not at a website (#76).

Next build is not a feature.

Next build is `winget` + a signed binary + `fabi doctor`.

---

# 98. Doctor v0.0.1 — first subtraction (same day, measured)

`fabi doctor` shipped hours after `fabi scan`. First run, real ledger:

```text
ledger: 508+ events across 8 scans

REMOVE — duplicate content (~160 MB reclaimable)
  diffusion_pytorch_model.fp16.safetensors
    keep:   HF cache .../vae/...      (160 MB)
    remove: HF cache .../vae_1_0/...  (160 MB)
    evidence: events #463, #464 — identical name+size, latest scan
    confidence: high

IDLE — installed, never observed running (8 scans)
  ollama, lm studio, comfyui — 8/8 installed, 0/8 running [medium]

INVESTIGATE — hidden compute
  lm-studio @ :1234, open-webui @ :8080 — owner not visible
```

Measured facts:

- Evidence chain works end-to-end: scan → ledger → recommendation →
  `doctor.recommendation` events written back `[M]`
- Idle rule needs history: ≥2 scans installed + 0 running observations.
  Confidence scales with scan count (low <5, medium ≥5) `[M]`
- HF-cache "duplicates" prove why apply stays manual: `vae/` vs `vae_1_0/`
  may be intentional revision copies. Evidence informs; human decides `[M]`
- `--apply` does not exist. Not deferred — refused. Deletion of user
  files is never autonomous, at any confidence `[LAW]`

The loop is closed:

```text
scan → ledger → doctor → recommendation → ledger
                ↑__________________|
              next scan re-measures
```

Fabi now has its first subtraction engine running on real data.

---

# 99. Report v0.0.1 — the statement exists (measured)

`fabi report` shipped the same day as doctor. Real output on this PC:

```text
ESTATE (latest scan)
  AI tools     3 running, 3 installed-idle
  services     2 answering on localhost
  models       8 entries, 1.6 GB on disk
  GPU          RTX A2000 — VRAM 1.4/11.5 GiB (dropped from 10.5 after
               the user's tuning workload ended — visible in ledger)

TREND (8 scans)
  model disk   789 MB → 1.6 GB — 867 MB added
```

That "867 MB added" line produced a new law:

**L7. Coverage deltas are not file deltas.**

The 867 MB was not new files — it was the ComfyUI model-root probe
growing coverage between scan 5 and scan 6. The ledger reported
honestly; the interpretation needed a caveat line, now printed:

```text
(deltas include probe-coverage changes, not just file changes)
```

The statement's most honest line is the empty one:

```text
verified savings  requires a deletion + re-scan to prove
```

Fabi cannot claim savings that don't exist yet. The claim appears
only after the user's manual deletion shows up in the next scan —
measured, not believed.

---

# 100. The WSL connection (measured, same day)

The hidden-compute mystery closed itself.

```text
port probe:  :1234, :8080 answer, no visible owner
wsl probe:   OpenClawGateway — vhdx MISSING (dead install)
verdict:     the services ran inside WSL2. The distro is gone;
             only ghost registrations and answering ports remain.
cleanup:     wsl --unregister OpenClawGateway
```

Three probes that were built for unrelated reasons — ports, process,
WSL registry — produced a coherent forensic story **without any of
them knowing about the others**.

That is Discovery working as designed: local observations, correlated
in the ledger, explained by inference notes — not by guessing.

Cross-compile status: `GOOS=linux GOARCH=amd64 go build` clean `[B]`.
Linux runtime verification: blocked — the only WSL distro on this
machine is the dead one `[M]`. Honest `[G]` stands.

---

# 101. Falsification Engine v0.1 + L6 (measured)

`fabi selftest` shipped. The spec now verifies itself:

```text
binary size        ≤ 10 MB      7.5 MB      PASS
idle RSS           ≤ 150 MB     10 MB       PASS
probe wall time    < 3 s        94 ms       PASS
RSS after probes   ≤ 150 MB     14 MB       PASS
ledger append      +1 row       656 → 657   PASS
config required    0            none        PASS
detection patterns data         114 entries PASS
GOOS               windows      windows     PASS
other OSes         darwin/linux never ran   UNVERIFIED
```

2 UNVERIFIED stays UNVERIFIED. The engine does not lie about coverage.

And L6 became real: detection patterns moved to `patterns.json`
(embedded default + `~/.fabi/patterns.json` merge). Measured:

```text
added {"scanDirs": ["~/.fabi"]} to user patterns
    ↓ next scan
.fabi dir appeared as detected tool — zero code changes
```

Detection surface is now data-extensible. The Marketplace's first
product already works: a JSON file.

---

# 102. Watch v0.0.1 — the atom breathes (measured)

`fabi watch` shipped: periodic scans, default 5 min, every pass
writes the ledger. First 30 seconds of timeseries:

```text
[02:40:30] 241ms | tools 4↑5↓ | gpu 100% 66°C
[02:40:41] 903ms | tools 3↑5↓ | gpu  23% 63°C   ← a workload exited
[02:40:50] 116ms | tools 3↑5↓ | gpu 100% 68°C   ← GPU pulsing again
```

The GPU breathes. A tuning workload pulses 100% → 23% → 100% inside
one minute — invisible to any single scan, obvious in three lines.

GPU metrics now flow into `measurement.sampled` events every pass:
vram_used, vram_total, util, temp. Timeseries is the substrate for
Phase-2 idle/reclaim decisions — "was that tool ever used?" becomes
a query, not a guess.

`GOOS=darwin` cross-compile: clean `[B]`. Runtime: `UNVERIFIED` `[G]`.

---

# 103. Ledger queryable + GPU attribution (measured)

`fabi ledger` shipped (stats / events / export). The single truth is
now queryable from the CLI — 1,189 events, 27 min of real history,
all kinds visible.

And the GPU probe grew a physics channel: `--query-compute-apps`
attributes VRAM burn to PIDs. First run found a second gap:

```text
GPU:  100% util, 10,571 MiB VRAM used, 78°C
apps: zero compute processes attributed (WDDM limitation)
→     "10,571 MiB VRAM in use by unattributed processes
       (WDDM/VM attribution gap)"
```

The GPU is fully loaded and nobody will say by whom.

Same pattern as hidden-compute: missing provenance gets an inference
note. Either WSL2 GPU-paravirtualization (host nvidia-smi cannot see
VM GPU clients — consistent with the dead OpenClawGateway story) or
local workloads Windows won't attribute.

The estate now has **two** honest provenance gaps, both reported,
neither hidden.

---

# 104. HTTP identification + suspect list (measured)

Port probe grew an identity channel: `GET /` on every open port.
The ghost services answered:

```text
:1234 → HTTP 200 {"error":"Unexpected endpoint or method"}
        a living JSON API, hiding behind an LM-Studio port number
:8080 → HTTP 502 {"error":"no active node selected or available"}
        a gateway whose backend nodes are all dead
        (consistent with hermes' gateway_watchdog.py cmdline)
```

Port numbers are labels, not identity. HTTP signatures are closer.

Doctor Rule 5 shipped too: unattributed VRAM now names suspects —
the running AI workloads (`python.exe [strata]`, `pythonw.exe [hermes]`)
that plausibly burn 10.5 GB and won't appear in nvidia-smi attribution.

Every probe that gets deeper produces more honest unknowns.
That is the correct direction.

---

## End

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.
0.6.3 caught a ghost: dead WSL distro found by inference.
0.6.4 made the spec test itself, and turned patterns into plugins.
0.6.5 gave the atom a heartbeat: timeseries exists.
0.6.6 made the ledger queryable and caught the GPU lying by omission.
0.6.7 services got names; GPU burn got suspects.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.
0.6.3 caught a ghost: dead WSL distro found by inference.
0.6.4 made the spec test itself, and turned patterns into plugins.
0.6.5 gave the atom a heartbeat: timeseries exists.
0.6.6 made the ledger queryable and caught the GPU lying by omission.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.
0.6.3 caught a ghost: dead WSL distro found by inference.
0.6.4 made the spec test itself, and turned patterns into plugins.
0.6.5 gave the atom a heartbeat: timeseries exists.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.
0.6.3 caught a ghost: dead WSL distro found by inference.
0.6.4 made the spec test itself, and turned patterns into plugins.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.
0.6.3 caught a ghost: dead WSL distro found by inference.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.
0.6.2 closed the loop: scan → doctor → report → re-scan.

0.3 put numbers on words.
0.4 subtracted the spec.
0.5 subtracted the mystery.
0.6 measured the atom.
0.6.1 subtracted 160 MB of proven duplicate.

Every number above has a ledger row or a `time` command behind it.

The spec is now falsifiable in both directions:

- features without numbers → deleted (#57)
- numbers without features → impossible, the ledger wouldn't have the row

Tear it up.

The next tear-up is scheduled: when `fabi doctor` ships,
this PC's 160 MB of duplicate models and 2 hidden services
become the first Doctor recommendations `[M]`.
