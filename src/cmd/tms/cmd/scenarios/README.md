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
