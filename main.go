package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

const version = "0.0.7-dev"

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
			fmt.Println(tr("tagline"))
			fmt.Println()
			fmt.Println(tr("firstrun.scan"))
			fmt.Println(tr("firstrun.help"))
			return
		}
		runStatus()
		fmt.Println()
		fmt.Println(tr("hint.commands"))
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
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runSelfTest()
	case "ledger":
		runLedgerCmd(os.Args[2:])
	case "route":
		arg := ""
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				arg = a
			}
		}
		runRoute(arg)
	case "goal":
		var gargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				gargs = append(gargs, a)
			}
		}
		runGoal(gargs)
	case "install":
		var iargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				iargs = append(iargs, a)
			}
		}
		runInstall(iargs)
	case "uninstall":
		var xargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				xargs = append(xargs, a)
			}
		}
		runUninstall(xargs)
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
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runRun(os.Args[2:])
	case "serve":
		runServe()
	case "unload":
		var uargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				uargs = append(uargs, a)
			}
		}
		runUnload(uargs)
	case "auto":
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runAuto()
	case "pattern":
		var pargs []string
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			} else {
				pargs = append(pargs, a)
			}
		}
		runPattern(pargs)
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
	case "update":
		for _, a := range os.Args[2:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		runUpdate()
	case "help", "-h", "--help":
		usage()
	case "version", "--version", "-v":
		fmt.Println("fabi", version, runtime.GOOS+"/"+runtime.GOARCH, "lang:"+detectLang())
	case "lang":
		code, src := detectLangSource()
		var langs []string
		for k := range messages {
			langs = append(langs, k)
		}
		sort.Strings(langs)
		fmt.Println("detected:", code, "(source:", src+")")
		fmt.Println("supported:", strings.Join(langs, " "))
		fmt.Println("override:  set FABI_LANG=<code> — e.g. set FABI_LANG=en")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(tr("tagline"))
	fmt.Println(tr("usage.line"))
	fmt.Println()
	fmt.Println(tr("usage.cmds"))
	fmt.Println("  scan     " + tr("cmd.scan"))
	fmt.Println("  doctor   " + tr("cmd.doctor"))
	fmt.Println("  report   " + tr("cmd.report"))
	fmt.Println("  selftest " + tr("cmd.selftest"))
	fmt.Println("  watch    " + tr("cmd.watch"))
	fmt.Println("  ledger   " + tr("cmd.ledger"))
	fmt.Println("  route    " + tr("cmd.route"))
	fmt.Println("  goal     " + tr("cmd.goal"))
	fmt.Println("  install  " + tr("cmd.install"))
	fmt.Println("  uninstall " + tr("cmd.uninstall"))
	fmt.Println("  bench    " + tr("cmd.bench"))
	fmt.Println("  status   " + tr("cmd.status"))
	fmt.Println("  run      " + tr("cmd.run"))
	fmt.Println("  serve    " + tr("cmd.serve"))
	fmt.Println("  unload   " + tr("cmd.unload"))
	fmt.Println("  auto     " + tr("cmd.auto"))
	fmt.Println("  pattern  " + tr("cmd.pattern"))
	fmt.Println("  diff     " + tr("cmd.diff"))
	fmt.Println("  why      " + tr("cmd.why"))
	fmt.Println("  update   " + tr("cmd.update"))
	fmt.Println("  version  " + tr("cmd.version"))
}
