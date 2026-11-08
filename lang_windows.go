//go:build windows

package main

import "golang.org/x/sys/windows"

// osDefaultLang returns the Windows UI display language as an ISO
// code — needed because Windows rarely sets LANG, so env detection
// alone would pin most Windows users to English.
func osDefaultLang() string {
	lid, _, _ := windows.NewLazySystemDLL("kernel32.dll").
		NewProc("GetUserDefaultUILanguage").Call()
	// LANGID: low 10 bits = primary language id
	switch lid & 0x3ff {
	case 0x11:
		return "ja"
	case 0x04:
		return "zh"
	case 0x12:
		return "ko"
	case 0x0a:
		return "es"
	case 0x07:
		return "de"
	case 0x0c:
		return "fr"
	case 0x16:
		return "pt"
	case 0x19:
		return "ru"
	case 0x10:
		return "it"
	}
	return "en"
}
