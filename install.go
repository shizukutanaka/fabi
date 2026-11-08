package main

import (
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

	fmt.Println()
	fmt.Println(tr("install.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Println("this will place a launcher so `fabi watch` runs at logon:")
	fmt.Println()
	fmt.Printf("  write: %s\n", cmd)
	fmt.Printf("  contents: \"%s\" watch\n", exe)
	fmt.Println()
	fmt.Println(tr("install.apply"))
	fmt.Printf("  fabi install --yes\n")
	fmt.Println("undo later:")
	fmt.Printf("  fabi uninstall\n")
}

func installStartup(exe, cmd string) {
	l := openLedger()
	defer l.close()
	content := fmt.Sprintf("@echo off\r\nstart \"\" /min \"%s\" watch\r\n", exe)
	if err := os.WriteFile(cmd, []byte(content), 0644); err != nil {
		fmt.Printf("FABI INSTALL — failed: %v\n", err)
		l.emit("install.action", map[string]any{"ok": false, "error": err.Error()})
		return
	}
	fmt.Printf("FABI INSTALL — wrote %s\n", cmd)
	fmt.Println("`fabi watch` will accumulate history at next logon.")
	fmt.Println("verify:  dir \"%APPDATA%\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\"")
	l.emit("install.action", map[string]any{"ok": true, "path": cmd})
}

func runUninstall(args []string) {
	l := openLedger()
	defer l.close()
	cmd := startupCmdPath()
	if !hasYesFlag(args) {
		fmt.Println()
		fmt.Println("FABI UNINSTALL — dry run")
		fmt.Println("─────────────────────────────────────────────")
		fmt.Printf("  delete: %s\n", cmd)
		fmt.Println("  (ledger ~/.fabi/ is NOT deleted — your data stays)")
		fmt.Println()
		fmt.Println("run with --yes to apply: fabi uninstall --yes")
		return
	}
	if err := os.Remove(cmd); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("nothing installed — no launcher found")
			return
		}
		fmt.Printf("fabi: uninstall failed: %v\n", err)
		l.emit("uninstall.action", map[string]any{"ok": false})
		return
	}
	fmt.Printf("removed %s. ledger kept at ~/.fabi/\n", cmd)
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
