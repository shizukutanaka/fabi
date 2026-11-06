# Fabi
## Universal AI Control Plane

**Version:** 0.4

**思考法:** First-Principles Thinking Method — Algorithm Order

**前版:** 0.3（数字を入れた版）
**今版:** 0.4（引き算をした版）

---

# 68. The Algorithm Is the Build Order

Musk's algorithm is not a metaphor.

It is the order Fabi builds in.

```text
1. Question every requirement
2. Delete the part or process
3. Simplify / optimize
4. Accelerate cycle time
5. Automate
```

Order is law.

```text
Automate before delete = build waste faster.
Simplify before delete = polish what should die.
```

0.1/0.2 violated this order: 19 boxes, 6 databases, 11 API methods, 4 phases of features.

0.3 fixed the numbers.

0.4 fixes the order.

---

# 69. Requirement Audit

Every component answers one question:

**Who demanded this, and what breaks without it?**

Nobody demanded it + nothing breaks → delete candidate.

| Component (v0.2) | Demanded by | Breaks without it? | Verdict |
|---|---|---|---|
| Observe (Discovery, Monitor) | User | Yes — nothing else has data | KEEP — the atom |
| Think (Goal, Planner) | Spec | No — heuristics route fine at MVP | DEFER to Phase 2 |
| Act (Task, Automation) | Spec | No — read-only Fabi already saves money | DEFER until Guard exists |
| Remember (Resource, Object, Knowledge) | Spec | No — filesystem + SQLite suffice | SIMPLIFY to one DB |
| Guard (Permission, Policy) | Safety | Yes, once Act exists; cheap now | KEEP minimal — deny-by-default |
| Tell (Event Bus) | Spec | No — log stream is an event bus | SIMPLIFY to ledger |
| Ship (Marketplace, SDK, API) | Ecosystem | No — CLI is already the API | DEFER SDK to 10K installs |

Audit result:

```text
Phase 1 Fabi = Observe + Guard(min) + Ledger
```

Everything else waits for a number that demands it.

---

# 70. Deletion List

What 0.4 deletes from the spec:

