# Fabi
## Universal AI Control Plane

**Version:** 0.3

**思考法:** First-Principles Thinking Method（Elon・Musk Thinking Method）

---

# 46. First-Principles Engine

FabiのPlannerは思考法。

思考法名は内部法実施。

---

## Method

```text
Question
    ↓
Physics
    ↓
Math
    ↓
Cost
    ↓
Constraint
    ↓
Reconstruct upward
    ↓
Number
    ↓
Proof
```

---

## Prohibited

```text
Analogy
    ↓
Plan
    ↓
Hope
```

Hope is not a plan.

---

## Required

```text
Truth
    ↓
Constraint
    ↓
Plan
    ↓
Number
    ↓
Test
```

---

## Three Questions

GoalはPlannerへ三問渡前.

Q1. 物理は、不可能で做什么。

Q2. 成本floorは。

Q3. 10x不做，做itをworth-doingは。

三回答のConstraint Setへconvert becomes.

Constraint SetがTask Graphへconvert becomes.

---

# 47. Spec Law

Spec no numberはwrong。

Tear it up.

「Dynamic routing」はnot a spec.

「Routingはfirst-pass候选12件listからselect, decide in 250 ms」is a spec.

---

## Mandatory Measure

| Vague | Measured |
|---|---|
| ローカルファースト | 70%+ of tokens are local-served |
| 最適化 | Cost is 1/10 of baseline, time is 1/3 |
| 自動detection | 95%-recall, cold 3 s, RAM 150 MB |
| 修復candidate | 100%-safe-apply, 80%-fix-rate |
| 意味understanding | NDCG@10 > 0.8 on own benchmark |
| Enterprise対応 | Policy-enforcement 100%, 0 bypass |
| 世界standard | 150M installs, 0 config |

---

## Law

No number → Feature off.

Number → Test auto-runs.

Test fail → Feature off.

Silent degradation is allowed.

Silent marketing is not allowed.

---

# 48. Physics Budget

## The physics that binds

Inference is bandwidth-bound on decode.

```text
tokens/sec ≈ model bytes resident in memory ÷ memory bandwidth
```

Prefill is compute-bound.

```text
FLOPs ≈ 2 × parameters × prompt tokens
```

---

## The numbers that bind

| Model | Resident bytes | Bandwidth | tok/s |
|---|---|---|---|
| 3B Q4 | 1.9 GB | 200 GB/s (M-class) | ~105 |
| 8B Q4 | 4.7 GB | 200 GB/s | ~42 |
| 8B Q4 | 4.7 GB | 180 GB/s (RTX-class) | ~38 |
| 8B Q4 | 4.7 GB | 15 GB/s (CPU 4T) | ~3 |
| 70B Q4 | 40 GB | 180 GB/s | ~4.5 |

---

## The physical consequence

```text
1 turn
    ↓
1,000 output tokens
    ↓
5 sec
    ↓
need 200 tok/s
```

Single local 8B = impossible.

Fabiをfollowing does:

- speculative decoding
- draft-model parallel
- local + cloud hybrid
- prefix-cache
- model routing by physics, not by name

---

## The budget that binds

| Item | Floor |
|---|---|
| Local inference cost | Electricity only, ¥0 |
| Cloud inference | Per-1K-token price, logged |
| VRAM | 0 GB to 24 GB, all sizes work |
| RAM | 150 MB idle |
| Cold start | 3 s |
| Install | 60 s, no config |
| Offline | Core runs, cloud off |

If the product fails any of these, the physics budget is broken.

Broken budget → Doctor proposes the repair.

---

# 49. Cost Law

```text
Value × 10
    ↓
Cost ÷ 10
```

Not allowed:

```text
Cost × 10
Value + 1
```

---

## Per-user monthly, measured

| Item | Today | Fabi target |
|---|---|---|
| Cloud LLM spend | ¥12,000 | ¥1,200 |
| GPU-hours | 40 h | 4 h |
| Model-download | 30 GB | 6 GB |
| Time-to-first-result | 180 s | 20 s |
| Setup time | 4 h | 0 h |

Setup time zero is the biggest cost.

Setup time zero is also the biggest value.

Same number, both sides.

---

# 50. Allocation Law

20% of what people make yields 80% of value.

Fabi's resource allocation is therefore not UI.

---

## Budget

```text
Discovery + Knowledge + Proof .... 50%
Engines ........................ 30%
Drivers ........................ 15%
UI ............................ 5%
```

UI is 5%.

That is the law, not the preference.

---

# 51. Regret Table

Every Goal records the following:

- what was actually used
- what was actually paid
- what was actually discarded
- what was actually slower than last time

A plan that doesn't lose is wrong.

Iteration is as fast as possible.

