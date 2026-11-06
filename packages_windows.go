//go:build windows

package main

import (
	"context"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var uninstallKeys = []struct {
	hive registry.Key
	path string
}{
	{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
}

func installedPackages(ctx context.Context) ([]pkgInfo, error) {
	var out []pkgInfo
	seen := map[string]bool{}
	for _, uk := range uninstallKeys {
		if ctx.Err() != nil {
			break
		}
		k, err := registry.OpenKey(uk.hive, uk.path, registry.READ)
		if err != nil {
			continue
		}
		names, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}
		for _, name := range names {
			if ctx.Err() != nil {
				return out, nil
			}
			sk, err := registry.OpenKey(uk.hive, uk.path+`\`+name, registry.READ)
			if err != nil {
				continue
			}
			dn, _, _ := sk.GetStringValue("DisplayName")
			ver, _, _ := sk.GetStringValue("DisplayVersion")
			sk.Close()
			dn = strings.TrimSpace(dn)
			if dn == "" || seen[strings.ToLower(dn)] {
				continue
			}
			seen[strings.ToLower(dn)] = true
			out = append(out, pkgInfo{name: dn, version: ver, source: "registry"})
		}
	}
	return out, nil
}
