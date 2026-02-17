# POC シナリオ一覧

このディレクトリには、TMS POCの業務フローを端から端まで実行・検証するシナリオが格納されています。

## 実行前の準備

```bash
# 1. DBを起動してマイグレーションを適用
cd src && make docker-up && make migrate-up

# 2. (任意) DBデータをリセット
cd src && go run ./cmd/tms db reset --confirm
```

## シナリオの実行

```bash
# シナリオ一覧を表示
cd src && go run ./cmd/tms scenario list

# シナリオを実行
cd src && go run ./cmd/tms scenario run <scenario-name>
```

---

## シナリオ一覧

### `sourcing-bid` — Sourcing入札フロー

**概要**: 3社のフォワーダー（FWD）を対象に10ルートの料金を入札で収集し、ルート毎に最安値の業者を選定して契約を成立させ、社内レートに反映するまでの一連の業務フローを実行します。

#### 業務フロー（7ステップ）

| ステップ | 内容 | UseCase / 処理 |
|---------|------|---------------|
| Setup | マスターデータ作成 | LocationRepo・VendorRepo 直接操作 |
| Step 1 | 3社とのDRAFT契約作成 | `CreateBidContractUseCase` |
| Step 2 | 各社 1 Tariff（10ルート分のLineItemをまとめて）登録 | `RegisterTariffDirectUseCase` |
| Step 3 | ルート毎の料金比較（表示） | 集計・表示ロジック |
| Step 4 | 最安業者の契約をCONTRACTED化 | `AwardBidContractUseCase` |
| Step 5 | DRAFTレート作成 | `CreateRateUseCase` |
| Step 6 | 契約TariffのLineItemをレートEntryに反映（ルート別） | `ApplyContractToRateUseCase` (ACL経由) |
| Step 7 | レートをACTIVE化 | `ActivateRateUseCase` |

#### Setup で作成するマスターデータ

**拠点（11箇所）**

| 名称 | UN/LOCODE | 国 |
|-----|-----------|---|
| Tokyo | JPTYO | JP |
| Yokohama | JPYOK | JP |
| Shanghai | CNSHA | CN |
| Singapore | SGSIN | SG |
| Bangkok | THBKK | TH |
| Jakarta | IDJKT | ID |
| Ho Chi Minh | VNSGN | VN |
| Los Angeles | USLAX | US |
| Rotterdam | NLRTM | NL |
| Hamburg | DEHAM | DE |
| Dubai | AEDXB | AE |

**業者（3社）**

| 名称 | 種別 |
|-----|------|
| FWD Alpha | FORWARDER |
| FWD Beta | FORWARDER |
| FWD Gamma | FORWARDER |

#### 対象ルート（10ルート、全て海上輸送）

| # | ルート | 基準価格（目安） |
|---|-------|----------------|
| 1 | Tokyo → Shanghai | $1,200 |
| 2 | Tokyo → Singapore | $800 |
| 3 | Yokohama → Los Angeles | $2,500 |
| 4 | Shanghai → Rotterdam | $1,800 |
| 5 | Singapore → Hamburg | $2,000 |
| 6 | Bangkok → Dubai | $1,500 |
| 7 | Jakarta → Singapore | $600 |
| 8 | Ho Chi Minh → Tokyo | $1,000 |
| 9 | Shanghai → Los Angeles | $3,000 |
| 10 | Rotterdam → Dubai | $1,600 |

> 各社の実際の料金は基準価格に±30%のランダム変動が加わります。毎回異なる結果になります。

#### 出力例

