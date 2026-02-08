# hcpt

HCP Terraform の設定やワークスペース情報を取得する CLI ツール。

## インストール

### Go install

```bash
go install github.com/nnstt1/hcpt@latest
```

### GitHub Releases

[Releases](https://github.com/nnstt1/hcpt/releases) からバイナリをダウンロードできます。

## 認証

API トークンを以下のいずれかで設定します（上が優先）:

1. 環境変数 `TFE_TOKEN`
2. 設定ファイル `~/.hcpt.yaml` の `token` フィールド

```yaml
# ~/.hcpt.yaml
token: "your-api-token"
address: "https://app.terraform.io"  # デフォルト
org: "my-organization"
```

アドレスは環境変数 `TFE_ADDRESS` でも指定できます。

## 使い方

### Organization 一覧

```bash
hcpt org list
```

### Workspace 一覧

```bash
hcpt workspace list --org my-org

# 名前で検索
hcpt workspace list --org my-org --search "prod"

# エイリアス ws も利用可
hcpt ws list --org my-org
```

### Workspace 詳細

```bash
hcpt workspace show my-workspace --org my-org
```

### Run 履歴

```bash
hcpt run list --org my-org -w my-workspace
```

### 共通オプション

| フラグ | 説明 |
|--------|------|
| `--org` | Organization 名（設定ファイルや環境変数でも指定可） |
| `--json` | JSON 形式で出力 |
| `--config` | 設定ファイルパス（デフォルト: `~/.hcpt.yaml`） |

## 開発

```bash
# ビルド
make build

# テスト
make test

# Lint
make lint

# クリーン
make clean
```

## ライセンス

MIT
