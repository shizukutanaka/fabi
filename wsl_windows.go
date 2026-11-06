//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// wslDistros lists registered WSL2 distros and whether their disk image
// actually exists. A registered distro with a missing vhdx is a dead
// install — dead weight that Doctor can flag.
func wslDistros(ctx context.Context) []wslDistro {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Lxss`, registry.READ)
	if err != nil {
		return nil // WSL not installed — not an error
	}
	defer k.Close()
	guids, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil
	}
	var out []wslDistro
	for _, g := range guids {
		if ctx.Err() != nil {
			break
		}
		sk, err := registry.OpenKey(k, g, registry.READ)
		if err != nil {
			continue
		}
		name, _, _ := sk.GetStringValue("DistributionName")
		base, _, _ := sk.GetStringValue("BasePath")
		sk.Close()
		if name == "" {
			continue
		}
		base = strings.TrimPrefix(base, `\\?\`)
		vhdx := base
		if !strings.HasSuffix(strings.ToLower(base), ".vhdx") {
			vhdx = filepath.Join(base, "ext4.vhdx")
		}
		d := wslDistro{name: name, vhdx: vhdx}
		if _, err := os.Stat(vhdx); err == nil {
			d.present = true
		}
		out = append(out, d)
	}
	return out
}