```
=== Sourcing Bid Scenario ===

[Setup] Creating 11 locations... done
[Setup] Creating 3 vendors... done

[Step 1] Creating bid contracts...
  FWD Alpha:   contract a1b2c3d4 (DRAFT)
  FWD Beta:    contract e5f6g7h8 (DRAFT)
  FWD Gamma:   contract i9j0k1l2 (DRAFT)

[Step 2] Registering tariffs (1 tariff per FWD × 10 line items)...
  FWD Alpha:   1 tariff registered (10 line items)
  FWD Beta:    1 tariff registered (10 line items)
  FWD Gamma:   1 tariff registered (10 line items)

[Step 3] Comparing tariffs per route...
  Route                              | FWD Alpha     | FWD Beta      | FWD Gamma     | Winner
  -----------------------------------+--------------+--------------+--------------+--------------
  Tokyo → Shanghai (OCEAN)          | $1100        | $980 ★       | $1350        | FWD Beta
  Tokyo → Singapore (OCEAN)         | $750 ★       | $920         | $870         | FWD Alpha
  ...

[Step 4] Awarding contracts...
  FWD Alpha:   CONTRACTED (3 routes won)
  FWD Beta:    CONTRACTED (5 routes won)
  FWD Gamma:   CONTRACTED (2 routes won)

[Step 5] Creating rate "2026 H1 Rate"... done (DRAFT)

[Step 6] Applying contracts to rate...
  → ContractID a1b2c3d4 の 3 エントリを適用（累計: 3）
  → ContractID e5f6g7h8 の 5 エントリを適用（累計: 8）
  → ContractID i9j0k1l2 の 2 エントリを適用（累計: 10）

  ┌─ [レートカード] ルート別レート一覧 ────────────────────────────────────
  │ Route                              │ Provider    │ Charge  │ UnitPrice
  │────────────────────────────────────┼─────────────┼─────────┼────────────
  │ Tokyo → Shanghai (OCEAN)           │ FWD Beta    │ OFT     │ $980.00 USD
  │ Tokyo → Singapore (OCEAN)          │ FWD Alpha   │ OFT     │ $750.00 USD
  │ ...                                │ ...         │ ...     │ ...
  └─ 計 10 エントリ（ルート別最安レート）

[Step 7] Activating rate... done (ACTIVE)

=== Scenario Complete ===
```

---

### `rate-baf-update` — BAF更新によるレート更新

**概要**: レートに登録されたTariffに対して、業者からBAF（Bunker Adjustment Factor / 燃油サーチャージ）の更新通知が届いた場合に、契約の料金表を改定しレートのエントリを差し替えるまでの業務フローを実行します。

#### 業務フロー（8ステップ）

| ステップ | 内容 | UseCase / 処理 |
|---------|------|-----------------|
| Setup | マスターデータ作成（2拠点・1業者） | LocationRepo・VendorRepo 直接操作 |
| Step 1 | FWD AlphaとのDRAFT契約作成 | `CreateBidContractUseCase` |
| Step 2 | Tariff登録（OFT + BAF × 2ルート = 4 LineItems） | `RegisterTariffDirectUseCase` |
| Step 3 | 契約をCONTRACTED化 | `AwardBidContractUseCase` |
| Step 4 | DRAFTレート作成 | `CreateRateUseCase` |
| Step 5 | 契約のTariffをレートに反映 | `ApplyContractToRateUseCase` (ACL経由) |
| Step 6 | **BAF更新通知 → Tariff v2作成（BAF金額変更）** | `AmendContractTariffDirectUseCase` |
| Step 7 | **レートのBAFエントリをv2のLineItemに差し替え** | `UpdateRateEntryTariffUseCase` |
| Step 8 | レートをACTIVE化 | `ActivateRateUseCase` |

#### Setup で作成するマスターデータ

**拠点（2箇所）**

| 名称 | UN/LOCODE | 国 |
|-----|-----------|---|
| Tokyo | JPTYO | JP |
| Shanghai | CNSHA | CN |

**業者（1社）**

| 名称 | 種別 |
|-----|------|
| FWD Alpha | FORWARDER |

#### Tariff構成（v1 → v2）

| ルート | 費目 | v1 価格 | v2 価格 | 変動 |
|-------|------|---------|---------|------|
| Tokyo → Shanghai | OFT | $1,200 | $1,200 | — |
| Tokyo → Shanghai | BAF | $350 | $420 | +$70 (↑20%) |
| Shanghai → Tokyo | OFT | $1,100 | $1,100 | — |
| Shanghai → Tokyo | BAF | $320 | $380 | +$60 (↑19%) |

> BAFのみ値上がりし、OFT（海上運賃）は据え置きです。

#### 出力例

