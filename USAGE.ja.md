# fabi — 実践ガイド

あなたのPCのAI資産を見る正直な目。これが実際の使い方です。

## 最初の60秒

```sh
fabi            # 1行ステータス(初回はイントロ)
fabi scan       # このPCのAIツール・モデル・サービス・GPUを全検出
fabi doctor     # 証拠ベースの推奨 — 何を消す/遊休化するか
fabi status     # いつでも1行でヘルス確認
```

これがループの全て。以下は任意の深掘りです。

## 日常のワークフロー

**「このマシンに何があるか?」**
`fabi scan` → `fabi report` — 資産インベントリ + 実測推移。

**「昨日から何が変わったか?」**
`fabi diff` — スキャン間で出現/消失したもの。
`fabi diff 5` — 5スキャン前と比較。

**「何を手放すべきか?」**
`fabi doctor` — remove / idle / investigate の推奨、
それぞれconfidence注記と証拠付き。

**「なぜそう言うのか?」**
`fabi why <target>` — 推奨を正当化する台帳イベント。
全ての推奨は実証拠にトレース可能。

**「モデルで何か実行する」**
`fabi run "prompt"` — 実測推論(tok/s, TTFT)。
`fabi run --sweep` — 全ライブモデルのベンチ。
`fabi goal "要約して" --run` — 目的をタスクに分解し、ローカルで
実現可能なものを実行。

**「VRAMを解放する」**
`fabi unload` — ロード済みモデルをアンロード。
`fabi auto` — 安全・可逆なアクションのみ適用(遊休アンロード)。
行動しない場合は `auto.skipped` を記録。

## 機械可読出力

主要コマンドは `--json` — スクリプト・監視用:

```sh
fabi status --json     # ヘルスをオブジェクトで
fabi doctor --json     # 推奨を配列で
fabi diff --json       # 資産差分をリストで
fabi scan --json       # 全発見をJSONで
fabi report --json     # ステートメントを数値で
fabi ledger stats --json
fabi why X --json      # 証拠チェーン
fabi update --json     # バージョンデルタ
```

## Universal API

`fabi serve` が同じ面を `127.0.0.1:11435` で公開 —
設計上ループバックのみ:

```
GET  /status /models /ledger /doctor /diff /why /pattern /update
POST /run /goal /unload /scan /pattern /import
```

## マルチPC資産

```sh
# 別のマシンで:
fabi ledger export > estate.jsonl
# ファイルを移し、このマシンで:
fabi ledger import estate.jsonl        # または POST /import
fabi ledger stats                      # "scans by os" で由来を表示
```

外部スキャンはverbatimで保存(来歴保持)され、ローカル推奨から
除外 — マージは資産を黙って混ぜない。

## 更新

```sh
fabi update     # あなたのバージョンと最新GitHubリリースを比較
```

fabiは自動ダウンロードしない — 差分とURLを報告するだけ。
あなたがバイナリを置き換える。deny-by-defaultはfabi自身にも適用。

## 言語

出力は10言語: en/ja/zh/es/de/fr/ko/pt/ru/it。
検出順: `FABI_LANG` > `LANG` > `LC_ALL` > OS UI言語 > en。
言語追加は `lang_<code>.go` にデータを追加するだけ。

## トラブルシューティング

| 症状 | 正直な答え |
|---|---|
| `fabi run` が「no live generative model」 | `ollama serve` 起動かLM Studioでモデルロード |
| `fabi auto` が「nothing to do」 | エンジン停止 or 全モデル使用中 — 両方正しい状態 |
| 初回実行でSmartScreen警告 | バイナリ未署名 — READMEのSHA256で検証;署名は計画中 |
| `fabi update` がGitHubに届かない | オフラインかプロキシ — 他は全て動作 |
| `fabi why X` が「no recommendation」 | Xへの推奨がない — 代わりに生の観測を表示 |

## 哲学、一行で

fabiは推測も隠蔽もしない: 何があるか、何が遊んでいるか、
何がVRAMを持っているか、何が変わったか、なぜそう言うかを記録し、
証明できないことは `UNVERIFIED` と言う。
