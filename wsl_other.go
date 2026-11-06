//go:build !windows

package main

import "context"

func wslDistros(ctx context.Context) []wslDistro {
	return nil
}