```
=== Rate BAF Update Scenario ===

[Setup] Creating 2 locations... done
[Setup] Creating 1 vendor... done

[Step 1] Creating bid contract...
  → FWD Alpha: contract a1b2c3d4 作成（DRAFT）

[Step 2] Registering tariff (OFT + BAF × 2 routes = 4 line items)...
  → 1 tariff registered (4 line items)

[Step 3] Awarding contract...
  → FWD Alpha: DRAFT → CONTRACTED

[Step 4] Creating rate "2026 H1 Rate"... done

[Step 5] Applying contract to rate...
  → 4 エントリを適用

  ┌─ [レートカード] レート一覧（BAF更新前） ──────────
  │ Route          │ Provider    │ Charge  │ UnitPrice
  │ TYO→SHA        │ FWD Alpha   │ OFT     │ $1200.00 USD
  │ TYO→SHA        │ FWD Alpha   │ BAF     │ $350.00 USD
  │ SHA→TYO        │ FWD Alpha   │ OFT     │ $1100.00 USD
  │ SHA→TYO        │ FWD Alpha   │ BAF     │ $320.00 USD
  └─ 計 4 エントリ

[Step 6] BAF update notification received → Creating tariff v2...
  ※ 業者から燃油サーチャージ（BAF）の改定通知が届きました

  ┌─ [料金表画面] BAF改定前後の比較 ──────────
  │ Route              │ Code    │ v1 Price   │ v2 Price   │ Change
  │ Tokyo → Shanghai   │ BAF     │ $350       │ $420       │ +$70 (↑20%)
  │ Shanghai → Tokyo   │ BAF     │ $320       │ $380       │ +$60 (↑19%)
  └─ Tariff v1 → v2

[Step 7] Updating rate entries with new BAF tariff...
  → Entry xxxx: BAF Tokyo → Shanghai → Tariff v2
  → Entry yyyy: BAF Shanghai → Tokyo → Tariff v2
  → 2 件のBAFエントリを更新

[Step 8] Activating rate... done
  Status: ACTIVE ← BAF改定反映済み

=== Scenario Complete ===
```

### `rate-simulation` — レートコストシミュレーション

**概要**: 入札で確定したACTIVEレートに対して、輸送前にルート・貨物条件（数量・重量・容積）を指定してコストをシミュレーションします。`SimulateRateCostUseCase` は ACL パターンで `TariffCalculator` インターフェースを経由し、Sourcing BC の `Tariff.CalculateCharges()` に料金計算を委譲します。未登録ルートは「輸送不可」として表示されます。

#### 業務フロー（8ステップ）

| ステップ | 内容 | UseCase / 処理 |
|---------|------|-----------------|
| Setup | マスターデータ作成（3拠点・2業者） | LocationRepo・VendorRepo 直接操作 |
| Step 1 | 2社とのDRAFT契約作成 | `CreateBidContractUseCase` |
| Step 2 | 各社 Tariff登録（OFT/BAF/THC/CFS × 3ルート） | `RegisterTariffDirectUseCase` |
| Step 3 | ルート別料金比較・最安業者選定 | 集計・表示ロジック |
| Step 4 | 契約Award | `AwardBidContractUseCase` |
| Step 5 | DRAFTレート作成 | `CreateRateUseCase` |
| Step 6 | 契約のTariffをレートに反映 | `ApplyContractToRateUseCase` |
| Step 7 | レートをACTIVE化 | `ActivateRateUseCase` |
| Step 8 | **レートコストシミュレーション** | **`SimulateRateCostUseCase`** |

#### Setup で作成するマスターデータ

**拠点（3箇所）**

| 名称 | UN/LOCODE | 国 |
|-----|-----------|---|
| Tokyo | JPTYO | JP |
| Shanghai | CNSHA | CN |
| Singapore | SGSIN | SG |

**業者（2社）**

| 名称 | 種別 |
|-----|------|
| FWD Alpha | FORWARDER |
| FWD Beta | FORWARDER |

#### 対象ルート（3ルート、全て海上輸送）

| # | ルート | OFT基準 (Flat) | BAF基準 (Flat) | THC単価 (Expr) | CFS基本/単価 (Comp) |
|---|-------|---------------|---------------|---------------|-------------------|
| 1 | Tokyo → Shanghai | $1,200 | $350 | $0.50/kg | $150 + $0.02/kg |
| 2 | Shanghai → Singapore | $800 | $200 | $0.40/kg | $120 + $0.015/kg |
| 3 | Tokyo → Singapore | $1,500 | $400 | $0.60/kg | $180 + $0.025/kg |

> 各社の実際の料金は基準価格に±20%のランダム変動が加わります。
> - **OFT/BAF**: Flat Strategy (Amount × Quantity)
> - **THC**: Expression Strategy (Chargeable Weight × Rate)
> - **CFS**: Composite Strategy (Flat Base + Weight × Rate)

