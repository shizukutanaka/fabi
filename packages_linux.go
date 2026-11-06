//go:build linux

package main

import (
	"context"
	"os/exec"
	"strings"
)

// installedPackages on Linux queries dpkg. Slow paths (brew, snap, flatpak)
// are deferred — a partial inventory beats none.
func installedPackages(ctx context.Context) ([]pkgInfo, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // dpkg absent (non-Debian) — not an error
	}
	var pkgs []pkgInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		pkgs = append(pkgs, pkgInfo{
			name:    parts[0],
			version: parts[1],
			source:  "dpkg",
		})
	}
	return pkgs, nil
}
