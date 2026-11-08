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
	// Env unset (common on Windows) — ask the OS for its UI language;
	// only honor it if we actually have strings for it.
	if l := osDefaultLang(); l != "" {
		if _, ok := messages[l]; ok {
			return l
		}
	}
	return "en"
}

// tr returns the localized string for key — English fallback, then
// the key itself as last resort so a missing string is VISIBLE
// rather than silently empty (L2: gaps must be shown, not hidden).
func tr(key string) string {
	if s, ok := messages[langCode][key]; ok {
		return s
	}
	if s, ok := messages["en"][key]; ok {
		return s
	}
	return "tr:" + key
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
	"zh": {
		"tagline":         "fabi — 您的 PC AI 资产的诚实之眼",
		"firstrun.scan":   "  运行 `fabi scan` 查看此 PC 上有什么。",
		"firstrun.help":   "  运行 `fabi help` 查看所有命令。",
		"hint.commands":   "命令: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — 诚实的版本检查",
		"update.running":  "  当前:",
		"update.latest":   "  最新:",
		"update.behind":   "  → 有新版本可用: %s",
		"update.uptodate": "  → 已是最新",
		"update.ahead":    "  → 比最新发布版本更新(dev 构建)",
		"update.noauto":   "  下载并替换 — fabi 不会自动安装",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "命令:",
		"usage.line":      "usage: fabi <command>",
	},
	"es": {
		"tagline":         "fabi — el ojo honesto sobre los activos de IA de tu PC",
		"firstrun.scan":   "  Ejecuta `fabi scan` para ver qué hay aquí.",
		"firstrun.help":   "  Ejecuta `fabi help` para todos los comandos.",
		"hint.commands":   "comandos: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — verificación honesta de versión",
		"update.running":  "  actual:",
		"update.latest":   "  última:",
		"update.behind":   "  → nueva versión disponible: %s",
		"update.uptodate": "  → actualizado",
		"update.ahead":    "  → por delante de la última versión (build dev)",
		"update.noauto":   "  descarga y reemplaza — fabi nunca instala solo",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "comandos:",
		"usage.line":      "usage: fabi <command>",
	},
	"de": {
		"tagline":         "fabi — das ehrliche Auge auf dem KI-Bestand deines PCs",
		"firstrun.scan":   "  `fabi scan` ausführen, um zu sehen, was hier ist.",
		"firstrun.help":   "  `fabi help` für alle Befehle.",
		"hint.commands":   "Befehle: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — ehrliche Versionsprüfung",
		"update.running":  "  laufend:",
		"update.latest":   "  neueste:",
		"update.behind":   "  → neuere Version verfügbar: %s",
		"update.uptodate": "  → auf dem neuesten Stand",
		"update.ahead":    "  → neuer als das letzte Release (Dev-Build)",
		"update.noauto":   "  herunterladen & ersetzen — fabi installiert nie automatisch",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "Befehle:",
		"usage.line":      "usage: fabi <command>",
	},
	"fr": {
		"tagline":         "fabi — l'œil honnête sur les actifs IA de votre PC",
		"firstrun.scan":   "  Lancez `fabi scan` pour voir ce qui est ici.",
		"firstrun.help":   "  Lancez `fabi help` pour toutes les commandes.",
		"hint.commands":   "commandes : scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — vérification honnête de version",
		"update.running":  "  actuelle :",
		"update.latest":   "  dernière :",
		"update.behind":   "  → nouvelle version disponible : %s",
		"update.uptodate": "  → à jour",
		"update.ahead":    "  → plus récent que la dernière version (build dev)",
		"update.noauto":   "  téléchargez et remplacez — fabi n'installe jamais seul",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "commandes :",
		"usage.line":      "usage: fabi <command>",
	},
	"ko": {
		"tagline":         "fabi — PC의 AI 자산을 보는 정직한 눈",
		"firstrun.scan":   "  `fabi scan`으로 여기 있는 것을 확인하세요.",
		"firstrun.help":   "  `fabi help`로 모든 명령을 보세요.",
		"hint.commands":   "명령: scan · doctor · run · serve · help",
		"update.title":    "FABI UPDATE — 정직한 버전 확인",
		"update.running":  "  실행 중:",
		"update.latest":   "  최신:",
		"update.behind":   "  → 새 버전이 있습니다: %s",
		"update.uptodate": "  → 최신 상태입니다",
		"update.ahead":    "  → 최신 릴리스보다 앞섬(dev 빌드)",
		"update.noauto":   "  다운로드 후 교체 — fabi는 자동 설치하지 않습니다",
		"update.err":      "fabi update — %v",
		"usage.cmds":      "명령:",
		"usage.line":      "usage: fabi <command>",
	},
}