#### シミュレーション対象（Step 8）

| # | ルート条件 | 貨物条件 | 期待される結果 |
|---|----------|---------|---------------|
| 1 | Tokyo → Shanghai (OCEAN) | 1本/18,000kg/30m³ | ✓ 輸送可能（Tariff計算で金額算出） |
| 2 | Tokyo → Singapore (OCEAN) | 1本/18,000kg/30m³ | ✓ 輸送可能（Tariff計算で金額算出） |
| 3 | Singapore → Tokyo (OCEAN) | 1本/18,000kg/30m³ | ✗ 輸送不可（レート未登録ルート） |

#### 出力例

```
=== Rate Simulation Scenario ===

[Setup] Creating 3 locations... done
[Setup] Creating 2 vendors... done

[Step 1] Creating bid contracts...
  → FWD Alpha:   contract a1b2c3d4 作成（DRAFT）
  → FWD Beta:    contract e5f6g7h8 作成（DRAFT）

[Step 2] Registering tariffs (OFT + BAF × 3 routes per FWD)...
  → FWD Alpha:   1 tariff registered (6 line items)
  → FWD Beta:    1 tariff registered (6 line items)

[Step 3] Comparing tariffs per route (OFT + BAF total)...
  ┌─ [入札比較画面] ルート別料金比較 ──────────────────────
  │ Route                         │ FWD Alpha        │ FWD Beta         │ Winner
  │ Tokyo → Shanghai (OCEAN)      │ $1480 ★          │ $1620            │ FWD Alpha
  │ Shanghai → Singapore (OCEAN)  │ $1050            │ $960 ★           │ FWD Beta
  │ Tokyo → Singapore (OCEAN)     │ $1830            │ $1780 ★          │ FWD Beta
  └─ ★ = 最安値（OFT + BAF合計）

[Step 4] Awarding contracts...
  → FWD Alpha:   DRAFT → CONTRACTED (1 routes won)
  → FWD Beta:    DRAFT → CONTRACTED (2 routes won)

[Step 5] Creating rate "2026 H1 Rate"... done
[Step 6] Applying contracts to rate...
  → ContractID a1b2c3d4 の 2 エントリを適用（累計: 2）
  → ContractID e5f6g7h8 の 4 エントリを適用（累計: 6）

[Step 7] Activating rate... done

[Step 8] Simulating rate cost for transport routes...
  ※ 輸送前のコスト見積もりを実施します（貨物条件: 1本 20DC / 18,000kg / 30m³）

  Rate: 2026 H1 Rate (Status: ACTIVE)

  ┌─ [シミュレーション結果] Tokyo → Shanghai (OCEAN) ✓ 輸送可能 ────
  │ Charge    Category      Amount
  │ ----------------------------------------
  │ OFT       FREIGHT       $1130.00 USD
  │ BAF       SURCHARGE     $350.00 USD
  │ THC       HANDLING      $145.00 USD
  │ CFS       FREIGHT       $510.00 USD
  │ ----------------------------------------
  │ 合計見積額: $2135.00 USD
  └──────────────────────────────────────────────

  ┌─ [シミュレーション結果] Tokyo → Singapore (OCEAN) ✓ 輸送可能 ────
  │ Charge    Category      Amount
  │ ----------------------------------------
  │ OFT       FREIGHT       $1420.00 USD
  │ BAF       SURCHARGE     $360.00 USD
  │ THC       HANDLING      $180.00 USD
  │ CFS       FREIGHT       $600.00 USD
  │ ----------------------------------------
  │ 合計見積額: $2560.00 USD
  └──────────────────────────────────────────────

  ┌─ [シミュレーション結果] Singapore → Tokyo (OCEAN) ✗ 輸送不可 ────
  │ 該当するレートエントリが見つかりません
  │ → このルートは現在のレートではカバーされていません
  └──────────────────────────────────────────────

  シミュレーション完了: 2/3 ルートが輸送可能

=== Scenario Complete ===
```

---

### `shipment-document` — 輸送案件 書類E2Eフロー

**概要**: 輸送案件を作成し、海上輸送（Tokyo → Shanghai）のE2Eで発生する書類を登録・確認してマイルストーンを蓄積し、輸送完了まで管理するシナリオを実行します。書類の確認がマイルストーン記録とステータス自動遷移をトリガーします。

