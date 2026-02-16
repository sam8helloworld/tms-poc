package rate

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// UpdateRateEntryTariffUseCase: DRAFT状態のRateのエントリのTariffIDを新しいTariffIDに差し替えるユースケース
type UpdateRateEntryTariffUseCase struct {
	rateRepo     domainrate.RateRepository
	contractRepo contract.ServiceContractRepository
	tariffRepo   pricing.TariffRepository
}

// NewUpdateRateEntryTariffUseCase: コンストラクタ
func NewUpdateRateEntryTariffUseCase(
	rateRepo domainrate.RateRepository,
	contractRepo contract.ServiceContractRepository,
	tariffRepo pricing.TariffRepository,
) *UpdateRateEntryTariffUseCase {
	return &UpdateRateEntryTariffUseCase{
		rateRepo:     rateRepo,
		contractRepo: contractRepo,
		tariffRepo:   tariffRepo,
	}
}

// Execute: ユースケースを実行
func (uc *UpdateRateEntryTariffUseCase) Execute(
	ctx context.Context,
	input UpdateRateEntryTariffInput,
) (*UpdateRateEntryTariffOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, err
	}

	// 2. レートを取得
	r, err := uc.rateRepo.FindByID(ctx, input.RateID)
	if err != nil {
		return nil, NewUpdateRateEntryTariffError("RATE_NOT_FOUND", "rate not found").
			WithDetail("rateID", input.RateID)
	}

	// DRAFT状態の確認
	if r.Status() != domainrate.RateStatusDraft {
		return nil, NewUpdateRateEntryTariffError(
			"RATE_NOT_DRAFT",
			"entry tariffs can only be replaced in DRAFT rates",
		).
			WithDetail("rateID", input.RateID).
			WithDetail("status", string(r.Status()))
	}

	// 3. 契約を取得
	c, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewUpdateRateEntryTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// 4. 新Tariffを取得し契約内に存在することを確認
	newTariff, err := uc.getValidatedTariff(ctx, c, input.NewTariffID)
	if err != nil {
		return nil, err
	}

	// 5. ACL経由でLineItem情報を取得
	rateableItems := pricing.ExtractRateableItems(newTariff)
	var targetItem *pricing.RateableLineItem
	for i, item := range rateableItems {
		if item.LineItemID == input.NewTariffLineItemID {
			targetItem = &rateableItems[i]
			break
		}
	}
	if targetItem == nil {
		return nil, NewUpdateRateEntryTariffError(
			"LINE_ITEM_NOT_FOUND",
			fmt.Sprintf("line item %s not found in tariff %s", input.NewTariffLineItemID, input.NewTariffID),
		).
			WithDetail("tariffID", input.NewTariffID).
			WithDetail("lineItemID", input.NewTariffLineItemID)
	}

	// 6. 差し替え前の情報を記録
	var oldTariffID, oldLineItemID uuid.UUID
	for _, entry := range r.Entries() {
		if entry.ID == input.EntryID {
			oldTariffID = entry.TariffID
			oldLineItemID = entry.TariffLineItemID
			break
		}
	}

	// 7. RouteScopeの構築
	var routeScope domainrate.RouteScope
	if targetItem.OriginID != nil {
		lid := route.LocationID(*targetItem.OriginID)
		routeScope.OriginID = &lid
	}
	if targetItem.DestinationID != nil {
		lid := route.LocationID(*targetItem.DestinationID)
		routeScope.DestinationID = &lid
	}
	if targetItem.TransportMode != nil {
		m := shared.TransportMode(*targetItem.TransportMode)
		routeScope.TransportMode = &m
	}

	unitPrice := shared.ZeroMoney("USD")
	if targetItem.UnitPrice != nil {
		unitPrice = *targetItem.UnitPrice
	}

	// 8. エントリのLineItem情報を差し替え
	replacement := domainrate.RateEntryReplacement{
		TariffID:         input.NewTariffID,
		TariffLineItemID: input.NewTariffLineItemID,
		RouteScope:       routeScope,
		ChargeCode:       targetItem.ChargeCode,
		Category:         targetItem.Category,
		UnitPrice:        unitPrice,
	}
	if err := r.ReplaceEntryLineItem(input.EntryID, replacement); err != nil {
		return nil, NewUpdateRateEntryTariffError(
			"REPLACE_ERROR",
			"failed to replace entry tariff",
		).
			WithDetail("entryID", input.EntryID).
			WithDetail("rateID", input.RateID)
	}

	// 9. 永続化
	if err := uc.rateRepo.Save(ctx, r); err != nil {
		return nil, NewUpdateRateEntryTariffError("SAVE_ERROR", "failed to save rate").
			WithDetail("rateID", r.ID)
	}

	// 10. 出力DTOの作成
	return &UpdateRateEntryTariffOutput{
		RateID:              r.ID,
		RateStatus:          string(r.Status()),
		EntryID:             input.EntryID,
		OldTariffID:         oldTariffID,
		NewTariffID:         input.NewTariffID,
		OldTariffLineItemID: oldLineItemID,
		NewTariffLineItemID: input.NewTariffLineItemID,
		TotalEntryCount:     len(r.Entries()),
	}, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *UpdateRateEntryTariffUseCase) validateInput(input UpdateRateEntryTariffInput) error {
	if input.RateID == uuid.Nil {
		return NewUpdateRateEntryTariffError("INVALID_INPUT", "rateID is required")
	}
	if input.EntryID == uuid.Nil {
		return NewUpdateRateEntryTariffError("INVALID_INPUT", "entryID is required")
	}
	if input.ContractID == uuid.Nil {
		return NewUpdateRateEntryTariffError("INVALID_INPUT", "contractID is required")
	}
	if input.NewTariffID == uuid.Nil {
		return NewUpdateRateEntryTariffError("INVALID_INPUT", "newTariffID is required")
	}
	if input.NewTariffLineItemID == uuid.Nil {
		return NewUpdateRateEntryTariffError("INVALID_INPUT", "newTariffLineItemID is required")
	}
	return nil
}

// getValidatedTariff: 新TariffIDが契約内に存在することを確認しTariffを返す
func (uc *UpdateRateEntryTariffUseCase) getValidatedTariff(
	ctx context.Context,
	c *contract.ServiceContract,
	tariffID uuid.UUID,
) (*pricing.Tariff, error) {
	tariff, err := uc.tariffRepo.FindByID(ctx, tariffID)
	if err != nil {
		return nil, NewUpdateRateEntryTariffError(
			"TARIFF_NOT_FOUND",
			"new tariff not found",
		).
			WithDetail("tariffID", tariffID)
	}
	if tariff.ContractID != c.ID {
		return nil, NewUpdateRateEntryTariffError(
			"TARIFF_NOT_IN_CONTRACT",
			"new tariff does not belong to the specified contract",
		).
			WithDetail("tariffID", tariffID).
			WithDetail("contractID", c.ID)
	}
	return tariff, nil
}
