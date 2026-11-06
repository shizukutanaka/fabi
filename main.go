package main

import (
	"fmt"
	"os"
	"time"
)

const version = "0.0.5-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "scan":
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runScan(time.Now(), false)
	case "watch":
		runWatch(os.Args[2:])
	case "doctor":
		runDoctor()
	case "report":
		runReport()
	case "selftest":
		runSelfTest()
	case "ledger":
		runLedgerCmd(os.Args[2:])
	case "route":
		arg := ""
		if len(os.Args) > 2 {
			arg = os.Args[2]
		}
		runRoute(arg)
	case "goal":
		runGoal(os.Args[2:])
	case "install":
		runInstall(os.Args[2:])
	case "uninstall":
		runUninstall(os.Args[2:])
	case "bench":
		// Deleted — Phase-1 proof, superseded by `fabi run --sweep`
		// (generative benchmarks across engines). Musk: delete beats add.
		fmt.Println("fabi bench removed — use `fabi run --sweep`")
	case "status":
		runStatus()
	case "run":
		runRun(os.Args[2:])
	case "serve":
		runServe()
	case "unload":
		runUnload(os.Args[2:])
	case "auto":
		runAuto()
	case "pattern":
		runPattern(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "why":
		runWhy(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("fabi", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("fabi — Universal AI Control Plane")
	fmt.Println("usage: fabi <command>")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  scan     detect AI tools, models, hardware; write ledger; print report")
	fmt.Println("  doctor   evidence-based recommendations from ledger history")
	fmt.Println("  report   monthly statement — estate trend + reclaimable, measured")
	fmt.Println("  selftest falsification engine — every claim gets a test")
	fmt.Println("  watch    periodic scan, timeseries history (default 5m)")
	fmt.Println("  ledger   query the event log (stats | events | export)")
	fmt.Println("  route    physics-derived routing check for a task")
	fmt.Println("  goal     purpose → tasks → feasibility, physics only")
	fmt.Println("  install  register watch at logon (--yes required)")
	fmt.Println("  uninstall remove watch task (ledger kept)")
	fmt.Println("  bench    (removed — use run --sweep)")
	fmt.Println("  status   one-line machine health from the ledger")
	fmt.Println("  run      execute a prompt on a live local model, measure real tok/s")
	fmt.Println("  serve    JSON API on :11435 (GET /status|/models|/ledger|/doctor · POST /run|/goal)")
	fmt.Println("  unload   free VRAM — unload loaded models (first real control action)")
	fmt.Println("  auto     apply safe recs only — idle unload, user-triggered (Automation seed)")
	fmt.Println("  pattern  extend detection via data — add/list user patterns (Marketplace seed)")
	fmt.Println("  diff     what changed between the last two scans (+appeared / -gone)")
	fmt.Println("  why      explain a recommendation — show its evidence chain")
	fmt.Println("  version  print version")
}