| Deleted | Replaced by | Reason |
|---|---|---|
| 6 internal DBs | 1 Ledger (SQLite) | Same data, one truth (#53 already said it) |
| Knowledge Graph | Materialized view over Ledger | Graph is a query, not a database |
| Universal API 11 methods | 4 verbs: `scan`, `status`, `run`, `ledger` | API-first means CLI-first; rest is composition |
| Goal Engine (LLM) | Constraint Solver + heuristics | LLM planner needs data that doesn't exist yet |
| Task Engine states | Ledger states | State machine is a view over events |
| Multi-PC / PAIR / exo / GPUStack | Nothing until 1M installs | Distribution before federation |
| GUI | TUI + generated views | UI is 5%; hand-built GUI is 20% of effort |
| "AI Explorer" as product | `fabi scan` output | Explorer is a report, not an app |

Deletion is not loss.

Deletion is the 50% budget going to data (#50).

---

# 71. The Atom

Minimum Fabi, irreducible:

```text
fabi scan
    ↓
detect AI tools, models, hardware
    ↓
measure: VRAM, tok/s, disk, cost
    ↓
ledger: one SQLite file
    ↓
report: numbers a human can read
```

That is all of Phase 1.

Not a reduced version.

The atom.

Everything in 0.1–0.3 is either:

- inside the atom, or
- a plugin that must pass its own four questions (#62), or
- deleted.

Test: remove any Phase-1 feature and ask "does scan still tell the truth?"

If yes → plugin, not core.

---

# 72. Deletion Budget

New law:

```text
Release surface area may grow ≤10%
    ↓
unless it deletes ≥ what it adds
```

Surface area = CLI verbs + plugin API + config keys + drivers.

Net complexity must trend to zero.

A release that only adds is a regression wearing a version number.

Measured by: `fabi doctor --complexity` reports surface area per release.

Target: v1.0 surface < v0.4 surface × 1.5, with 10× the capability.

If capability doesn't grow 10× while surface grows 1.5×, the growth was waste.

---

# 73. One Database

0.2 had six databases:

Resource DB / Capability DB / Object DB / Knowledge DB / Policy DB / Driver DB.

Six databases is six truths.

One truth: the Ledger.

```text
Ledger (SQLite, append-only events)
    ↓
Resource view   = query
Capability view = query
Object view     = query
Knowledge view  = materialized graph, rebuilt nightly
Policy view     = query + signature check
Driver view     = query
```

All six are views.

Views can be wrong; they rebuild.

The ledger cannot be wrong; it only appends.

Rule: no Fabi component may keep state the Ledger doesn't have.

State outside the Ledger is a second truth.

Second truth is deleted.

---

# 74. Cycle-Time Numbers

0.3 says "iteration is 5 min, not 1 week."

0.4 makes it a budget:

| Loop | Budget | Over budget means |
|---|---|---|
| Edit → test | 60 s | Build too big → split |
| Test → delete decision | 24 h | Feature didn't earn its test → off |
| Release → measurement | 7 days | Number missing → claim was a rumor |
| Quarter → roadmap tear-up | 90 days | Roadmap is hope → rewrite |

Automate-last rule:

No automation ships until its manual loop has run ≥10 times.

A macro for a process that ran twice is waste with extra steps.

---

# 75. Automate Last — What Ships First

The only automation allowed in Phase 1:

1. **Falsification Engine** (#57) — automates deletion of lies.
2. **Discovery probes** — automates observation.
3. **Ledger writer** — automates truth.

Everything else — routing, planning, fixing — is manual or absent until Phase 2+.

Rationale: automation amplifies whatever it touches.

Automating a bad plan = wrong answers at machine speed.

Automating deletion = waste removed at machine speed.

Fabi automates subtraction first.

That is the Musk order.

---

# 76. Distribution Physics

150M installs is not a marketing goal.

It is a physics problem.

```text
installs = channels × conversion × retention
```

## Channels — the only ones that scale

| Channel | Scale ceiling | Tax |
|---|---|---|
| winget / brew / apt / scoop | 100M+ | signing, review, updates |
| OEM / preinstall | 1B+ | needs brand at 1M+ first |
| Website download | ~1M | support burden, no auto-update |
| Word of mouth | unbounded | needs a number worth showing |

Website download is deleted as a primary channel at 10K+ installs.

Package-manager presence is a Phase 1 exit criterion, not Phase 3.

Signed, notarized, auto-updating — or it doesn't ship.

## Conversion

```text
download → install → first number on screen
```

Budget: 60 s total, 0 config (#59).

Every config screen is a conversion leak.

Leaks are measured: funnel events go to the Ledger.

## Retention

Monthly statement, auto-generated:

```text
Fabi saved you this month:
  ¥4,320 cloud spend avoided
  11.2 GPU-hours not burned
  3 tools you didn't need
```

If the statement shows ≤0 for 2 consecutive months → churn predicted → Doctor interviews the user.

Retention metric = statement value > 0.

Not DAU. Not session length. Yen.

---

# 77. Honest Numbers

0.3 numbers are targets, not facts.

0.4 tags every number:

| Tag | Meaning |
|---|---|
| `[M]` measured | Comes from the Ledger on a real PC |
| `[B]` benchmarked | Comes from CI on reference hardware |
| `[A]` assumed | Physics estimate, unverified |
| `[G]` guess | Nobody knows yet |

Rules:

- `[A]` and `[G]` numbers cannot appear in marketing.
- `[M]` numbers must link to a Ledger query.
- Unverified numbers are hypotheses.
- Hypotheses are allowed in the spec.
- Hypotheses are not allowed in the marketing.

## Corrections to 0.3

| 0.3 claim | 0.4 status |
|---|---|
| Size ≤10 MB | `[A]` — core only; drivers lazy-load, total ≤50 MB |
| RAM idle ≤150 MB | `[B]` achievable in Rust/Go + SQLite; verify on 3 OSes |
| Discovery recall 95% | `[A]` — needs a 200-tool ground-truth corpus first |
| Local tokens ≥70% | `[A]` — only valid on GPU machines; 0-GPU target is cost, not locality |
| 150M installs by 2035 | `[G]` — distribution-bound; weekly install velocity is the real metric |
| Install ≤60 s, 0 config | `[B]` — already MVP law, keep |

Marketing that survives degradation is a lie (#57).

So is marketing that precedes measurement.

---

# 78. Doctor as Subtractor

0.2 Doctor fixes.

0.4 Doctor primarily subtracts.

Monthly Doctor report:

```text
SAFE TO REMOVE:
  ollama model "llama3.1:70b" — 40 GB, last used 94 days ago
  LM Studio — 2 versions installed, newest unused 60 days
  3 duplicate GGUFs of qwen2.5:7b — keep newest

SAFE TO DISABLE:
  ComfyUI auto-start — 0 runs in 30 days, 1.2 GB RAM idle
  Docker Desktop — starts on boot, 0 containers in 90 days
```

Every deletion candidate carries evidence from the Ledger.

No evidence → no recommendation.

Doctor's best output is the thing the user didn't know they could delete.

"The best code is no code."

"The best AI tool is the one you didn't need to install."

Fabi's value is measured in subtractions.

---

# 79. What 0.4 Does Not Do

- Does not add features.
- Does not add engines.
- Does not add databases.
- Does not add API surface.
- Does not promise 2035 harder.

It subtracts.

If 0.4 were applied to the codebase today, the diff would be mostly red.

That is the point.

---

# 80. Roadmap v4

| Phase | Contains | Exit number |
|---|---|---|
| 0 — this week | `fabi scan` on real PC, ledger on, <10 MB core | 60 s install, 0 config, numbers on screen |
| 1 — MVP | Scan + Monitor + Doctor(subtract) + Ledger + falsification | 10K installs, 95% recall `[B]` |
| 2 | Constraint Solver, heuristic routing, Regret Table | cloud spend ÷2 vs baseline `[M]` |
| 3 | Package managers (winget/brew/apt), signed builds | 100K installs |
| 4 | SDK, Marketplace, plugins | 1M installs |
| 5 | Federation, cluster, mesh | only if Phase 4 retention >80% `[M]` |

Phases 4–5 exist in the spec but not in the plan.

Plan is numbers. Spec is options.

---

# 81. The Subtraction Spec

```text
0.1 added vision.
0.2 added philosophy.
0.3 added numbers.
0.4 subtracts.
```

Vision without numbers is a wish.

Numbers without subtraction is a ledger of waste.

Fabi's physics budget applies to Fabi itself:

- Cost of Fabi to build = surface area
- Value of Fabi = deletions it enables + yen it saves
- Surface grows faster than value → tear it up

Version 0.4 is smaller than 0.3 on purpose.

Not because less is more.

Because more was never measured.

---

## End

0.3 ended with:

> Make the number come out.

0.4 ends with:

> Make the number go down — surface area, cost, RAM, cloud spend, regret.

Numbers that go down are proof.

Proof is the product.
