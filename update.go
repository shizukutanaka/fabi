package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// fabi update — the honest update check. No auto-download, no
// auto-install (deny-by-default holds): it queries the GitHub release
// API once, compares the tag to the running binary's version, and
// reports the delta. The only network call fabi makes — and only
// when the user explicitly asks.
func runUpdate() {
	st, body, err := httpsGet("api.github.com",
		"/repos/shizukutanaka/fabi/releases/latest", 8*time.Second)
	if err != nil || st != 200 {
		fmt.Println("fabi update — could not reach github releases:",
			err)
		return
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &rel); err != nil || rel.TagName == "" {
		fmt.Println("fabi update — unexpected release response")
		return
	}

	cur := strings.TrimPrefix(version, "v")
	cur = strings.TrimSuffix(cur, "-dev")
	latest := strings.TrimPrefix(rel.TagName, "v")

	fmt.Println()
	fmt.Println("FABI UPDATE — honest version check")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("  running:  v%s\n", cur)
	fmt.Printf("  latest:   %s\n", rel.TagName)

	if semverLess(cur, latest) {
		fmt.Println()
		fmt.Printf("  → newer release available: %s\n", rel.HTMLURL)
		fmt.Println("  download & replace — fabi never auto-installs")
	} else if cur == latest {
		fmt.Println("  → up to date")
	} else {
		fmt.Println("  → ahead of latest release (dev build)")
	}

	if l := openLedger(); l.db != nil {
		l.emit("update.checked", map[string]any{
			"current": cur, "latest": latest,
			"behind": semverLess(cur, latest),
		})
		l.close()
	}
}

// semverLess compares "0.0.5" style versions numerically.
func semverLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na < nb
		}
	}
	return len(pa) < len(pb)
}
