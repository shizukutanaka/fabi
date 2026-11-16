package main

import (
	"time"
)

// fabi watch --tray — a resident face for people who never open a
// terminal: the status one-liner as a system-tray tooltip, plus a
// balloon when VRAM runs nearly full.
//
// This file is the platform-independent half and contains every
// decision: what the one line says, and whether a VRAM notification
// fires. The OS contact (Shell_NotifyIcon on Windows) lives in
// tray_windows.go; tray_other.go declines politely elsewhere.
// No state lives here — every snapshot is a view over the ledger,
// same as `fabi status` (the ledger is the only truth).

// VRAM alert tuning. Pressure = at/above vramPressureRatio; the alert
// re-arms only after usage falls below vramClearRatio (hysteresis, so
// a workload hovering at the line does not flap the balloon), and a
// fresh crossing inside vramAlertCooldown is noted but not shown.
const (
	vramPressureRatio = 0.90
	vramClearRatio    = 0.85
	vramAlertCooldown = 30 * time.Minute
)

// trayTipMax is the tooltip budget in runes — Shell_NotifyIcon's
// szTip holds 127 UTF-16 chars + NUL; stay under it with margin.
const trayTipMax = 120

// traySnapshot is one reading of machine health for the tray face.
// Counts come from statusJSON() (the same source serve and the TUI
// read); VRAM comes from latestGPUSample over the ledger.
type traySnapshot struct {
	HasScans     bool
	Running      int
	Installed    int
	Services     int
	Models       int
	LoadedModels int
	HasVRAM      bool
	VRAMUsedMiB  float64
	VRAMTotalMiB float64
}

// traySnapshotNow reads the current snapshot from the ledger.
func traySnapshotNow() traySnapshot {
	sj := statusJSON()
	if _, isErr := sj["error"]; isErr {
		return traySnapshot{}
	}
	s := traySnapshot{HasScans: true}
	s.Running = intOf(sj, "running")
	s.Installed = intOf(sj, "installed_idle")
	s.Services = intOf(sj, "services")
	s.Models = intOf(sj, "models")
	s.LoadedModels = intOf(sj, "loaded_models")

	l := openLedger()
	defer l.close()
	_, _, used, total := latestGPUSample(localEvents(l.allEvents(), l.importedRanges()))
	if total > 0 {
		s.HasVRAM = true
		s.VRAMUsedMiB = used
		s.VRAMTotalMiB = total
	}
	return s
}

// trayLine renders the snapshot as the one-line tooltip text. Pure:
// the same snapshot always renders the same line, on every OS.
func trayLine(s traySnapshot) string {
	if !s.HasScans {
		return tr("tray.tip.noscan")
	}
	line := trf("tray.tip", s.Running, s.Installed, s.Models)
	if s.HasVRAM && s.VRAMTotalMiB > 0 {
		line += trf("tray.tip.vram",
			s.VRAMUsedMiB/s.VRAMTotalMiB*100,
			s.VRAMUsedMiB/1024, s.VRAMTotalMiB/1024)
	}
	if r := []rune(line); len(r) > trayTipMax {
		line = string(r[:trayTipMax-1]) + "…"
	}
	return line
}

// vramAlert is the edge-trigger state for the pressure balloon.
type vramAlert struct {
	InPressure bool
	LastNotify time.Time
}

// vramAlertUpdate folds one VRAM sample into the alert state and
// reports whether a notification should fire now. Pure function of
// (state, sample, clock) so the whole policy is unit-testable:
// fire once on crossing into pressure, stay quiet while pressure
// lasts, re-arm below the clear ratio, and never fire twice inside
// the cooldown even across a re-arm.
func vramAlertUpdate(st vramAlert, usedMiB, totalMiB float64, now time.Time) (vramAlert, bool) {
	if totalMiB <= 0 {
		return st, false
	}
	ratio := usedMiB / totalMiB
	if ratio < vramClearRatio {
		st.InPressure = false
		return st, false
	}
	if ratio < vramPressureRatio || st.InPressure {
		return st, false
	}
	st.InPressure = true
	if !st.LastNotify.IsZero() && now.Sub(st.LastNotify) < vramAlertCooldown {
		return st, false
	}
	st.LastNotify = now
	return st, true
}

// trayNotification renders the balloon for a pressure sample. It
// carries only aggregate numbers (percent, GiB, loaded-model count) —
// no model names, paths, or host identifiers leave the machine; the
// balloon is rendered by the user's own shell on their own screen.
func trayNotification(s traySnapshot) (title, body string) {
	return tr("tray.notify.title"), trf("tray.notify.body",
		s.VRAMUsedMiB/s.VRAMTotalMiB*100,
		s.VRAMUsedMiB/1024, s.VRAMTotalMiB/1024, s.LoadedModels)
}

// trayNote is one balloon request handed to the OS face.
type trayNote struct {
	Title string
	Body  string
}

// trayHandle is the running OS face. update swaps the tooltip (and
// shows a balloon when note != nil); quit closes when the user picks
// Quit in the tray menu; close removes the icon and stops the loop.
type trayHandle struct {
	updateFn func(line string, note *trayNote)
	quitCh   chan struct{}
	closeFn  func()
}

func (h *trayHandle) update(line string, note *trayNote) {
	if h != nil && h.updateFn != nil {
		h.updateFn(line, note)
	}
}

func (h *trayHandle) close() {
	if h != nil && h.closeFn != nil {
		h.closeFn()
	}
}

// trayScanNow is how the tray menu's "Scan now" reaches the watch
// loop — buffered so a click never blocks the message loop.
var trayScanNow = make(chan struct{}, 1)

// refreshTray pushes one fresh snapshot to the tray face and folds
// the VRAM sample into the alert state. Called by the watch loop
// after each scan; a nil handle (non-Windows, or tray failed to
// start) makes it a no-op.
func refreshTray(h *trayHandle, st *vramAlert) {
	if h == nil {
		return
	}
	s := traySnapshotNow()
	var note *trayNote
	if s.HasVRAM {
		var fire bool
		*st, fire = vramAlertUpdate(*st, s.VRAMUsedMiB, s.VRAMTotalMiB, time.Now())
		if fire {
			title, body := trayNotification(s)
			note = &trayNote{Title: title, Body: body}
		}
	}
	h.update(trayLine(s), note)
}

// intOf reads a numeric field from a statusJSON-style map.
func intOf(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}
