package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// fabi install — drop a launcher in the Startup folder so `fabi watch`
// accumulates history automatically at logon.
// No admin needed, fully reversible. Dry-run by default; --yes to apply.

func startupCmdPath() string {
	return filepath.Join(os.Getenv("APPDATA"),
		`Microsoft\Windows\Start Menu\Programs\Startup`, "fabi-watch.cmd")
}

func runInstall(args []string) {
	yes := hasYesFlag(args)
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("fabi: cannot resolve own path:", err)
		return
	}
	cmd := startupCmdPath()

	if yes {
		installStartup(exe, cmd)
		return
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"dry_run": true, "will_write": cmd,
			"contents": fmt.Sprintf("\"%s\" watch", exe),
			"apply":    "fabi install --yes", "undo": "fabi uninstall",
		}, "", "  ")
		fmt.Println(string(out))
		return
	}
	fmt.Println()
	fmt.Println(tr("install.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Println(tr("install.will"))
	fmt.Println()
	fmt.Printf("  write: %s\n", cmd)
	fmt.Printf("  contents: \"%s\" watch\n", exe)
	fmt.Println()
	fmt.Println(tr("install.apply"))
	fmt.Printf("  fabi install --yes\n")
	fmt.Println(tr("install.undo"))
	fmt.Printf("  fabi uninstall\n")
}

func installStartup(exe, cmd string) {
	l := openLedger()
	defer l.close()
	content := fmt.Sprintf("@echo off\r\nstart \"\" /min \"%s\" watch\r\n", exe)
	if err := os.WriteFile(cmd, []byte(content), 0644); err != nil {
		if jsonOut {
			fmt.Printf(`{"ok":false,"error":"%v"}`+"\n", err)
		} else {
			fmt.Println(trf("install.failed", err))
		}
		l.emit("install.action", map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"ok": true, "path": cmd, "runs": "fabi watch at logon",
		}, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(trf("install.wrote", cmd))
		fmt.Println(tr("install.willrun"))
		fmt.Println("verify:  dir \"%APPDATA%\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\"")
	}
	l.emit("install.action", map[string]any{"ok": true, "path": cmd})
}

func runUninstall(args []string) {
	l := openLedger()
	defer l.close()
	cmd := startupCmdPath()
	if !hasYesFlag(args) {
		if jsonOut {
			out, _ := json.MarshalIndent(map[string]any{
				"dry_run": true, "will_delete": cmd,
				"ledger_kept": true, "apply": "fabi uninstall --yes",
			}, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Println()
		fmt.Println(tr("uninstall.title"))
		fmt.Println("─────────────────────────────────────────────")
		fmt.Printf("  delete: %s\n", cmd)
		fmt.Println(tr("uninstall.keep"))
		fmt.Println()
		fmt.Println(tr("uninstall.apply"))
		return
	}
	if err := os.Remove(cmd); err != nil {
		if os.IsNotExist(err) {
			if jsonOut {
				fmt.Println(`{"ok":true,"removed":false,"reason":"nothing installed"}`)
			} else {
				fmt.Println(tr("uninstall.none"))
			}
			return
		}
		if jsonOut {
			fmt.Printf(`{"ok":false,"error":"%v"}`+"\n", err)
		} else {
			fmt.Println(trf("uninstall.failed", err))
		}
		l.emit("uninstall.action", map[string]any{"ok": false})
		return
	}
	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"ok": true, "removed": cmd, "ledger_kept": true,
		}, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(trf("uninstall.removed", cmd))
	}
	l.emit("uninstall.action", map[string]any{"ok": true})
}

func hasYesFlag(args []string) bool {
	for _, a := range args {
		if a == "--yes" || a == "-y" {
			return true
		}
	}
	return false
}
