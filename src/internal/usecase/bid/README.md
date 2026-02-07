# Bid Use Cases

入札プロセスに関連するユースケースを提供します。

## 概要

物流業界における入札プロセスでは、荷主企業（Shipper）が複数の物流業者（Logistics Provider）に対して料金の見積もりを依頼し、最適な業者を選定します。このパッケージは、そのような入札プロセスをサポートするユースケースを提供します。

## 入札プロセスのフロー

```
1. 荷主が入札を開始（BidRequest作成）
   ↓
2. 複数の物流業者を選択
   ↓
3. CreateBidContractUseCase: 各業者に対してDRAFT契約を作成
   ↓
4. 各業者が料金表を提出
   ↓
5. RegisterTariffUseCase: 料金表をDRAFT契約に登録
   ↓
6. 荷主が各DRAFT契約の料金表を比較検討
   ↓
7. 最適な契約を選択し、MarkAsContractedで正式契約化
   ↓
8. 他の契約はMarkAsCancelledでキャンセル
```

## CreateBidContractUseCase

### 目的

入札プロセスにおいて、各物流業者との契約をDRAFT状態で作成します。

### 使用例

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/usecase/bid"
)

func main() {
	// ユースケースの初期化
	// 注: 実際の使用時はリポジトリを注入します
	uc := bid.NewCreateBidContractUseCase()

	// 入札要求の情報
	bidRequestID := uuid.New()
	shipperID := uuid.New() // 荷主企業ID

	// 入札対象の業者リスト
	providerIDs := []uuid.UUID{
		uuid.New(), // Maersk
		uuid.New(), // ONE
		uuid.New(), // MSC
	}

	// 契約有効期間
	validFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC)

	// 入札対象のルート
	targetRoutes := []bid.BidRouteInfo{
		{
			OriginID:      uuid.New(), // Tokyo Port
			DestinationID: uuid.New(), // Los Angeles Port
			TransportMode: "OCEAN",
		},
	}

	// 各業者に対して契約を作成
	ctx := context.Background()
	for i, providerID := range providerIDs {
		input := bid.CreateBidContractInput{
			BidRequestID:   bidRequestID,
			ProviderID:     providerID,
			ShipperID:      shipperID,
			ValidFrom:      validFrom,
			ValidTo:        validTo,
			BidRequestName: "2026年度 北米航路FCL入札",
			RequestedBy:    uuid.New(),
			RequestedAt:    time.Now(),
			DueDate:        time.Date(2026, 3, 15, 23, 59, 59, 0, time.UTC),
			TargetRoutes:   targetRoutes,
		}

		output, err := uc.Execute(ctx, input)
		if err != nil {
			fmt.Printf("Provider %d の契約作成に失敗: %v\n", i+1, err)
			continue
		}

		fmt.Printf("Provider %d の契約を作成しました:\n", i+1)
		fmt.Printf("  ContractID: %s\n", output.ContractID)
		fmt.Printf("  Status: %s\n", output.Status)
		fmt.Printf("  ValidFrom: %s\n", output.ValidFrom)
		fmt.Printf("  ValidTo: %s\n", output.ValidTo)
		fmt.Printf("  次のステップ:\n")
		for _, step := range output.NextSteps {
			fmt.Printf("    - %s\n", step)
		}
	}
}
```

### 入力パラメータ

| フィールド | 型 | 必須 | 説明 |
|----------|---|------|------|
| BidRequestID | uuid.UUID | ○ | 入札要求ID（複数の業者への入札をグループ化） |
| ProviderID | uuid.UUID | ○ | 物流企業ID（入札参加業者） |
| ShipperID | uuid.UUID | ○ | 荷主企業ID |
| ValidFrom | time.Time | ○ | 契約開始日 |
| ValidTo | time.Time | ○ | 契約終了日 |
| BidRequestName | string | - | 入札要求名（例: "2026年度 北米航路FCL入札"） |
| RequestedBy | uuid.UUID | - | 入札を開始したユーザーのID |
| RequestedAt | time.Time | - | 入札開始日時 |
| DueDate | time.Time | - | 入札締切日（業者からの回答期限） |
| TargetRoutes | []BidRouteInfo | - | 入札対象のルート情報 |

### 出力

| フィールド | 型 | 説明 |
|----------|---|------|
| ContractID | uuid.UUID | 作成された契約のID |
| ProviderID | uuid.UUID | 物流企業ID |
| ShipperID | uuid.UUID | 荷主企業ID |
| Status | string | 契約ステータス（常に "DRAFT"） |
| ValidFrom | time.Time | 契約開始日 |
| ValidTo | time.Time | 契約終了日 |
| CreatedAt | time.Time | 契約作成日時 |
| BidRequestID | uuid.UUID | 入札要求ID |
| BidRequestName | string | 入札要求名 |
| TargetRouteInfo | []BidRouteInfo | 入札対象のルート情報 |
| NextSteps | []string | 次に実行すべきアクション |

### エラー

| エラーコード | 説明 |
|------------|------|
| INVALID_INPUT | 入力パラメータが不正（必須項目が空、有効期間が逆転など） |
| PROVIDER_NOT_FOUND | 指定された物流業者が見つからない（リポジトリ実装時） |
| DUPLICATE_CONTRACT | 同じ入札要求に対する契約が既に存在する（リポジトリ実装時） |
| SAVE_ERROR | 契約の保存に失敗（リポジトリ実装時） |

## 関連ユースケース

- **RegisterTariffUseCase** (`internal/usecase/tariff`): DRAFT契約に料金表を登録
- **ApplyContractToRateUseCase** (`internal/usecase/rate`): 正式契約化した契約をRateに適用

## ドメインモデル

- **ServiceContract** (`internal/domain/commercial/contract.go`): 契約集約ルート
  - 初期状態: `DRAFT`
  - 正式契約化: `MarkAsContracted()`で`CONTRACTED`状態へ
  - キャンセル: `MarkAsCancelled()`で`CANCELLED`状態へ

## 注意事項

- 現在の実装では、リポジトリ層がコメントアウトされているため、データの永続化は行われません
- 実際の運用時は、以下のリポジトリを実装して注入する必要があります:
  - `ServiceContractRepository`: 契約の永続化
  - `LogisticsProviderRepository`: 物流業者の検証
