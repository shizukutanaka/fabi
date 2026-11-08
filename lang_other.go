//go:build !windows

package main

// osDefaultLang — on Unix the LANG/LC_* env vars are the standard;
// no extra syscall needed.
func osDefaultLang() string { return "" }
