# Fabi Serve API — v0

`fabi serve` exposes a loopback-only JSON API on `127.0.0.1:11435`.
No auth (loopback = same trust as probes). Any local agent or tool
can call fabi — this is the Universal API seed (SPEC #21).

Start it:

```sh
fabi serve    # blocks; Ctrl+C to stop
```

## Endpoints

### GET /status

Current estate summary (from latest scan in the ledger).

```json
{"running":5,"installed_idle":5,"services":3,"models":8,"scans":28,"version":"0.0.2-dev"}
```

### GET /models

Live local generative models (ollama passthrough).

```json
{"models":["llama3.2:1b","qwen2.5:0.5b"]}
```

### POST /run

Run a prompt on a local model. Body:

```json
{"prompt":"say hi","model":"qwen2.5:0.5b"}
```

`model` optional — defaults to the smallest/fastest live model.

```json
{"model":"qwen2.5:0.5b","response":"Hello!","tok_s":200.4,"wall_s":0.13,"eval_count":9}
```

### POST /goal

Decompose a purpose into tasks; optionally execute the chain.

```json
{"goal":"research local ai","run":true}
```

```json
{"goal":"research local ai","tasks":["search","reasoning"],
 "steps":[{"task":"search","model":"qwen2.5:0.5b","tok_s":190.1,"output":"..."}],
 "executed":2}
```

`run:false` (default) returns decomposition only — no execution.

### POST /unload

Free VRAM — unloads every loaded model (`keep_alive:0`). Returns
what was freed; recorded as `model.unloaded` with `via: "serve"`.

```json
{"unloaded":["qwen2.5:0.5b"],"vram_freed_bytes":483645193,"count":1}
```

### POST /scan

Trigger a fresh observation — runs the full 6-probe pipeline and
returns the new estate summary. Findings land in the ledger.

```json
{"installed_idle":5,"models":8,"running":5,"scans":30,"services":4}
```

### GET|POST /pattern

Extend detection through data — the Marketplace seed. POST adds a
user pattern; GET lists current user patterns.

```sh
curl -X POST /pattern -d '{"field":"exe","value":"myserver.exe"}'
curl -X POST /pattern -d '{"field":"port","value":"mysvc:9000"}'
```

```json
{"added":"myserver.exe","field":"exe"}
```

fields: `exe | cmd | scripthost | dir | modelroot | pkgtool | pkgruntime | port`
(port value format: `name:port`). Writes `~/.fabi/patterns.json`,
which merges on the next scan — same store as `fabi pattern`.

### GET /doctor

Latest open recommendations per target (investigate / idle / remove).

```json
{"count":8,"recommendations":[{"class":"investigate","target":"hidden compute","detail":"..."}]}
```

### GET /diff

Estate delta between the last two scans — what appeared or
disappeared. Machine-readable version of `fabi diff`.

```json
{"from":...,"to":...,"added":[{"kind":"service","name":"ollama-alt"}],"removed":null}
```

### GET /why?target=X

Evidence chain for a recommendation — why fabi says what it says.
Returns the recommendation plus the ledger events that justify it.

```json
{"target":"gpu burn","class":"investigate","confidence":"info — ...",
 "evidence":[{"id":2492,"kind":"resource.found","ts":...}]}
```

### GET /ledger?kind=&n=

Query the append-only event ledger. Params: `kind` (filter by event
kind), `n` (max events, default 20, cap 200). Returns newest first.

```sh
curl "http://127.0.0.1:11435/ledger?kind=task.executed&n=5"
```

```json
{"count":5,"events":[{"id":1,"ts":...,"kind":"task.executed","payload":{...}}]}
```

### POST /import

Merge an exported estate (JSONL body — `fabi ledger export` output)
into this ledger. API parity with `fabi ledger import`: exact
duplicates are skipped, original ts/payload preserved, and a
`ledger.imported` provenance event records the imported id range.

```sh
curl -X POST /import --data-binary @estate.jsonl
```

```json
{"imported":142,"duplicates_skipped":3,"malformed":0}
```

### GET /update

Honest version check — API parity with `fabi update`. Queries GitHub
releases once and reports the version delta; `502` when GitHub is
unreachable. Never auto-downloads.

```json
{"current":"0.0.6","latest":"0.0.5","behind":false,"url":"https://.../v0.0.5"}
```

### GET /route?task=summarize

Physics-derived routing verdict — API parity with `fabi route <task>`.
`400` when `?task=` is missing or unknown.

```json
{"task":"summarize","model_class":"8b","gpu":"NVIDIA RTX A2000 12GB",
 "vram_mib":11514,"bandwidth_gbps":288,"tok_s":61.3,"needed_tok_s":150,
 "latency_s":9.8,"target_s":4,"fit_vram":true,"verdict":"LOCAL POSSIBLE"}
```

## Notes

- Everything the API does is also recorded back to `~/.fabi/ledger.db`
  (`task.executed` with `via: "serve"`) — the ledger stays the one truth.
- Errors: `400` bad request, `404` unknown endpoint, `502` upstream
  (model/engine) failure.
- Loopback only by design — no external exposure.