#### 業務フロー（11ステップ）

| ステップ | 業務 | DocType | Origin | マイルストーン | Status遷移 |
|---------|------|---------|--------|------------|-----------|
| Setup | マスターデータ作成（2拠点・1業者） | - | - | - | - |
| Step 1 | 輸送案件作成 | - | - | ShipmentCreated | → PLANNED |
| Step 2 | ブッキング確認 | BOOKING_CONFIRMATION | PROVIDER | BOOKING_CONFIRMED | → BOOKED |
| Step 3 | S/I発行 | SHIPPING_INSTRUCTION | SHIPPER | SHIPPING_INSTRUCTION_ISSUED | - |
| Step 4 | Commercial Invoice登録 | COMMERCIAL_INVOICE | SHIPPER | - | - |
| Step 5 | Packing List登録 | PACKING_LIST | SHIPPER | - | - |
| Step 6 | B/L受領・船積確認 | BILL_OF_LADING | PROVIDER | SHIPPED | → IN_TRANSIT |
| Step 7 | 輸出通関 | CUSTOMS_DECLARATION | PROVIDER | CUSTOMS_EXPORT_CLEARED | - |
| Step 8 | 到着通知 | ARRIVAL_NOTICE | PROVIDER | ARRIVED | - |
| Step 9 | 輸入通関 | CUSTOMS_DECLARATION | PROVIDER | CUSTOMS_IMPORT_CLEARED | - |
| Step 10 | 配送指図・納品 | DELIVERY_ORDER | PROVIDER | DELIVERED | → COMPLETED |
| Step 11 | Freight Invoice受領 | FREIGHT_INVOICE | PROVIDER | INVOICE_RECEIVED | - |

#### Setup で作成するマスターデータ

**拠点（2箇所）**

| 名称 | UN/LOCODE | 国 |
|-----|-----------|---|
| Tokyo | JPTYO | JP |
| Shanghai | CNSHA | CN |

**業者（1社）**

| 名称 | 種別 |
|-----|------|
| Ocean Express Co. | CARRIER |

#### ステータス自動遷移ルール

マイルストーン記録時に、以下のルールで Shipment ステータスが自動遷移します：

- `BOOKING_CONFIRMED` → PLANNED から BOOKED へ
- `SHIPPED` → PLANNED/BOOKED から IN_TRANSIT へ
- `DELIVERED` → 任意の状態から COMPLETED へ

#### 出力例

