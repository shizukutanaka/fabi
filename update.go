package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// updateCheck queries GitHub releases once and reports the version
// delta — the only network call fabi makes, user-invoked only.
func updateCheck() (cur, latest, url string, behind bool, err error) {
	st, body, err := httpsGet("api.github.com",
		"/repos/shizukutanaka/fabi/releases/latest", 8*time.Second)
	if err != nil || st != 200 {
		return "", "", "", false, fmt.Errorf("github releases: %v", err)
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &rel); err != nil || rel.TagName == "" {
		return "", "", "", false, fmt.Errorf("unexpected release response")
	}
	cur = strings.TrimSuffix(strings.TrimPrefix(version, "v"), "-dev")
	latest = strings.TrimPrefix(rel.TagName, "v")
	return cur, latest, rel.HTMLURL, semverLess(cur, latest), nil
}

// fabi update — the honest update check. No auto-download, no
// auto-install (deny-by-default holds): reports the delta.
func runUpdate() {
	cur, latest, url, behind, err := updateCheck()
	if err != nil {
		fmt.Println("fabi update —", err)
		return
	}
	fmt.Println()
	fmt.Println("FABI UPDATE — honest version check")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("  running:  v%s\n", cur)
	fmt.Printf("  latest:   v%s\n", latest)
	switch {
	case behind:
		fmt.Printf("\n  → newer release available: %s\n", url)
		fmt.Println("  download & replace — fabi never auto-installs")
	case cur == latest:
		fmt.Println("  → up to date")
	default:
		fmt.Println("  → ahead of latest release (dev build)")
	}
	if l := openLedger(); l.db != nil {
		l.emit("update.checked", map[string]any{
			"current": cur, "latest": latest, "behind": behind,
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
