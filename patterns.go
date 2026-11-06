package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	dir, err := ledgerDir()
	if err == nil {
		if b, err := os.ReadFile(filepath.Join(dir, "patterns.json")); err == nil {
			var user patternSet
			if err := json.Unmarshal(b, &user); err == nil {
				ps.ExePatterns = append(ps.ExePatterns, user.ExePatterns...)
				ps.CmdPatterns = append(ps.CmdPatterns, user.CmdPatterns...)
				ps.Ports = append(ps.Ports, user.Ports...)
				ps.ScanDirs = append(ps.ScanDirs, user.ScanDirs...)
				ps.ModelRoots = append(ps.ModelRoots, user.ModelRoots...)
				ps.PkgAITools = append(ps.PkgAITools, user.PkgAITools...)
				ps.PkgRuntimes = append(ps.PkgRuntimes, user.PkgRuntimes...)
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
		fmt.Println("FABI PATTERNS — user additions in ~/.fabi/patterns.json")
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
			fmt.Println("  (none — add with: fabi pattern add <field> <value>)")
		}
		return
	}

	if args[0] != "add" || len(args) != 3 {
		fmt.Println("usage: fabi pattern add <exe|cmd|scripthost|dir|modelroot|pkgtool|pkgruntime|port> <value>")
		fmt.Println("example: fabi pattern add exe myserver.exe")
		fmt.Println("         fabi pattern add port myservice:9000")
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
	fmt.Printf("added %s = %s → ~/.fabi/patterns.json (next scan picks it up)\n", field, value)
}
