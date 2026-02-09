package bid

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateBidContractUseCase_Execute(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateBidContractInput
		wantErr bool
		errCode string
	}{
		{
			name: "正常系: 入札契約を作成できる",
			input: CreateBidContractInput{
				BidRequestID:   uuid.New(),
				ProviderID:     uuid.New(),
				ShipperID:      uuid.New(),
				ValidFrom:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				ValidTo:        time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC),
				BidRequestName: "2026年度 北米航路FCL入札",
				RequestedBy:    uuid.New(),
				RequestedAt:    time.Now(),
				DueDate:        time.Date(2026, 3, 15, 23, 59, 59, 0, time.UTC),
				TargetRoutes: []BidRouteInfo{
					{
						OriginID:      uuid.New(),
						DestinationID: uuid.New(),
						TransportMode: "OCEAN",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: BidRequestIDが空",
			input: CreateBidContractInput{
				BidRequestID: uuid.Nil,
				ProviderID:   uuid.New(),
				ShipperID:    uuid.New(),
				ValidFrom:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				ValidTo:      time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC),
			},
			wantErr: true,
			errCode: "INVALID_INPUT",
		},
		{
			name: "異常系: ProviderIDが空",
			input: CreateBidContractInput{
				BidRequestID: uuid.New(),
				ProviderID:   uuid.Nil,
				ShipperID:    uuid.New(),
				ValidFrom:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				ValidTo:      time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC),
			},
			wantErr: true,
			errCode: "INVALID_INPUT",
		},
		{
			name: "異常系: ShipperIDが空",
			input: CreateBidContractInput{
				BidRequestID: uuid.New(),
				ProviderID:   uuid.New(),
				ShipperID:    uuid.Nil,
				ValidFrom:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				ValidTo:      time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC),
			},
			wantErr: true,
			errCode: "INVALID_INPUT",
		},
		{
			name: "異常系: 有効期間が逆転している",
			input: CreateBidContractInput{
				BidRequestID: uuid.New(),
				ProviderID:   uuid.New(),
				ShipperID:    uuid.New(),
				ValidFrom:    time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC),
				ValidTo:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			errCode: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			uc := NewCreateBidContractUseCase()
			ctx := context.Background()

			// Act
			output, err := uc.Execute(ctx, tt.input)

			// Assert
			if tt.wantErr {
				if err == nil {
					t.Errorf("期待されるエラーが発生しませんでした")
					return
				}
				var bidErr *CreateBidContractError
				if !errors.As(err, &bidErr) {
					t.Errorf("エラー型が *CreateBidContractError ではありません: %T", err)
					return
				}
				if bidErr.Code != tt.errCode {
					t.Errorf("エラーコードが一致しません: got %s, want %s", bidErr.Code, tt.errCode)
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラーが発生しました: %v", err)
					return
				}
				if output == nil {
					t.Error("出力がnilです")
					return
				}

				// 出力DTOの検証
				if output.ContractID == uuid.Nil {
					t.Error("ContractIDが空です")
				}
				if output.ProviderID != tt.input.ProviderID {
					t.Errorf("ProviderIDが一致しません: got %v, want %v", output.ProviderID, tt.input.ProviderID)
				}
				if output.ShipperID != tt.input.ShipperID {
					t.Errorf("ShipperIDが一致しません: got %v, want %v", output.ShipperID, tt.input.ShipperID)
				}
				if output.Status != "DRAFT" {
					t.Errorf("Statusが一致しません: got %s, want DRAFT", output.Status)
				}
				if !output.ValidFrom.Equal(tt.input.ValidFrom) {
					t.Errorf("ValidFromが一致しません: got %v, want %v", output.ValidFrom, tt.input.ValidFrom)
				}
				if !output.ValidTo.Equal(tt.input.ValidTo) {
					t.Errorf("ValidToが一致しません: got %v, want %v", output.ValidTo, tt.input.ValidTo)
				}
				if output.BidRequestID != tt.input.BidRequestID {
					t.Errorf("BidRequestIDが一致しません: got %v, want %v", output.BidRequestID, tt.input.BidRequestID)
				}
				if output.BidRequestName != tt.input.BidRequestName {
					t.Errorf("BidRequestNameが一致しません: got %s, want %s", output.BidRequestName, tt.input.BidRequestName)
				}
				if len(output.TargetRouteInfo) != len(tt.input.TargetRoutes) {
					t.Errorf("TargetRouteInfoの長さが一致しません: got %d, want %d", len(output.TargetRouteInfo), len(tt.input.TargetRoutes))
				}
				if len(output.NextSteps) == 0 {
					t.Error("NextStepsが空です")
				}
				if output.CreatedAt.IsZero() {
					t.Error("CreatedAtが設定されていません")
				}
			}
		})
	}
}

func TestCreateBidContractUseCase_MultipleProviders(t *testing.T) {
	// 複数の業者に対して入札契約を作成するシナリオ
	t.Run("複数業者への入札契約作成", func(t *testing.T) {
		// Arrange
		uc := NewCreateBidContractUseCase()
		ctx := context.Background()

		bidRequestID := uuid.New()
		shipperID := uuid.New()
		validFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		validTo := time.Date(2027, 3, 31, 23, 59, 59, 0, time.UTC)

		providers := []uuid.UUID{
			uuid.New(), // Provider A
			uuid.New(), // Provider B
			uuid.New(), // Provider C
		}

		targetRoutes := []BidRouteInfo{
			{
				OriginID:      uuid.New(),
				DestinationID: uuid.New(),
				TransportMode: "OCEAN",
			},
		}

		// Act & Assert: 各業者に対して契約を作成
		var contractIDs []uuid.UUID
		for i, providerID := range providers {
			input := CreateBidContractInput{
				BidRequestID:   bidRequestID,
				ProviderID:     providerID,
				ShipperID:      shipperID,
				ValidFrom:      validFrom,
				ValidTo:        validTo,
				BidRequestName: "2026年度 北米航路FCL入札",
				TargetRoutes:   targetRoutes,
			}

			output, err := uc.Execute(ctx, input)
			if err != nil {
				t.Errorf("Provider %d の契約作成に失敗: %v", i+1, err)
				return
			}
			if output == nil {
				t.Errorf("Provider %d の出力がnilです", i+1)
				return
			}

			contractIDs = append(contractIDs, output.ContractID)

			// すべての契約が同じBidRequestIDを持つ
			if output.BidRequestID != bidRequestID {
				t.Errorf("BidRequestIDが一致しません: got %v, want %v", output.BidRequestID, bidRequestID)
			}
			if output.Status != "DRAFT" {
				t.Errorf("Statusが一致しません: got %s, want DRAFT", output.Status)
			}
		}

		// すべての契約IDがユニークであることを確認
		if len(contractIDs) != len(providers) {
			t.Errorf("契約IDの数が一致しません: got %d, want %d", len(contractIDs), len(providers))
		}
		uniqueIDs := make(map[uuid.UUID]bool)
		for _, id := range contractIDs {
			uniqueIDs[id] = true
		}
		if len(uniqueIDs) != len(providers) {
			t.Errorf("契約IDが重複しています: unique %d, total %d", len(uniqueIDs), len(providers))
		}
	})
}
