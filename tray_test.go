package main

import (
	"strings"
	"testing"
	"time"
)

// trayLine — the tooltip is one honest line: counts from the latest
// scan, VRAM segment only when a GPU sample exists, never longer
// than the shell's tooltip budget.
func TestTrayLine(t *testing.T) {
	// No scans yet → the first-run hint, not a row of zeros.
	if got := trayLine(traySnapshot{}); !strings.Contains(got, "no scans yet") {
		t.Errorf("empty snapshot line = %q, want the no-scans hint", got)
	}

	s := traySnapshot{
		HasScans: true, Running: 3, Installed: 5, Services: 2, Models: 7,
		HasVRAM: true, VRAMUsedMiB: 12288, VRAMTotalMiB: 16384,
	}
	got := trayLine(s)
	for _, want := range []string{"3 running", "5 idle", "7 models", "VRAM 75%", "12.0/16.0 GB"} {
		if !strings.Contains(got, want) {
			t.Errorf("line %q missing %q", got, want)
		}
	}
	if n := len([]rune(got)); n > trayTipMax {
		t.Errorf("line is %d runes, over the %d tooltip budget", n, trayTipMax)
	}

	// No GPU sample → no VRAM segment (never invent numbers).
	s.HasVRAM = false
	if got := trayLine(s); strings.Contains(got, "VRAM") {
		t.Errorf("line %q shows VRAM without a sample", got)
	}
}

// vramAlertUpdate — the notification policy: fire once on crossing
// into pressure, silence while it lasts, hysteresis re-arm, and a
// cooldown that survives a re-arm.
func TestVRAMAlertUpdate(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0)
	const total = 16384.0
	below := total * 0.50
	press := total * 0.95
	mid := total * 0.87 // between clear (0.85) and pressure (0.90)

	st := vramAlert{}
	// No GPU / no sample → never fires, state untouched.
	if st2, fire := vramAlertUpdate(st, 0, 0, t0); fire || st2 != st {
		t.Errorf("total=0: fire=%v state=%v, want no fire, unchanged", fire, st2)
	}
	// Below pressure → no fire.
	if _, fire := vramAlertUpdate(st, below, total, t0); fire {
		t.Fatal("fired below pressure")
	}
	// Crossing into pressure → fires, records the time.
	var fire bool
	if st, fire = vramAlertUpdate(st, press, total, t0); !fire {
		t.Fatal("did not fire on crossing into pressure")
	}
	if !st.InPressure || !st.LastNotify.Equal(t0) {
		t.Errorf("state after fire = %+v", st)
	}
	// Staying in pressure → quiet.
	if st, fire = vramAlertUpdate(st, press, total, t0.Add(time.Minute)); fire {
		t.Fatal("fired again while pressure lasted")
	}
	// The hysteresis band does NOT re-arm.
	if st, fire = vramAlertUpdate(st, mid, total, t0.Add(2*time.Minute)); fire || !st.InPressure {
		t.Errorf("band sample: fire=%v inPressure=%v, want quiet + still armed-off", fire, st.InPressure)
	}
	// Dropping below the clear ratio re-arms…
	if st, fire = vramAlertUpdate(st, below, total, t0.Add(3*time.Minute)); fire || st.InPressure {
		t.Errorf("clear sample: fire=%v inPressure=%v, want quiet + re-armed", fire, st.InPressure)
	}
	// …but a re-crossing inside the cooldown stays quiet.
	if st, fire = vramAlertUpdate(st, press, total, t0.Add(10*time.Minute)); fire {
		t.Fatal("fired inside cooldown after re-arm")
	}
	if !st.InPressure {
		t.Error("re-crossing must still mark pressure (edge consumed)")
	}
	// After clearing again and waiting out the cooldown, it fires.
	st, _ = vramAlertUpdate(st, below, total, t0.Add(20*time.Minute))
	if _, fire = vramAlertUpdate(st, press, total, t0.Add(31*time.Minute)); !fire {
		t.Fatal("did not fire after cooldown elapsed")
	}
}

// startupContent — the Startup-folder launcher: plain watch by
// default, `watch --tray` when the tray face was requested.
func TestStartupContent(t *testing.T) {
	plain := startupContent(`C:\fabi\fabi.exe`, false)
	if !strings.Contains(plain, `"C:\fabi\fabi.exe" watch`) || strings.Contains(plain, "--tray") {
		t.Errorf("plain launcher = %q", plain)
	}
	tray := startupContent(`C:\fabi\fabi.exe`, true)
	if !strings.Contains(tray, `"C:\fabi\fabi.exe" watch --tray`) {
		t.Errorf("tray launcher = %q", tray)
	}
	if !strings.HasPrefix(tray, "@echo off\r\n") {
		t.Errorf("launcher lost its batch header: %q", tray)
	}
}

// trayNotification — the balloon carries aggregates only: percent,
// GiB figures, loaded count. No model names or paths.
func TestTrayNotificationAggregatesOnly(t *testing.T) {
	s := traySnapshot{
		HasScans: true, LoadedModels: 2,
		HasVRAM: true, VRAMUsedMiB: 15360, VRAMTotalMiB: 16384,
	}
	title, body := trayNotification(s)
	if title == "" || !strings.Contains(body, "94%") || !strings.Contains(body, "15.0 of 16.0 GB") {
		t.Errorf("notification = %q / %q", title, body)
	}
}
