//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows face for `fabi watch --tray`: a Shell_NotifyIcon tray icon
// whose tooltip is the status one-liner, a balloon for VRAM pressure,
// and a right-click menu (Scan now / Quit). Implemented directly on
// user32/shell32 via the already-required golang.org/x/sys — no new
// dependency, no bundled icon asset (the stock application icon is
// used; the tooltip text carries the identity).
//
// All decisions live in tray.go. This file only renders: it receives
// finished lines and balloons, and forwards two user intents (scan,
// quit) back to the watch loop.

var (
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procShellNotifyIcon  = shell32.NewProc("Shell_NotifyIconW")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procUnregisterClass  = user32.NewProc("UnregisterClassW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procPostMessage      = user32.NewProc("PostMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procLoadIcon         = user32.NewProc("LoadIconW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procSetForegroundWnd = user32.NewProc("SetForegroundWindow")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
)

const (
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifInfo    = 0x00000010
	niifInfo   = 0x00000001

	wmCommand       = 0x0111
	wmDestroy       = 0x0002
	wmRButtonUp     = 0x0205
	wmLButtonDblClk = 0x0203
	wmApp           = 0x8000

	wmTrayCallback = wmApp + 100 // shell → us: mouse events on the icon
	wmTrayUpdate   = wmApp + 101 // watch loop → us: new line / balloon
	wmTrayClose    = wmApp + 102 // watch loop → us: remove icon, stop

	idiApplication = 32512

	mfString    = 0x00000000
	mfGrayed    = 0x00000001
	mfDisabled  = 0x00000002
	mfSeparator = 0x00000800
	tpmRightBtn = 0x0002

	idTrayScan = 1
	idTrayQuit = 2
)

type trayPoint struct{ X, Y int32 }

type trayMsg struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      trayPoint
}

type trayWndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type notifyIconData struct {
	Size             uint32
	Hwnd             windows.HWND
	UID              uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             windows.Handle
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUIDItem         windows.GUID
	BalloonIcon      windows.Handle
}

// trayWin is the single tray instance (one icon per process).
type trayWin struct {
	mu       sync.Mutex
	hwnd     windows.HWND
	line     string
	note     *trayNote
	quitCh   chan struct{}
	quitOnce sync.Once
}

var trayWinInst *trayWin

func (t *trayWin) getHwnd() windows.HWND {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hwnd
}

func (t *trayWin) requestQuit() {
	t.quitOnce.Do(func() { close(t.quitCh) })
}

// requestTrayScan forwards a menu/double-click intent to the watch
// loop without ever blocking the message loop.
func requestTrayScan() {
	select {
	case trayScanNow <- struct{}{}:
	default:
	}
}

func startTray() *trayHandle {
	inst := &trayWin{quitCh: make(chan struct{})}
	trayWinInst = inst
	ready := make(chan error, 1)
	h := &trayHandle{quitCh: inst.quitCh}
	h.updateFn = func(line string, note *trayNote) {
		inst.mu.Lock()
		inst.line = line
		if note != nil {
			inst.note = note
		}
		hwnd := inst.hwnd
		inst.mu.Unlock()
		if hwnd != 0 {
			procPostMessage.Call(uintptr(hwnd), wmTrayUpdate, 0, 0)
		}
	}
	h.closeFn = func() {
		if hwnd := inst.getHwnd(); hwnd != 0 {
			procPostMessage.Call(uintptr(hwnd), wmTrayClose, 0, 0)
		}
	}
	go inst.run(ready)
	if err := <-ready; err != nil {
		trayWinInst = nil
		fmt.Println(trf("tray.failed", err))
		return nil
	}
	fmt.Println(tr("tray.active"))
	return h
}

// run owns the message loop on one locked OS thread: create the
// message-only window, add the icon, pump until close.
func (t *trayWin) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hinst, _, _ := procGetModuleHandle.Call(0)
	className, _ := windows.UTF16PtrFromString("FabiTrayWindow")
	wc := trayWndClassEx{
		WndProc:   windows.NewCallback(trayWndProc),
		Instance:  windows.Handle(hinst),
		ClassName: className,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if atom, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		ready <- fmt.Errorf("RegisterClassEx: %v", err)
		return
	}
	defer procUnregisterClass.Call(uintptr(unsafe.Pointer(className)), hinst)

	winName, _ := windows.UTF16PtrFromString("fabi")
	hwndRaw, _, err := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(winName)),
		0, 0, 0, 0, 0,
		uintptr(windows.HWND(^uintptr(2))), // HWND_MESSAGE — never visible
		0, hinst, 0)
	if hwndRaw == 0 {
		ready <- fmt.Errorf("CreateWindowEx: %v", err)
		return
	}
	hwnd := windows.HWND(hwndRaw)
	icon, _, _ := procLoadIcon.Call(0, idiApplication)

	t.mu.Lock()
	t.hwnd = hwnd
	t.mu.Unlock()

	nid := notifyIconData{
		Hwnd:            hwnd,
		UID:             1,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: wmTrayCallback,
		Icon:            windows.Handle(icon),
	}
	nid.Size = uint32(unsafe.Sizeof(nid))
	copyUTF16(nid.Tip[:], "fabi")
	if ok, _, serr := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid))); ok == 0 {
		ready <- fmt.Errorf("Shell_NotifyIcon(add): %v", serr)
		return
	}
	ready <- nil

	var m trayMsg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if ret == 0 || ret == ^uintptr(0) { // WM_QUIT or error
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}

	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	procDestroyWindow.Call(hwndRaw)
}

