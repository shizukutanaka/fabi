package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Pattern lists are data, not code — the first capability pack (SPEC v0.6, L6).
// Defaults are embedded; ~/.fabi/patterns.json merges user entries on top.

//go:embed patterns.json
var defaultPatterns []byte

type patternSet struct {
	ExePatterns []string     `json:"exePatterns"`
	CmdPatterns []string     `json:"cmdPatterns"`
	ScriptHosts []string     `json:"scriptHosts"`
	Ports       []portTarget `json:"ports"`
	ScanDirs    []string     `json:"scanDirs"`
	ModelRoots  []string     `json:"modelRoots"`
	PkgAITools  []string     `json:"pkgAITools"`
	PkgRuntimes []string     `json:"pkgRuntimes"`
}

var activePatterns *patternSet

func loadPatterns() *patternSet {
	if activePatterns != nil {
		return activePatterns
	}
	ps := &patternSet{}
	if err := json.Unmarshal(defaultPatterns, ps); err != nil {
		fmt.Fprintf(os.Stderr, "fabi: embedded patterns broken: %v\n", err)
	}
	// User override merges: user's entries append to defaults.
	// A malformed user file must not be silently ignored — L2:
	// report the gap, don't guess.
	dir, err := ledgerDir()
	if err == nil {
		path := filepath.Join(dir, "patterns.json")
		if b, err := os.ReadFile(path); err == nil {
			var user patternSet
			if err := json.Unmarshal(b, &user); err == nil {
				ps.ExePatterns = append(ps.ExePatterns, user.ExePatterns...)
				ps.CmdPatterns = append(ps.CmdPatterns, user.CmdPatterns...)
				ps.Ports = append(ps.Ports, user.Ports...)
				ps.ScanDirs = append(ps.ScanDirs, user.ScanDirs...)
				ps.ModelRoots = append(ps.ModelRoots, user.ModelRoots...)
				ps.PkgAITools = append(ps.PkgAITools, user.PkgAITools...)
				ps.PkgRuntimes = append(ps.PkgRuntimes, user.PkgRuntimes...)
			} else {
				// Honest-eye: the user's patterns are broken — say so,
				// don't pretend the file doesn't exist.
				fmt.Fprintf(os.Stderr,
					"fabi: %s is malformed (%v) — user patterns ignored\n",
					path, err)
			}
		}
	}
	activePatterns = ps
	return ps
}

// runPattern — Marketplace seed: users extend detection through data,
// not code. `fabi pattern add <field> <value>` writes to
// ~/.fabi/patterns.json, which loadPatterns merges on next scan.
//
//	fields: exe | cmd | scripthost | dir | modelroot | pkgtool |
//	        pkgruntime | port (name:port e.g. "mysvc:9000")
func runPattern(args []string) {
	dir, err := ledgerDir()
	if err != nil {
		fmt.Println("fabi pattern:", err)
		return
	}
	path := filepath.Join(dir, "patterns.json")

	var user patternSet
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &user)
	}

	if len(args) == 0 || args[0] == "list" {
		if jsonOut {
			out, _ := json.MarshalIndent(user, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Println(tr("pattern.hdr"))
		fmt.Println("─────────────────────────────────────────────")
		empty := true
		show := func(name string, xs []string) {
			for _, x := range xs {
				fmt.Printf("  %-10s %s\n", name, x)
				empty = false
			}
		}
		show("exe", user.ExePatterns)
		show("cmd", user.CmdPatterns)
		show("scripthost", user.ScriptHosts)
		show("dir", user.ScanDirs)
		show("modelroot", user.ModelRoots)
		show("pkgtool", user.PkgAITools)
		show("pkgruntime", user.PkgRuntimes)
		for _, p := range user.Ports {
			fmt.Printf("  %-10s %s:%d\n", "port", p.Name, p.Port)
			empty = false
		}
		if empty {
			fmt.Println(tr("pattern.none"))
		}
		return
	}

	if args[0] == "install" && len(args) == 2 {
		installPatternPack(args[1], path, &user)
		return
	}

	if args[0] != "add" || len(args) != 3 {
		fmt.Println("usage: fabi pattern add <exe|cmd|scripthost|dir|modelroot|pkgtool|pkgruntime|port> <value>")
		fmt.Println(tr("pattern.hint.install"))
		fmt.Println(tr("pattern.example1"))
		fmt.Println(tr("pattern.example2"))
		return
	}
	field, value := args[1], args[2]
	if err := addPatternField(&user, field, value); err != nil {
		fmt.Println("fabi pattern:", err)
		return
	}

	b, _ := json.MarshalIndent(user, "", "  ")
	if err := os.WriteFile(path, b, 0644); err != nil {
		fmt.Println("fabi pattern:", err)
		return
	}
	activePatterns = nil // reload on next use
	fmt.Println(trf("pattern.added", field, value))
}

// installPatternPack fetches a community pattern pack (JSON with the
// same patternSet shape) and merges it into the user file. The real
// marketplace: detection knowledge shared as data, never code.
// Fetch via curl — present on all supported OSes, no TLS deps added.
func installPatternPack(url, path string, user *patternSet) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		fmt.Println(tr("pattern.need.url"))
		return
	}
	out, err := exec.Command("curl", "-sfL", "--max-time", "15", url).Output()
	if err != nil {
		fmt.Println(trf("pattern.fetchfail", err))
		return
	}
	var pack patternSet
	if err := json.Unmarshal(out, &pack); err != nil {
		fmt.Println(tr("pattern.notpack"))
		return
	}
	user.ExePatterns = append(user.ExePatterns, pack.ExePatterns...)
	user.CmdPatterns = append(user.CmdPatterns, pack.CmdPatterns...)
	user.ScriptHosts = append(user.ScriptHosts, pack.ScriptHosts...)
	user.Ports = append(user.Ports, pack.Ports...)
	user.ScanDirs = append(user.ScanDirs, pack.ScanDirs...)
	user.ModelRoots = append(user.ModelRoots, pack.ModelRoots...)
	user.PkgAITools = append(user.PkgAITools, pack.PkgAITools...)
	user.PkgRuntimes = append(user.PkgRuntimes, pack.PkgRuntimes...)

	n := len(pack.ExePatterns) + len(pack.CmdPatterns) + len(pack.ScriptHosts) +
		len(pack.Ports) + len(pack.ScanDirs) + len(pack.ModelRoots) +
		len(pack.PkgAITools) + len(pack.PkgRuntimes)
	if n == 0 {
		fmt.Println(tr("pattern.pacitempty"))
		return
	}
	b, _ := json.MarshalIndent(user, "", "  ")
	if err := os.WriteFile(path, b, 0644); err != nil {
		fmt.Println("fabi pattern install:", err)
		return
	}
	activePatterns = nil
	fmt.Println(trf("pattern.packmerged", n))
}
