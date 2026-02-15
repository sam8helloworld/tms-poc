# TMS CLI

国際物流SCMプラットフォームのCLIツール。UseCaseの実行とデータ照会をコマンドラインから行えます。

## ビルド・実行

```bash
cd src

# ビルド
make build-cli        # bin/tms にバイナリ生成

# 直接実行
make run-cli ARGS="contract list --shipper-id <uuid>"

# または
go run ./cmd/tms <command> [flags]
```

## DB接続オプション

すべてのコマンドに共通のグローバルフラグでDB接続先を指定できます（デフォルト: localhost:5432/tms_db）。

```bash
tms --db-host localhost --db-port 5432 --db-user postgres --db-password postgres --db-name tms_db <command>
```

## コマンド一覧

### contract — サービス契約管理

```bash
# DRAFT契約を作成
tms contract create \
  --provider-id "550e8400-e29b-41d4-a716-446655440001" \
  --shipper-id "550e8400-e29b-41d4-a716-446655440002" \
  --bid-request-id "550e8400-e29b-41d4-a716-446655440003" \
  --valid-from 2026-04-01 \
  --valid-to 2027-03-31

# DRAFT契約を削除（CANCELLED化）
tms contract delete --contract-id <uuid>

# DRAFT契約の有効期間を変更
tms contract update-period \
  --contract-id <uuid> \
  --valid-from 2026-05-01 \
  --valid-to 2027-04-30

# 契約を取得
tms contract get <contract-id>

# 荷主の契約一覧を取得
tms contract list --shipper-id <uuid>
```

### tariff — 料金表管理

```bash
# 料金表を取得
tms tariff get <tariff-id>

# 契約に紐づく料金表一覧
tms tariff list --contract-id <uuid>

# DRAFT契約から料金表を削除
tms tariff remove --contract-id <uuid> --tariff-id <uuid>
```

### vendor — 物流業者照会

```bash
# 業者情報を取得
tms vendor get <vendor-id>
```

### rate — レート管理

```bash
# CONTRACTED契約の料金表をDRAFTレートに反映
tms rate apply-contract \
  --rate-id <uuid> \
  --contract-id <uuid>

# レートエントリのTariffIDを差し替え
tms rate update-entry-tariff \
  --rate-id <uuid> \
  --entry-id <uuid> \
  --contract-id <uuid> \
  --new-tariff-id <uuid>

# レートを取得
tms rate get <rate-id>

# 荷主のレート一覧
tms rate list --shipper-id <uuid>
```

### tracking — トラッキング管理

```bash
# トラッキングユニットを登録
tms tracking register \
  --shipment-id <uuid> \
  --tracking-number "ABCD1234567" \
  --number-type CONTAINER \
  --carrier-id <uuid> \
  --origin-id <uuid> \
  --dest-id <uuid> \
  --mode OCEAN

# 外部プロバイダーからイベントを同期
tms tracking sync --tracking-unit-id <uuid>

# トラッキングユニットを取得
tms tracking get <tracking-unit-id>

# トラッキング番号で検索
tms tracking get-by-number "ABCD1234567"
```

### document — 書類管理

```bash
# 書類をアップロード
tms document upload \
  --shipment-id <uuid> \
  --doc-type INVOICE \
  --origin SHIPPER \
  --file-name "invoice_202604.pdf" \
  --storage-uri "s3://bucket/invoice_202604.pdf" \
  --uploaded-by <uuid>

# 書類から構造化コンテンツを抽出
tms document extract --document-id <uuid>

# 書類を確認済みにする
tms document confirm --document-id <uuid>

# 書類を取得
tms document get <document-id>

# 出荷に紐づく書類一覧
tms document list --shipment-id <uuid>
```

### shipment — 出荷照会

```bash
# 出荷を取得
tms shipment get <shipment-id>

# 出荷番号で検索
tms shipment get-by-no "SHP-2026-001"
```

### location — ロケーション照会

```bash
# ロケーションを取得
tms location get <location-id>

# UN/LOCODEで検索
tms location get-by-unlocode "JPTYO"
```

### lane — レーン照会

```bash
# レーンを取得
tms lane get <lane-id>
```

### standard-route — 標準ルート照会

```bash
# 標準ルートを取得
tms standard-route get <route-id>

# 荷主の有効な標準ルート一覧
tms standard-route list --shipper-id <uuid>
```

### sop-definition — SOP定義照会

```bash
# SOP定義を取得
tms sop-definition get <definition-id>

# 有効なSOP定義一覧
tms sop-definition list
```

### sop-instance — SOPインスタンス照会

```bash
# SOPインスタンスを取得
tms sop-instance get <instance-id>

# 出荷に紐づくSOPインスタンスを取得
tms sop-instance get-by-shipment <shipment-id>
```

### event — ドメインイベント照会

```bash
# 集約のイベント履歴を取得
tms event list \
  --aggregate-id <uuid> \
  --aggregate-type TrackingUnit
```

## 出力形式

すべてのコマンドはJSON形式で出力されます。`jq` と組み合わせて利用できます。

```bash
tms contract list --shipper-id <uuid> | jq '.[].id'
tms rate get <uuid> | jq '.status'
```
