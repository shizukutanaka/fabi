package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// updateCheck queries GitHub releases once and reports the version
// delta — the only network call fabi makes, user-invoked only.
// It shells out to the OS's curl (present on Win10+, macOS, Linux)
// rather than linking crypto/tls — keeps ~2 MB out of the binary.
func updateCheck() (cur, latest, url string, behind bool, err error) {
	out, err := exec.Command("curl", "-sf", "--max-time", "8",
		"https://api.github.com/repos/shizukutanaka/fabi/releases/latest").Output()
	if err != nil {
		return "", "", "", false,
			fmt.Errorf("github releases unreachable (curl: %v)", err)
	}
	body := out
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
		if jsonOut {
			out, _ := json.MarshalIndent(map[string]any{
				"ok": false, "error": err.Error(),
				"current":      strings.TrimSuffix(strings.TrimPrefix(version, "v"), "-dev"),
				"auto_install": false,
			}, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(trf("update.err", err))
		}
		return
	}
	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"current": cur, "latest": latest, "behind": behind,
			"url": url, "auto_install": false,
		}, "", "  ")
		fmt.Println(string(out))
		return
	}
	fmt.Println()
	fmt.Println(tr("update.title"))
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("%s  v%s\n", tr("update.running"), cur)
	fmt.Printf("%s  v%s\n", tr("update.latest"), latest)
	switch {
	case behind:
		fmt.Println()
		fmt.Println(trf("update.behind", url))
		fmt.Println(tr("update.noauto"))
	case cur == latest:
		fmt.Println(tr("update.uptodate"))
	default:
		fmt.Println(tr("update.ahead"))
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
