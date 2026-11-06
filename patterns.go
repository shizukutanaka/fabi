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
