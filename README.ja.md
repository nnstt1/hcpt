# hcpt

HCP Terraform の設定やワークスペース情報を取得する CLI ツール。

[English](README.md)

## インストール

### install.sh (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/nnstt1/hcpt/main/install.sh | sh
```

`INSTALL_DIR` 環境変数でインストール先を変更できます:

```bash
curl -fsSL https://raw.githubusercontent.com/nnstt1/hcpt/main/install.sh | INSTALL_DIR=~/.local/bin sh
```

### Go install

```bash
go install github.com/nnstt1/hcpt@latest
```

### GitHub Releases

[Releases](https://github.com/nnstt1/hcpt/releases) からバイナリをダウンロードできます。

## 認証

### HCP Terraform

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

### GitHub（--pr フラグ使用時）

`run show` で `--pr` フラグを使用する場合、GitHub トークンを以下の優先順位で探索します:

1. `gh auth token`（GitHub CLI 認証）
2. 環境変数 `GITHUB_TOKEN`
3. 設定ファイル `~/.hcpt.yaml` の `github-token` フィールド

```bash
# GitHub CLI で認証
gh auth login

# --pr フラグを使用
hcpt run show --pr 42 --repo owner/repo
```

## 使い方

### Organization

```bash
# Organization 一覧
hcpt org list

# Organization 詳細・契約プラン・Entitlements の表示
hcpt org show --org my-org
```

### Project

```bash
# Organization 内の Project 一覧
hcpt project list --org my-org
```

### Workspace

```bash
hcpt workspace list --org my-org

# 名前で検索
hcpt workspace list --org my-org --search "prod"

# エイリアス ws も利用可
hcpt ws list --org my-org

# Workspace 詳細
hcpt workspace show my-workspace --org my-org
```

### ドリフト検出

```bash
# ドリフトしているワークスペース一覧
hcpt drift list --org my-org

# 全ワークスペースのドリフト状態を表示
hcpt drift list --all --org my-org

# 特定ワークスペースのドリフト詳細
hcpt drift show my-workspace --org my-org
```

### Run

```bash
# Workspace の Run 履歴
hcpt run list --org my-org -w my-workspace

# ステータスでフィルター
hcpt run list --org my-org -w my-workspace --status applied

# 複数ステータスでフィルター（カンマ区切り）
hcpt run list --org my-org -w my-workspace --status planned,applied

# Run 詳細
hcpt run show run-abc123

# Workspace の最新 Run を表示
hcpt run show --org my-org -w my-workspace

# Run の完了を監視
hcpt run show run-abc123 --watch

# GitHub PR から Run を表示
hcpt run show --pr 42 --repo owner/repo

# GitHub PR から Run を監視
hcpt run show --pr 42 --repo owner/repo --watch

# GitHub PR から Run を表示（monorepo で複数ワークスペースがある場合）
hcpt run show --pr 42 --repo owner/repo -w my-workspace

# Apply ログを表示
hcpt run logs run-abc123

# エラー行のみ表示
hcpt run logs run-abc123 --error-only

# Workspace の最新 Run の Apply ログを表示
hcpt run logs --org my-org -w my-workspace
```

### Variable

```bash
# Workspace の変数一覧
hcpt variable list --org my-org -w my-workspace

# エイリアス var も利用可
hcpt var list --org my-org -w my-workspace

# 変数の作成/更新（upsert）
hcpt variable set MY_KEY "my-value" --org my-org -w my-workspace

# 環境変数として設定
hcpt variable set AWS_REGION "us-east-1" --org my-org -w my-workspace --category env

# Sensitive 変数として設定
hcpt variable set SECRET_KEY "s3cret" --org my-org -w my-workspace --sensitive

# HCL 変数として設定
hcpt variable set my_list '["a","b"]' --org my-org -w my-workspace --hcl

# 変数の削除
hcpt variable delete MY_KEY --org my-org -w my-workspace
```

### 設定管理

```bash
# デフォルトの Organization を設定
hcpt config set org my-org

# API アドレスを設定
hcpt config set address https://tfe.example.com

# API トークンを設定
hcpt config set token your-api-token

# 設定値の取得
hcpt config get org

# 全設定値の一覧（トークンはマスク表示）
hcpt config list
```

値は `~/.hcpt.yaml` に保存されます。既存の設定値は保持されます。

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
