//go:build !windows

package main

import "fmt"

// startTray outside Windows: there is no shell tray to live in, so
// say so once and let watch run headless — the resident history
// function is unchanged, only the face is missing.
func startTray() *trayHandle {
	fmt.Println(tr("tray.unsupported"))
	return nil
}
