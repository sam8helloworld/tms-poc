package tariff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// AmendContractTariffDirectUseCase: 構造化データから直接料金表の改定版を追加するユースケース
// ファイルパーサーを経由せず、API/統合から構造化データを受け取ってTariff改定版を登録する
// CONTRACTED状態の契約に対してのみ実行可能
type AmendContractTariffDirectUseCase struct {
	tariffRepo   pricing.TariffRepository
	contractRepo contract.ServiceContractRepository
}

// NewAmendContractTariffDirectUseCase: コンストラクタ
func NewAmendContractTariffDirectUseCase(
	tariffRepo pricing.TariffRepository,
	contractRepo contract.ServiceContractRepository,
) *AmendContractTariffDirectUseCase {
	return &AmendContractTariffDirectUseCase{
		tariffRepo:   tariffRepo,
		contractRepo: contractRepo,
	}
}

// AmendContractTariffDirectInput: 構造化データによる料金表改定の入力DTO
type AmendContractTariffDirectInput struct {
	ContractID    uuid.UUID       // CONTRACTED状態の契約ID
	BaseTariffID  uuid.UUID       // 改定元のTariffID
	EffectiveFrom time.Time       // 新バージョンの有効期間開始
	EffectiveTo   time.Time       // 新バージョンの有効期間終了
	LineItems     []LineItemInput // 改定後のLineItem一覧
}

// AmendContractTariffDirectOutput: 構造化データによる料金表改定の出力DTO
type AmendContractTariffDirectOutput struct {
	ContractID       uuid.UUID
	ContractStatus   string
	TariffID         uuid.UUID
	TariffName       string
	TariffVersion    int
	BaseTariffID     uuid.UUID
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	LineItemCount    int
	LineItemIDs      []uuid.UUID // 登録された各LineItemのID
	TotalTariffCount int
}

// Execute: ユースケースを実行
func (uc *AmendContractTariffDirectUseCase) Execute(
	ctx context.Context,
	input AmendContractTariffDirectInput,
) (*AmendContractTariffDirectOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewAmendTariffError("INVALID_INPUT", err.Error())
	}

	// 2. 契約を取得し、CONTRACTED状態であることを確認
	c, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewAmendTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}
	if !c.IsActive() {
		return nil, NewAmendTariffError(
			"INVALID_CONTRACT_STATE",
			"tariff amendments can only be added to CONTRACTED contracts",
		).WithDetail("currentStatus", string(c.Status()))
	}

	// 3. ベースとなるTariffを取得
	baseTariff, err := uc.tariffRepo.FindByID(ctx, input.BaseTariffID)
	if err != nil || baseTariff == nil {
		return nil, NewAmendTariffError("BASE_TARIFF_NOT_FOUND", "base tariff not found").
			WithDetail("baseTariffID", input.BaseTariffID).
			WithDetail("contractID", input.ContractID)
	}
	if baseTariff.ContractID != input.ContractID {
		return nil, NewAmendTariffError("BASE_TARIFF_NOT_IN_CONTRACT", "base tariff does not belong to the specified contract").
			WithDetail("baseTariffID", input.BaseTariffID).
			WithDetail("contractID", input.ContractID)
	}

	// 4. 新バージョンのTariffを作成
	newTariff, err := pricing.NewTariffVersion(baseTariff, input.EffectiveFrom, input.EffectiveTo)
	if err != nil {
		return nil, NewAmendTariffError("TARIFF_CREATE_ERROR", "failed to create tariff version").
			WithCause(err)
	}

	// 5. LineItemの変換・追加
	converter := &RegisterTariffUseCase{} // convertToLineItemメソッド再利用
	for i, li := range input.LineItems {
		parsed := pricing.ParsedLineItem{
			ChargeCode:        li.ChargeCode,
			Category:          li.Category,
			ServiceScopeType:  li.ScopeType,
			ServiceScopeAttrs: li.ScopeAttrs,
			PricingType:       li.PricingType,
			PricingAttrs:      li.PricingAttrs,
			OperatorVendorID:  li.OperatorVendorID,
		}
		lineItem, err := converter.convertToLineItem(parsed)
		if err != nil {
			return nil, NewAmendTariffError("CONVERSION_ERROR",
				fmt.Sprintf("failed to convert line item at index %d: %s", i, err.Error()))
		}
		if err := newTariff.AddLineItem(*lineItem); err != nil {
			return nil, NewAmendTariffError("ADD_ITEM_ERROR",
				fmt.Sprintf("failed to add line item at index %d: %s", i, err.Error()))
		}
	}

	// 6. IsNewVersionチェック
	if !newTariff.IsNewVersion() {
		return nil, NewAmendTariffError("NOT_NEW_VERSION", "only new versions of existing tariffs can be added as amendments")
	}

	// 7. 永続化
	if err := uc.tariffRepo.Save(ctx, newTariff); err != nil {
		return nil, NewAmendTariffError("SAVE_ERROR", "failed to save tariff").
			WithCause(err)
	}

	// 8. Tariff件数を取得
	totalCount, _ := uc.tariffRepo.CountByContractID(ctx, input.ContractID)

	// 9. LineItemIDsの収集
	lineItemIDs := make([]uuid.UUID, len(newTariff.LineItems))
	for i, li := range newTariff.LineItems {
		lineItemIDs[i] = li.ID
	}

	return &AmendContractTariffDirectOutput{
		ContractID:       c.ID,
		ContractStatus:   string(c.Status()),
		TariffID:         newTariff.ID,
		TariffName:       newTariff.Name,
		TariffVersion:    newTariff.Version,
		BaseTariffID:     *newTariff.BaseVersionID,
		EffectiveFrom:    newTariff.EffectiveDate.From,
		EffectiveTo:      newTariff.EffectiveDate.To,
		LineItemCount:    len(newTariff.LineItems),
		LineItemIDs:      lineItemIDs,
		TotalTariffCount: totalCount,
	}, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *AmendContractTariffDirectUseCase) validateInput(input AmendContractTariffDirectInput) error {
	if input.ContractID == uuid.Nil {
		return errors.New("contract ID is required")
	}
	if input.BaseTariffID == uuid.Nil {
		return errors.New("base tariff ID is required")
	}
	if input.EffectiveFrom.After(input.EffectiveTo) {
		return errors.New("effective date range is invalid: from must be before or equal to to")
	}
	if len(input.LineItems) == 0 {
		return errors.New("at least one line item is required")
	}
	return nil
}
