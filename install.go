package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// fabi install — register `fabi watch` as a logon task so history
// accumulates without the user thinking about it.
// Permission-first: requires --yes. Dry-run by default (SPEC #17, #24).

const taskName = "FabiWatch"

func runInstall(args []string) {
	yes := hasYesFlag(args)
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("fabi: cannot resolve own path:", err)
		return
	}
	exe = strings.ReplaceAll(exe, "/", `\`)

	if yes {
		installTask(exe)
		return
	}

	fmt.Println()
	fmt.Println("FABI INSTALL — dry run")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Println("this will register a logon task so `fabi watch`")
	fmt.Println("accumulates history automatically:")
	fmt.Println()
	fmt.Printf("  schtasks /Create /TN %s /SC ONLOGON /F /TR\n", taskName)
	fmt.Printf("    \"%s watch\"\n", exe)
	fmt.Println()
	fmt.Println("run with --yes to apply:")
	fmt.Printf("  fabi install --yes\n")
	fmt.Println("undo later:")
	fmt.Printf("  fabi uninstall\n")
}

func installTask(exe string) {
	l := openLedger()
	defer l.close()
	cmd := exec.Command("schtasks", "/Create",
		"/TN", taskName,
		"/SC", "ONLOGON",
		"/TR", fmt.Sprintf(`"%s watch"`, exe),
		"/F")
	out, err := cmd.CombinedOutput()
	fmt.Println()
	if err != nil {
		fmt.Printf("FABI INSTALL — failed: %v\n%s\n", err, strings.TrimSpace(string(out)))
		l.emit("install.action", map[string]any{"ok": false, "error": err.Error()})
		return
	}
	fmt.Printf("FABI INSTALL — task %q registered on logon.\n", taskName)
	fmt.Println("history accumulates from next logon; `fabi report` shows it.")
	l.emit("install.action", map[string]any{"ok": true, "task": taskName})
}

func runUninstall(args []string) {
	l := openLedger()
	defer l.close()
	if !hasYesFlag(args) {
		fmt.Println()
		fmt.Println("FABI UNINSTALL — dry run")
		fmt.Println("─────────────────────────────────────────────")
		fmt.Printf("  schtasks /Delete /TN %s /F\n", taskName)
		fmt.Println("  (ledger ~/.fabi/ is NOT deleted — your data stays)")
		fmt.Println()
		fmt.Println("run with --yes to apply: fabi uninstall --yes")
		return
	}
	cmd := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("fabi: uninstall failed: %v — %s\n", err, strings.TrimSpace(string(out)))
		l.emit("uninstall.action", map[string]any{"ok": false})
		return
	}
	fmt.Printf("task %q removed. ledger kept at ~/.fabi/\n", taskName)
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
