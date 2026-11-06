//go:build !windows && !linux

package main

import "context"

// installedPackages is unimplemented on this platform.
// Returns empty rather than failing the whole scan.
func installedPackages(ctx context.Context) ([]pkgInfo, error) {
	return nil, nil
}
