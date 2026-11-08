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
		"tagline":               "fabi — the honest eye on your PC's AI estate",
		"firstrun.scan":         "  Run `fabi scan` to see what's here.",
		"firstrun.help":         "  Run `fabi help` for all commands.",
		"hint.commands":         "commands: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — honest version check",
		"update.running":        "  running:",
		"update.latest":         "  latest:",
		"update.behind":         "  → newer release available: %s",
		"update.uptodate":       "  → up to date",
		"update.ahead":          "  → ahead of latest release (dev build)",
		"update.noauto":         "  download & replace — fabi never auto-installs",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "commands:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: ledger empty — run `fabi scan`",
		"status.empty.scans":    "fabi: no scans recorded — run `fabi scan`",
		"doctor.empty.ledger":   "FABI DOCTOR — ledger is empty. Run `fabi scan` first.",
		"doctor.empty.findings": "FABI DOCTOR — scans exist but no findings recorded yet.",
		"doctor.title":          "FABI DOCTOR — evidence-based recommendations",
		"doctor.none":           "No recommendations. Every finding has provenance.",
		"doctor.note":           "  note: fabi never deletes user files autonomously.\n        remove candidates manually after review.",
		"scan.title":            "FABI SCAN — AI estate report",
		"report.title":          "FABI REPORT — your AI estate, measured",
		"report.empty.ledger":   "FABI REPORT — ledger is empty. Run `fabi scan` first.",
		"report.empty.scans":    "FABI REPORT — no scans in ledger.",
		"report.estate":         "ESTATE (latest scan)",
		"report.inference":      "INFERENCE — measured tok/s (real, not predicted)",
		"report.work":           "FABI'S WORK",
		"report.next":           "next:   fabi doctor  — evidence-based subtraction",
		"list.none":             "  (none)",
	},
	"ja": {
		"tagline":               "fabi — あなたのPCのAI資産を見る正直な目",
		"firstrun.scan":         "  `fabi scan` でこのPCにあるものを確認",
		"firstrun.help":         "  `fabi help` で全コマンドを表示",
		"hint.commands":         "コマンド: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — 正直なバージョン確認",
		"update.running":        "  現在:",
		"update.latest":         "  最新:",
		"update.behind":         "  → 新しいリリースがあります: %s",
		"update.uptodate":       "  → 最新です",
		"update.ahead":          "  → 最新リリースより新しい(devビルド)",
		"update.noauto":         "  ダウンロードして置き換え — fabiは自動インストールしません",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "コマンド:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: 台帳が空です — `fabi scan` を実行",
		"status.empty.scans":    "fabi: スキャン記録がありません — `fabi scan` を実行",
		"doctor.empty.ledger":   "FABI DOCTOR — 台帳が空です。まず `fabi scan` を実行",
		"doctor.empty.findings": "FABI DOCTOR — スキャンはありますが発見がまだ記録されていません",
		"doctor.title":          "FABI DOCTOR — 証拠に基づく推奨",
		"doctor.none":           "推奨はありません。全ての発見に来歴があります。",
		"doctor.note":           "  注: fabiはユーザーファイルを自動削除しません。\n      削除候補は確認後に手動で行ってください。",
		"scan.title":            "FABI SCAN — AI資産レポート",
		"report.title":          "FABI REPORT — あなたのAI資産・実測済み",
		"report.empty.ledger":   "FABI REPORT — 台帳が空です。まず `fabi scan` を実行",
		"report.empty.scans":    "FABI REPORT — 台帳にスキャンがありません。",
		"report.estate":         "資産 (最新スキャン)",
		"report.inference":      "推論 — 実測tok/s (予測ではなく実測)",
		"report.work":           "FABIの仕事",
		"report.next":           "次:   fabi doctor  — 証拠に基づく削減",
		"list.none":             "  (なし)",
	},
	"zh": {
		"tagline":               "fabi — 您的 PC AI 资产的诚实之眼",
		"firstrun.scan":         "  运行 `fabi scan` 查看此 PC 上有什么。",
		"firstrun.help":         "  运行 `fabi help` 查看所有命令。",
		"hint.commands":         "命令: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — 诚实的版本检查",
		"update.running":        "  当前:",
		"update.latest":         "  最新:",
		"update.behind":         "  → 有新版本可用: %s",
		"update.uptodate":       "  → 已是最新",
		"update.ahead":          "  → 比最新发布版本更新(dev 构建)",
		"update.noauto":         "  下载并替换 — fabi 不会自动安装",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "命令:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: 台账为空 — 运行 `fabi scan`",
		"status.empty.scans":    "fabi: 没有扫描记录 — 运行 `fabi scan`",
		"doctor.empty.ledger":   "FABI DOCTOR — 台账为空。请先运行 `fabi scan`",
		"doctor.empty.findings": "FABI DOCTOR — 已有扫描但尚未记录任何发现",
		"doctor.title":          "FABI DOCTOR — 基于证据的建议",
		"doctor.none":           "没有建议。每个发现都有出处。",
		"doctor.note":           "  注意: fabi 不会自动删除用户文件。\n        请检查后手动删除候选项。",
		"scan.title":            "FABI SCAN — AI 资产报告",
		"report.title":          "FABI REPORT — 您的 AI 资产,实测",
		"report.empty.ledger":   "FABI REPORT — 台账为空。请先运行 `fabi scan`",
		"report.empty.scans":    "FABI REPORT — 台账中没有扫描。",
		"report.estate":         "资产 (最新扫描)",
		"report.inference":      "推理 — 实测 tok/s (实测而非预测)",
		"report.work":           "FABI 的工作",
		"report.next":           "下一步: fabi doctor — 基于证据的削减",
		"list.none":             "  (无)",
	},
	"es": {
		"tagline":               "fabi — el ojo honesto sobre los activos de IA de tu PC",
		"firstrun.scan":         "  Ejecuta `fabi scan` para ver qué hay aquí.",
		"firstrun.help":         "  Ejecuta `fabi help` para todos los comandos.",
		"hint.commands":         "comandos: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — verificación honesta de versión",
		"update.running":        "  actual:",
		"update.latest":         "  última:",
		"update.behind":         "  → nueva versión disponible: %s",
		"update.uptodate":       "  → actualizado",
		"update.ahead":          "  → por delante de la última versión (build dev)",
		"update.noauto":         "  descarga y reemplaza — fabi nunca instala solo",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "comandos:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: ledger vacío — ejecuta `fabi scan`",
		"status.empty.scans":    "fabi: sin escaneos registrados — ejecuta `fabi scan`",
		"doctor.empty.ledger":   "FABI DOCTOR — el ledger está vacío. Ejecuta `fabi scan` primero.",
		"doctor.empty.findings": "FABI DOCTOR — hay escaneos pero aún sin hallazgos registrados.",
		"doctor.title":          "FABI DOCTOR — recomendaciones basadas en evidencia",
		"doctor.none":           "Sin recomendaciones. Cada hallazgo tiene procedencia.",
		"doctor.note":           "  nota: fabi nunca borra archivos del usuario automáticamente.\n        revisa los candidatos y bórralos manualmente.",
		"scan.title":            "FABI SCAN — informe de activos de IA",
		"report.title":          "FABI REPORT — tus activos de IA, medidos",
		"report.empty.ledger":   "FABI REPORT — el ledger está vacío. Ejecuta `fabi scan` primero.",
		"report.empty.scans":    "FABI REPORT — sin escaneos en el ledger.",
		"report.estate":         "ACTIVOS (último escaneo)",
		"report.inference":      "INFERENCIA — tok/s medidos (reales, no predichos)",
		"report.work":           "TRABAJO DE FABI",
		"report.next":           "siguiente: fabi doctor — sustracción basada en evidencia",
		"list.none":             "  (ninguno)",
	},
	"de": {
		"tagline":               "fabi — das ehrliche Auge auf dem KI-Bestand deines PCs",
		"firstrun.scan":         "  `fabi scan` ausführen, um zu sehen, was hier ist.",
		"firstrun.help":         "  `fabi help` für alle Befehle.",
		"hint.commands":         "Befehle: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — ehrliche Versionsprüfung",
		"update.running":        "  laufend:",
		"update.latest":         "  neueste:",
		"update.behind":         "  → neuere Version verfügbar: %s",
		"update.uptodate":       "  → auf dem neuesten Stand",
		"update.ahead":          "  → neuer als das letzte Release (Dev-Build)",
		"update.noauto":         "  herunterladen & ersetzen — fabi installiert nie automatisch",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "Befehle:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: Ledger leer — `fabi scan` ausführen",
		"status.empty.scans":    "fabi: keine Scans aufgezeichnet — `fabi scan` ausführen",
		"doctor.empty.ledger":   "FABI DOCTOR — Ledger ist leer. Zuerst `fabi scan` ausführen.",
		"doctor.empty.findings": "FABI DOCTOR — Scans vorhanden, aber noch keine Funde aufgezeichnet.",
		"doctor.title":          "FABI DOCTOR — evidenzbasierte Empfehlungen",
		"doctor.none":           "Keine Empfehlungen. Jeder Fund hat Herkunftsnachweis.",
		"doctor.note":           "  Hinweis: fabi löscht Benutzerdateien nie automatisch.\n           Kandidaten nach Prüfung manuell entfernen.",
		"scan.title":            "FABI SCAN — KI-Bestandsbericht",
		"report.title":          "FABI REPORT — dein KI-Bestand, gemessen",
		"report.empty.ledger":   "FABI REPORT — Ledger ist leer. Zuerst `fabi scan` ausführen.",
		"report.empty.scans":    "FABI REPORT — keine Scans im Ledger.",
		"report.estate":         "BESTAND (letzter Scan)",
		"report.inference":      "INFERENZ — gemessene tok/s (echt, nicht vorhergesagt)",
		"report.work":           "FABIS ARBEIT",
		"report.next":           "weiter:   fabi doctor — evidenzbasierte Reduktion",
		"list.none":             "  (keine)",
	},
	"fr": {
		"tagline":               "fabi — l'œil honnête sur les actifs IA de votre PC",
		"firstrun.scan":         "  Lancez `fabi scan` pour voir ce qui est ici.",
		"firstrun.help":         "  Lancez `fabi help` pour toutes les commandes.",
		"hint.commands":         "commandes : scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — vérification honnête de version",
		"update.running":        "  actuelle :",
		"update.latest":         "  dernière :",
		"update.behind":         "  → nouvelle version disponible : %s",
		"update.uptodate":       "  → à jour",
		"update.ahead":          "  → plus récent que la dernière version (build dev)",
		"update.noauto":         "  téléchargez et remplacez — fabi n'installe jamais seul",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "commandes :",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi : registre vide — lancez `fabi scan`",
		"status.empty.scans":    "fabi : aucun scan enregistré — lancez `fabi scan`",
		"doctor.empty.ledger":   "FABI DOCTOR — le registre est vide. Lancez `fabi scan` d'abord.",
		"doctor.empty.findings": "FABI DOCTOR — des scans existent mais aucun constat enregistré.",
		"doctor.title":          "FABI DOCTOR — recommandations fondées sur des preuves",
		"doctor.none":           "Aucune recommandation. Chaque constat a une provenance.",
		"doctor.note":           "  note : fabi ne supprime jamais les fichiers utilisateur.\n         supprimez les candidats manuellement après vérification.",
		"scan.title":            "FABI SCAN — rapport sur les actifs IA",
		"report.title":          "FABI REPORT — vos actifs IA, mesurés",
		"report.empty.ledger":   "FABI REPORT — le registre est vide. Lancez `fabi scan` d'abord.",
		"report.empty.scans":    "FABI REPORT — aucun scan dans le registre.",
		"report.estate":         "ACTIFS (dernier scan)",
		"report.inference":      "INFÉRENCE — tok/s mesurés (réels, pas prédits)",
		"report.work":           "LE TRAVAIL DE FABI",
		"report.next":           "suivant :  fabi doctor — soustraction fondée sur les preuves",
		"list.none":             "  (aucun)",
	},
	"ko": {
		"tagline":               "fabi — PC의 AI 자산을 보는 정직한 눈",
		"firstrun.scan":         "  `fabi scan`으로 여기 있는 것을 확인하세요.",
		"firstrun.help":         "  `fabi help`로 모든 명령을 보세요.",
		"hint.commands":         "명령: scan · doctor · run · serve · help",
		"update.title":          "FABI UPDATE — 정직한 버전 확인",
		"update.running":        "  실행 중:",
		"update.latest":         "  최신:",
		"update.behind":         "  → 새 버전이 있습니다: %s",
		"update.uptodate":       "  → 최신 상태입니다",
		"update.ahead":          "  → 최신 릴리스보다 앞섬(dev 빌드)",
		"update.noauto":         "  다운로드 후 교체 — fabi는 자동 설치하지 않습니다",
		"update.err":            "fabi update — %v",
		"usage.cmds":            "명령:",
		"usage.line":            "usage: fabi <command>",
		"status.empty.ledger":   "fabi: 장부가 비어 있습니다 — `fabi scan`을 실행하세요",
		"status.empty.scans":    "fabi: 기록된 스캔이 없습니다 — `fabi scan`을 실행하세요",
		"doctor.empty.ledger":   "FABI DOCTOR — 장부가 비어 있습니다. 먼저 `fabi scan`을 실행하세요.",
		"doctor.empty.findings": "FABI DOCTOR — 스캔은 있지만 아직 기록된 발견이 없습니다.",
		"doctor.title":          "FABI DOCTOR — 증거 기반 권장 사항",
		"doctor.none":           "권장 사항이 없습니다. 모든 발견에는 출처가 있습니다.",
		"doctor.note":           "  참고: fabi는 사용자 파일을 자동으로 삭제하지 않습니다.\n        후보는 검토 후 수동으로 제거하세요.",
		"scan.title":            "FABI SCAN — AI 자산 보고서",
		"report.title":          "FABI REPORT — AI 자산, 실측",
		"report.empty.ledger":   "FABI REPORT — 장부가 비어 있습니다. 먼저 `fabi scan`을 실행하세요.",
		"report.empty.scans":    "FABI REPORT — 장부에 스캔이 없습니다.",
		"report.estate":         "자산 (최신 스캔)",
		"report.inference":      "추론 — 실측 tok/s (예측이 아닌 실측)",
		"report.work":           "FABI의 작업",
		"report.next":           "다음:   fabi doctor — 증거 기반 감축",
		"list.none":             "  (없음)",
	},
}
