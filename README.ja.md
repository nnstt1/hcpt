# hcpt

HCP Terraform の設定やワークスペース情報を取得する CLI ツール。

[English](README.md)

## インストール

### Go install

```bash
go install github.com/nnstt1/hcpt@latest
```

### GitHub Releases

[Releases](https://github.com/nnstt1/hcpt/releases) からバイナリをダウンロードできます。

## 認証

API トークンを以下の優先順位で探索します:

1. 環境変数 `TFE_TOKEN`
2. 設定ファイル `~/.hcpt.yaml` の `token` フィールド
3. Terraform CLI の認証情報（以下の順で探索）
   1. 環境変数 `TF_TOKEN_<hostname>`（ホスト名のドット・ハイフンをアンダースコアに変換。例: `TF_TOKEN_app_terraform_io`）
   2. `~/.terraform.d/credentials.tfrc.json`（`terraform login` が生成する JSON）
   3. `~/.terraformrc`（HCL 形式の credentials ブロック）

`terraform login` で認証済みであれば、追加の設定なしで hcpt を利用できます。

```bash
# terraform login で認証済みの場合、そのまま利用可能
terraform login
hcpt org list
```

```yaml
# ~/.hcpt.yaml
token: "your-api-token"
address: "https://app.terraform.io"  # デフォルト
org: "my-organization"
```

アドレスは環境変数 `TFE_ADDRESS` でも指定できます。

`~/.terraformrc` のパスは環境変数 `TF_CLI_CONFIG_FILE` で上書きできます。

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