```
=== Shipment Document E2E Scenario ===

[Setup] Creating locations and vendor... done

  ┌─ [Setup] マスターデータ ─────────────────────────────
  │ Origin:      Tokyo (JPTYO)  ID: ebefec2a
  │ Destination: Shanghai (CNSHA) ID: d76e6ca4
  │ Vendor:      Ocean Express Co.  ID: e56658fc
  └──────────────────────────────────────────────────────

[Step 1] 輸送案件作成
  → Shipment SHP-2026-0001 作成完了

  ┌─ [輸送案件] ──────────────────────────────────────────
  │ ID:     d11488c1
  │ No:     SHP-2026-0001
  │ Status: PLANNED
  └──────────────────────────────────────────────────────

[Step 2] ブッキング確認
  → 書類アップロード: booking_confirmation.pdf (ID: 0b4e4100)
  → 書類確認完了: BOOKING_CONFIRMATION → CONFIRMED
  → マイルストーン記録: BOOKING_CONFIRMED → Status: BOOKED

[Step 3] S/I発行
  → 書類アップロード: shipping_instruction.pdf (ID: 17728b67)
  → 書類確認完了: SHIPPING_INSTRUCTION → CONFIRMED
  → マイルストーン記録: SHIPPING_INSTRUCTION_ISSUED → Status: BOOKED

[Step 4] Commercial Invoice登録
  → 書類アップロード: commercial_invoice.pdf (ID: d17c38c1)
  → 書類確認完了: COMMERCIAL_INVOICE → CONFIRMED

[Step 5] Packing List登録
  → 書類アップロード: packing_list.pdf (ID: a431df3a)
  → 書類確認完了: PACKING_LIST → CONFIRMED

[Step 6] B/L受領・船積確認
  → 書類アップロード: bill_of_lading.pdf (ID: 1328b1d1)
  → 書類確認完了: BILL_OF_LADING → CONFIRMED
  → マイルストーン記録: SHIPPED → Status: IN_TRANSIT

[Step 7] 輸出通関
  → 書類アップロード: export_customs_declaration.pdf (ID: 391624c0)
  → 書類確認完了: CUSTOMS_DECLARATION → CONFIRMED
  → マイルストーン記録: CUSTOMS_EXPORT_CLEARED → Status: IN_TRANSIT

[Step 8] 到着通知
  → 書類アップロード: arrival_notice.pdf (ID: 6b44ab9f)
  → 書類確認完了: ARRIVAL_NOTICE → CONFIRMED
  → マイルストーン記録: ARRIVED → Status: IN_TRANSIT

[Step 9] 輸入通関
  → 書類アップロード: import_customs_declaration.pdf (ID: 0387e4b1)
  → 書類確認完了: CUSTOMS_DECLARATION → CONFIRMED
  → マイルストーン記録: CUSTOMS_IMPORT_CLEARED → Status: IN_TRANSIT

[Step 10] 配送指図・納品
  → 書類アップロード: delivery_order.pdf (ID: 32e0d316)
  → 書類確認完了: DELIVERY_ORDER → CONFIRMED
  → マイルストーン記録: DELIVERED → Status: COMPLETED

[Step 11] Freight Invoice受領
  → 書類アップロード: freight_invoice.pdf (ID: 106d416f)
  → 書類確認完了: FREIGHT_INVOICE → CONFIRMED
  → マイルストーン記録: INVOICE_RECEIVED → Status: COMPLETED

  ┌─ [最終結果] 輸送案件 書類・マイルストーン一覧 ────────────
  │ Shipment ID: d11488c1
  │
  │  Step  DocType                   Origin    Milestone                    Status
  │  ------------------------------------------------------------------------------------------
  │   1    -                         -         ShipmentCreated              PLANNED
  │   2    BOOKING_CONFIRMATION      PROVIDER  BOOKING_CONFIRMED            BOOKED
  │   3    SHIPPING_INSTRUCTION      SHIPPER   SHIPPING_INSTRUCTION_ISSUED  BOOKED
  │   4    COMMERCIAL_INVOICE        SHIPPER   -                            BOOKED
  │   5    PACKING_LIST              SHIPPER   -                            BOOKED
  │   6    BILL_OF_LADING            PROVIDER  SHIPPED                      IN_TRANSIT
  │   7    CUSTOMS_DECLARATION       PROVIDER  CUSTOMS_EXPORT_CLEARED       IN_TRANSIT
  │   8    ARRIVAL_NOTICE            PROVIDER  ARRIVED                      IN_TRANSIT
  │   9    CUSTOMS_DECLARATION       PROVIDER  CUSTOMS_IMPORT_CLEARED       IN_TRANSIT
  │  10    DELIVERY_ORDER            PROVIDER  DELIVERED                    COMPLETED
  │  11    FREIGHT_INVOICE           PROVIDER  INVOICE_RECEIVED             COMPLETED
  └──────────────────────────────────────────────────────────────────────────────────

=== Scenario Complete ===
```

---

## 新しいシナリオの追加方法

1. このディレクトリに `my_scenario.go` を作成する
2. `Scenario` インターフェースを実装する

```go
package scenarios

type MyScenario struct{}

func (s *MyScenario) Name() string        { return "my-scenario" }
func (s *MyScenario) Description() string { return "My scenario description" }

func (s *MyScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
    // Setup（マスターデータ）は pool 経由でリポジトリを直接操作
    // Business Flow は deps 経由で UseCase を呼ぶ
    return nil
}
```

3. `src/cmd/tms/cmd/scenario.go` の `scenarioRegistry` に追加する

```go
var scenarioRegistry = []scenarios.Scenario{
    &scenarios.SourcingBidScenario{},
    &scenarios.MyScenario{}, // 追加
}
```

### 設計方針

- **Setup**（マスターデータseeding）: `pool` を使ってリポジトリを直接操作する。拠点・業者・ルートなどのマスターデータはインフラ関心事として扱う。
- **Business Flow**: `deps`（`ScenarioDeps`）経由で UseCase を呼ぶ。ドメインロジックを直接呼び出さない。
- 新しい UseCase が必要な場合は `deps.go` の `ScenarioDeps` に追加し、`scenario.go` の `scenarioRunCmd` で `deps` からフィールドを渡す。