Iteration is 5 min, not 1 week.

---

# 52. Time-Box

The goal is a number.

Not a wish.

2035 is the end. Today is the box.

---

## Back-cast

| Year | Must be true |
|---|---|
| 2035 | 150M installs, 0 config, 100% policy |
| 2030 | 15M installs, SDK stable, OS-native package |
| 2027 | 1M installs, Marketplace, offline core |
| 2026 | 100K installs, MVP works on all 3 OSes |
| This week | Discovery works on real PC, <10 MB, 60 s install |

---

## The week's rule

```text
This week's build
    ↓
real PC
    ↓
no config
    ↓
Discovery works
    ↓
numbers on screen
```

If this week's build doesn't run this way, 2035 is unreachable.

2035 is not postponed.

This week's plan is torn up.

---

# 53. Tear-Up Clause

#5 Architecture has 19 boxes.

19 boxes are not 19 engines.

Boxes must earn existence.

---

## 7 Primitives

| Primitive | Absorbed boxes | Why it exists |
|---|---|---|
| Observe | Discovery, Monitor, Doctor | Data is half the work (#50) |
| Think | Goal, Planner, Capability, Scheduler | Constraint to decision |
| Act | Task, Automation, Driver, Plugin | Change |
| Remember | Resource, Object, Knowledge | Graph of the world |
| Guard | Permission, Policy | No harm |
| Tell | Event Bus | Truth between parties |
| Ship | Marketplace, SDK, Universal API | Outward |

Object Manager and Task Engine and Knowledge Graph merge.

Reason: same data, different view.

Two views of one graph beat two databases.

---

## The box's four questions

Every box must answer:

1. What is it measuring, and in what unit
2. What is the physical limit
3. What is the cost floor
4. If it's not 10x, is it worth doing

Box that can't answer = delete.

Delete is allowed.

19 → 7 is the current answer.

---

# 54. Constraint Graph

Planner does not only output a Task Graph.

Planner outputs the Constraint Set first.

```text
Constraint Set
    ↓
physics: VRAM, tok/s, bandwidth
    ↓
cost: yen, GPU-hours
    ↓
policy: permission, locale
    ↓
time: deadline
    ↓
feasible?
    ↓ yes
Task Graph
    ↓ no
Tear up the Goal
```

Infeasible tasks are cut, not warned.

Physics first, then plan.

That is the physics-first Planner.

---

# 55. Method to Law

| Musk's thought | Fabi's law |
|---|---|
| First principles | First-Principles Engine (#46) |
| God-tier spec sheet | Spec Law (#47) |
| Cost and value, tears in the photo | Cost Law (#49) |
| Allocation | Allocation Law (#50) |
| Regret, iterate fast | Regret Table (#51) |
| Timeboxing | Time-Box (#52) |
| If it's not 10x, don't do it | 10x Rule |
| Don't optimize the wrong metric | Measure, then optimize |
| Don't invent a law that doesn't exist | Falsification Engine |

Fabi is the container that makes these enforceable.

Law is not a poster.

Law is a test that runs.

---

# 56. Capability Spec Sheet (God-tier)

| Capability | Unit | Target | Floor | VRAM |
|---|---|---|---|---|
| OCR | 1 page | 1.5 s | 0.6 s | 2 GB |
| Translation | 1,000 words | 4 s | 1.5 s | 0 (local 3B) |
| Reasoning | 1,000 tok out | 5 s | 2 s | 6 GB |
| Code review | 2,000 LOC | 12 s | 4 s | 6 GB |
| Search | 1M docs | 0.3 s/query | 0.08 s | 4 GB |
| Image gen | 1 image 1024² | 6 s | 2 s | 12 GB |
| Video understanding | 60 s clip | 25 s | 8 s | 16 GB |
| Speech | 60 s audio | 8 s | 3 s | 4 GB |
| Planning | 12-step graph | 1.2 s | 0.4 s | 4 GB |

Numbers are the promise.

Test is the promise.

Promise without test = off.

---

# 57. Falsification Engine

Every claim emits an auto-test.

```text
Claim
    ↓
Test generated
    ↓
CI runs on real PC
    ↓
pass / fail
    ↓
fail → Feature disabled, log written
```

The product degrades honestly.

The spec degrades honestly.

The marketing does not degrade.

Marketing that survives degradation is a lie.

Lie is deleted at once.

---

## Claim format

```text
"<feature> does <action> in <number><unit> on <hardware> with <budget>"
```

String that can't fill that template is not a claim.

Feature doesn't exist.

---

# 58. Vaporware List

Fabi does not promise:

- 100% automation
- Autonomous agent that doesn't ask permission
- An OS that's replaced
- A ChatGPT competitor
- An LLM that lives inside Fabi
- A subscription that unlocks features
- "It just works" without numbers

Those words don't appear in Fabi.

---

# 59. MVP God-tier Spec

| # | Promise | Unit | Target | Test |
|---|---|---|---|---|
| 1 | Discovery finds AI tools | recall | 95% | 3 real PCs |
| 2 | Discovery is cold | seconds | 3 | measured |
| 3 | Discovery RAM | MB | 150 | measured |
| 4 | Size | MB | 10 | package |
| 5 | Install | seconds | 60 | measured, no config |
| 6 | Works offline | core | 100% | network off |
| 7 | 3 OSes | count | 3 | CI |
| 8 | Model name awareness | user count | 0 | survey |
| 9 | Permission default | deny | 100% | CI |
| 10 | One number on screen | count | ≥1 | UI test |

User model-name awareness = zero is the MVP's hardest number.

If one name appears, MVP is broken.

---

# 60. Proof-of-Work Ledger

Every action logs:

- input
- output
- bytes read
- bytes written
- GPU used
- token count
- yen
- regret

Ledger is public.

Ledger is queryable.

Ledger is the source of the numbers.

Numbers that don't come from the Ledger are forbidden.

---

# 61. The One Number

World-standard = install count.

2035: 150M.

Today: 10K.

Gap: 15,000x.

Gap-closing mechanism is not the SDK.

Not the Marketplace.

Not the blog.

Mechanism is: it runs, and asks for nothing.

```text
0 config
    ↓
1 minute
    ↓
it works
    ↓
numbers show themselves
```

That mechanism is the one number.

If this week's build can't be installed in under 60 seconds, 15,000x doesn't close.

---

# 62. Four-Question Loop

Every box, every release, every PR:

```text
1. What is it measuring and in what unit
2. What is the physical limit
3. What is the cost floor
4. If it's not 10x, is it worth doing
```

4 answers exist → Ship.

4 answers don't → Delete.

Delete beats defer.

Defer is the sign of hope.

Hope is not a plan.

---

# 63. Design Principles v2

#24 rewritten as falsifiable numbers.

| v1 | v2 |
|---|---|
| ローcal-first | 70%+ of tokens served locally |
| 零vendor lock-in | 100% works with 0 cloud APIs |
| AI-model independence | 0 hardcoded model names |
| OS independence | 3 OSes, 1 codebase, CI |
| Plugin-first | 80% of features are plugins, 20% are core |
| SDK-first | SDK ≥ core API |
| API-first | 0 feature has GUI-only path |
| Offline-first | core 100% offline |
| Connect, not replace | 0 AI software wrapped, only probed |
| Goal-only | user model-name awareness 0 |

---

# 64. Ultimate Goal as a Number

2035:

```text
PC population ......... 1.5B
installs .............. 150M (10%)
config required ....... 0
policy compliance ..... 100%
local token share ..... ≥70%
median user time-to-outcome .... 20 s
cloud yen / user / month ....... ≤ ¥1,200
```

That is the goal.

Not the sentence.

The sentence is the memo.

The number is the plan.

---

# 65. Final Vision v2

Windows has Explorer, Task Manager, Device Manager.

AI-era Windows lacks AI Explorer, AI Planner, AI Runtime, AI Policy Manager.

Fabi supplies the missing layer.

Layer is not an app.

Layer is measured by install count, by setup time, by policy.

```text
install count ↑
setup time  → 0
policy compliance → 100%
```

That is the standard.

Standard is the number, not the claim.

---

# 66. The Law of the Fabi Spec

```text
1. 物理is the truth.
2. 物理is not the suggestion.
3. Spec has a number.
4. Number has a test.
5. Test fails → Feature off.
6. Cost ÷ 10.
7. Value × 10.
8. 50% of resources go to data.
9. UI is 5%.
10. 2035 is not pushed back.
11. This week's plan is torn up.
12. Box that can't answer the four questions is deleted.
13. Delete beats defer.
14. Defer is the sign of hope.
15. Hope is not a plan.
```

---

# 67. Roadmap Diff (0.2 → 0.3)

## Phase 1 (MVP) = 10 promises of #59

Add:

- Physics Budget measured
- Proof-of-Work Ledger on
- Falsification Engine on

## Phase 2

- Constraint Graph Planner (#54)
- 7 primitives, not 19 boxes (#53)
- Regret Table (#51)

## Phase 3

- SDK ≥ core (#63)
- Marketplace numbers-gated

## Phase 4

- 150M installs is the goal, not the slogan (#64)

---

## End

Version 0.3 is the version with numbers.

Version 0.2 is the version with words.

Version 0.3 is not the improvement of Version 0.2.

Version 0.3 is the improvement of itself.

Tear it up.

Make the number come out.
