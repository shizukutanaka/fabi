# fabi

**あなたのPCのAI資産を見る正直な目** — 1コマンドで「このPCで何の
AIが動いているか、何が遊んでいるか、何を安全に手放せるか」を答えます。
vibesではなく、falsified claims(反証可能な主張)で。

```sh
fabi          # 初回: イントロ / 以降: 1行ステータス
fabi scan     # AIツール・モデル・サービス・GPUを全検出
fabi doctor   # 証拠ベースの推奨 — 何を消す/遊休化するか
fabi status   # いつでも1行でヘルス確認
```

[English README](README.md)

## なぜfabiか

- **台帳が唯一の真実** — `~/.fabi/ledger.db` にappend-onlyイベント。
  全ての状態はビュー。推測でなく証拠。
- **削除は常に手動** — fabiは自動で消さない。候補を示し、あなたが決める。
- **全claimにテスト** — `fabi selftest` が全主張を反証
  (30 PASS · 0 FAIL · 4 UNVERIFIED)。
- **オフライン・ゼロ設定** — 単一バイナリ ~8MB、設定不要、
  ループバックのみ(唯一の外部通信は `fabi update` の明示実行時)。
- **--json で機械可読** — scan/status/doctor/report/diff/why/ledger
  全てJSON出力可能。

## 主なコマンド

```text
fabi scan       資産検出(6プローブ: process/port/fs/pkg/gpu/models)
fabi doctor     証拠ベース推奨 + evidence_ids
fabi why X      推奨の証拠チェーン
fabi diff       前回スキャンからの差分
fabi report     月次ステートメント(実測)
fabi status     1行ヘルス + GPU/VRAM
fabi run        実プロンプト実行・実tok/s計測
fabi goal "..." 目的→タスク分解→物理判定
fabi auto       安全な推奨のみ適用(自動化の種)
fabi serve      13エンドポイントJSON API(127.0.0.1のみ)
fabi ledger     append-onlyイベントのstats/events/export/import
fabi update     正直なバージョン確認(自動インストールなし)
fabi selftest   反証エンジン — 全claimにテスト
```

## 言語

出力は7言語対応(en/ja/zh/es/de/fr/ko)。`FABI_LANG`、`LANG`、
`LC_ALL` 環境変数、またはWindows UI言語で自動検出。
言語追加は `lang.go` にデータを追加するだけ。

## インストール

```powershell
# winget(申請中)または直接ダウンロード:
# https://github.com/shizukutanaka/fabi/releases/latest
winget install fabi        # 公開後
```

または [releases](https://github.com/shizukutanaka/fabi/releases) から
`fabi-windows-amd64.exe` をダウンロードして PATH の通った場所へ。

## ドキュメント

- [USAGE.md](USAGE.md) — 実践ガイド(英語)
- [API.md](API.md) — Universal APIエンドポイント
- [SECURITY.md](SECURITY.md) — セキュリティポリシー
- [SPEC_v0.7.md](SPEC_v0.7.md) — 設計仕様
- [CONTRIBUTING.md](CONTRIBUTING.md) — 貢献ガイド

MIT License.
