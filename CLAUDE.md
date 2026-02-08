# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

HCP Terraform の設定やワークスペース情報を取得する CLI ツール。

## 技術スタック

- **言語**: Go
- **モジュール名**: `github.com/nnstt1/hcpt`
- **CLI フレームワーク**: [Cobra](https://github.com/spf13/cobra)
- **設定管理**: [Viper](https://github.com/spf13/viper)
- **API クライアント**: [go-tfe](https://github.com/hashicorp/go-tfe) (HCP Terraform / Terraform Enterprise 公式 Go クライアント)
- **Linter**: golangci-lint
- **リリース**: GoReleaser
- **CI**: GitHub Actions

## コマンド体系

```
hcpt
├── org list          # Organization 一覧を取得
├── workspace list    # Organization 内の Workspace 一覧を取得
├── workspace show    # 特定の Workspace の詳細情報を表示
└── run list          # Workspace の Run 履歴を表示
```

## 出力形式

- デフォルト: テーブル形式
- `--json` フラグ: JSON 形式

## 認証

- 環境変数 `TFE_TOKEN` または設定ファイル `~/.hcpt.yaml` から API トークンを読み込み
- Viper で環境変数と設定ファイルの優先順位を管理（環境変数 > 設定ファイル）

## ビルド・開発コマンド

```bash
# ビルド
go build -o hcpt .

# テスト
go test ./...

# 単一パッケージのテスト
go test ./internal/cmd/workspace/

# 単一テストの実行
go test ./internal/cmd/workspace/ -run TestWorkspaceList

# Lint
golangci-lint run

# 依存関係の整理
go mod tidy
```

## アーキテクチャ

```
├── main.go                  # エントリポイント
├── internal/
│   ├── cmd/                 # Cobra コマンド定義
│   │   ├── root.go          # ルートコマンド（Viper 初期化含む）
│   │   ├── org/             # org サブコマンド
│   │   ├── workspace/       # workspace サブコマンド
│   │   └── run/             # run サブコマンド
│   ├── client/              # go-tfe クライアントのラッパー
│   └── output/              # テーブル / JSON 出力のフォーマッタ
├── .golangci.yml            # golangci-lint 設定
├── .goreleaser.yml          # GoReleaser 設定
└── .github/workflows/       # GitHub Actions CI
```

- `internal/` 配下にコードを配置し、外部パッケージからのインポートを防ぐ
- 各サブコマンドはディレクトリで分離し、コマンド登録は `init()` で行う
- `internal/client/` で go-tfe クライアントの初期化とトークン取得を一元管理
- `internal/output/` でテーブル/JSON の出力切り替えロジックを共通化