// applyUpdate renders the latest pending line (and balloon, if any).
// Runs on the message-loop thread via wmTrayUpdate.
func (t *trayWin) applyUpdate() {
	t.mu.Lock()
	line, note := t.line, t.note
	t.note = nil
	hwnd := t.hwnd
	t.mu.Unlock()
	if hwnd == 0 {
		return
	}
	nid := notifyIconData{Hwnd: hwnd, UID: 1, Flags: nifTip}
	nid.Size = uint32(unsafe.Sizeof(nid))
	copyUTF16(nid.Tip[:], line)
	if note != nil {
		nid.Flags |= nifInfo
		copyUTF16(nid.Info[:], note.Body)
		copyUTF16(nid.InfoTitle[:], note.Title)
		nid.InfoFlags = niifInfo
		nid.TimeoutOrVersion = 10000
	}
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func (t *trayWin) showMenu() {
	hwnd := t.getHwnd()
	if hwnd == 0 {
		return
	}
	hmenu, _, _ := procCreatePopupMenu.Call()
	if hmenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hmenu)

	t.mu.Lock()
	line := t.line
	t.mu.Unlock()
	if p, err := windows.UTF16PtrFromString(line); err == nil {
		procAppendMenu.Call(hmenu, mfString|mfGrayed|mfDisabled, 0,
			uintptr(unsafe.Pointer(p)))
	}
	procAppendMenu.Call(hmenu, mfSeparator, 0, 0)
	if p, err := windows.UTF16PtrFromString(tr("tray.menu.scan")); err == nil {
		procAppendMenu.Call(hmenu, mfString, idTrayScan, uintptr(unsafe.Pointer(p)))
	}
	if p, err := windows.UTF16PtrFromString(tr("tray.menu.quit")); err == nil {
		procAppendMenu.Call(hmenu, mfString, idTrayQuit, uintptr(unsafe.Pointer(p)))
	}
	procSetForegroundWnd.Call(uintptr(hwnd))
	var pt trayPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procTrackPopupMenu.Call(hmenu, tpmRightBtn,
		uintptr(int64(pt.X)), uintptr(int64(pt.Y)), 0, uintptr(hwnd), 0)
}

func trayWndProc(hwnd windows.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	inst := trayWinInst
	switch msg {
	case wmTrayCallback:
		if inst == nil {
			return 0
		}
		switch lparam {
		case wmRButtonUp:
			inst.showMenu()
		case wmLButtonDblClk:
			requestTrayScan()
		}
		return 0
	case wmTrayUpdate:
		if inst != nil {
			inst.applyUpdate()
		}
		return 0
	case wmTrayClose:
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case wmCommand:
		if inst == nil {
			return 0
		}
		switch wparam & 0xFFFF {
		case idTrayScan:
			requestTrayScan()
		case idTrayQuit:
			inst.requestQuit()
			procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return r
}

// copyUTF16 copies s into a fixed UTF-16 field, truncating honestly
// at the field's capacity (the last cell stays NUL).
func copyUTF16(dst []uint16, s string) {
	u := windows.StringToUTF16(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}
