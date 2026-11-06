package main

import (
	"fmt"
	"strings"
	"time"
)

// fabi diff — what changed between scans. The ledger holds every
// scan; `fabi diff` compares last-2, `fabi diff N` compares the
// latest against N scans back (a whole watch session's drift).
func runDiff(args []string) {
	l := openLedger()
	defer l.close()
	windows := loadScanWindows(l.allEvents())
	if len(windows) < 2 {
		fmt.Println("fabi diff — need at least 2 scans (run `fabi scan` twice)")
		return
	}
	back := 1
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &back)
	}
	if back < 1 || back >= len(windows) {
		back = len(windows) - 1
	}
	prev, latest := windows[len(windows)-1-back], windows[len(windows)-1]

	key := func(f findingRec) string {
		if f.kind == "tool" {
			return "tool:" + normalizeToolName(f.name) +
				map[bool]string{true: "|r", false: "|i"}[f.running]
		}
		return f.kind + ":" + strings.ToLower(f.name)
	}
	having := map[string]findingRec{}
	for _, f := range prev.findings {
		having[key(f)] = f
	}
	now := map[string]findingRec{}
	for _, f := range latest.findings {
		now[key(f)] = f
	}

	var added, removed []findingRec
	for k, f := range now {
		if _, ok := having[k]; !ok {
			added = append(added, f)
		}
	}
	for k, f := range having {
		if _, ok := now[k]; !ok {
			removed = append(removed, f)
		}
	}

	fmt.Println()
	fmt.Printf("FABI DIFF — %s → %s (%d scans back)\n",
		time.UnixMilli(prev.startTs).Format("15:04"),
		time.UnixMilli(latest.startTs).Format("15:04"), back)
	fmt.Println("─────────────────────────────────────────────")
	if len(added) == 0 && len(removed) == 0 {
		fmt.Println("  no changes — estate is identical")
		return
	}
	for _, f := range added {
		fmt.Printf("  + %-9s %s — %s\n", f.kind, f.name, trunc(f.detail, 50))
	}
	for _, f := range removed {
		fmt.Printf("  - %-9s %s — %s\n", f.kind, f.name, trunc(f.detail, 50))
	}
	fmt.Printf("\n%d appeared, %d gone\n", len(added), len(removed))
}
