package main

import (
	"fmt"
	"os"
)

// lang.go — minimal i18n. Strings are data (same law as patterns):
// adding a language means adding a map entry, never code changes.
// Detection: FABI_LANG > LANG > LC_ALL ("ja_JP.UTF-8" → "ja").
// English is always the fallback — a missing key is honest output,
// not an error.
var langCode = detectLang()

func detectLang() string {
	for _, e := range []string{"FABI_LANG", "LANG", "LC_ALL"} {
		v := os.Getenv(e)
		if len(v) >= 2 {
			if _, ok := messages[v[:2]]; ok {
				return v[:2]
			}
		}
	}
	return "en"
}

// tr returns the localized string for key, English fallback.
func tr(key string) string {
	if s, ok := messages[langCode][key]; ok {
		return s
	}
	return messages["en"][key]
}

func trf(key string, args ...any) string {
	return fmt.Sprintf(tr(key), args...)
}

var messages = map[string]map[string]string{
	"en": {
		"tagline":         "fabi — the honest eye on your PC's AI estate",
		"firstrun.scan":   "  Run `fabi scan` to see what's here.",
		"firstrun.help":   "  Run `fabi help` for all commands.",
		"hint.commands":   "commands: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — honest version check",
		"update.running":  "  running:",
		"update.latest":   "  latest:",
		"update.behind":   "  → newer release available: %s",
		"update.uptodate": "  → up to date",
		"update.ahead":    "  → ahead of latest release (dev build)",
		"update.noauto":   "  download & replace — fabi never auto-installs",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "commands:",
		"usage.line":      "usage: fabi <command>",
	},
	"ja": {
		"tagline":         "fabi — あなたのPCのAI資産を見る正直な目",
		"firstrun.scan":   "  `fabi scan` でこのPCにあるものを確認",
		"firstrun.help":   "  `fabi help` で全コマンドを表示",
		"hint.commands":   "コマンド: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — 正直なバージョン確認",
		"update.running":  "  現在:",
		"update.latest":   "  最新:",
		"update.behind":   "  → 新しいリリースがあります: %s",
		"update.uptodate": "  → 最新です",
		"update.ahead":    "  → 最新リリースより新しい(devビルド)",
		"update.noauto":   "  ダウンロードして置き換え — fabiは自動インストールしません",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "コマンド:",
		"usage.line":      "usage: fabi <command>",
	},
}
