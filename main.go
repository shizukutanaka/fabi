package main

import (
	"fmt"
	"os"
	"time"
)

const version = "0.0.6-dev"

func main() {
	if len(os.Args) < 2 {
		// Context-aware default: a first-timer needs "what is this +
		// one action", a returner needs the one-line status. The ledger
		// knows which — zero scans means they've never run fabi.
		l := openLedger()
		var scans int64
		if l.db != nil {
			l.db.QueryRow("SELECT COUNT(*) FROM events WHERE kind='scan.started'").Scan(&scans)
		}
		l.close()
		if scans == 0 {
			fmt.Println("fabi — the honest eye on your PC's AI estate")
			fmt.Println()
			fmt.Println("  Run `fabi scan` to see what's here.")
			fmt.Println("  Run `fabi help` for all commands.")
			return
		}
		runStatus()
		fmt.Println()
		fmt.Println("commands: scan · doctor · run · serve · help")
		return
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
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runDoctor()
	case "report":
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
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
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
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
		var dargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				dargs = append(dargs, a)
			}
		}
		runDiff(dargs)
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
	fmt.Println("fabi — the honest eye on your PC's AI estate")
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
