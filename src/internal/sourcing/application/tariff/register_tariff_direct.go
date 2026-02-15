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

// RegisterTariffDirectUseCase: 構造化データから直接料金表を登録するユースケース
// ファイルパーサーを経由せず、API/統合から構造化データを受け取って料金表を登録する
type RegisterTariffDirectUseCase struct {
	tariffRepo   pricing.TariffRepository
	contractRepo contract.ServiceContractRepository
}

// NewRegisterTariffDirectUseCase: RegisterTariffDirectUseCaseのコンストラクタ
func NewRegisterTariffDirectUseCase(
	tariffRepo pricing.TariffRepository,
	contractRepo contract.ServiceContractRepository,
) *RegisterTariffDirectUseCase {
	return &RegisterTariffDirectUseCase{
		tariffRepo:   tariffRepo,
		contractRepo: contractRepo,
	}
}

// RegisterTariffDirectInput: 構造化データによる料金表登録の入力DTO
type RegisterTariffDirectInput struct {
	ContractID    uuid.UUID
	TariffName    string
	EffectiveFrom time.Time
	EffectiveTo   time.Time
	LineItems     []LineItemInput
}

// LineItemInput: 料金明細の入力DTO
type LineItemInput struct {
	ChargeCode   string
	Category     string
	ScopeType    pricing.ServiceScopeType
	ScopeAttrs   map[string]string
	PricingType  pricing.PricingStrategyType
	PricingAttrs map[string]any
}

// RegisterTariffDirectOutput: 構造化データによる料金表登録の出力DTO
type RegisterTariffDirectOutput struct {
	ContractID       uuid.UUID
	ContractStatus   string
	TariffID         uuid.UUID
	TariffName       string
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	LineItemCount    int
	TotalTariffCount int
}

// Execute: ユースケースを実行
func (uc *RegisterTariffDirectUseCase) Execute(
	ctx context.Context,
	input RegisterTariffDirectInput,
) (*RegisterTariffDirectOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewRegisterTariffError("INVALID_INPUT", err.Error())
	}

	// 2. 契約の取得とDRAFT状態チェック
	c, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}
	if !c.IsDraft() {
		return nil, NewRegisterTariffError("CONTRACT_NOT_DRAFT", "contract is not in DRAFT status").
			WithDetail("contractID", input.ContractID).
			WithDetail("status", string(c.Status()))
	}

	// 3. Tariffドメインモデルの生成
	tariff, err := pricing.NewTariff(
		input.TariffName,
		input.ContractID,
		input.EffectiveFrom,
		input.EffectiveTo,
	)
	if err != nil {
		return nil, NewRegisterTariffError("CREATE_ERROR", err.Error())
	}

	// 4. LineItemの変換・追加
	converter := &RegisterTariffUseCase{} // convertToLineItemメソッド再利用
	for i, li := range input.LineItems {
		parsed := pricing.ParsedLineItem{
			ChargeCode:       li.ChargeCode,
			Category:         li.Category,
			ServiceScopeType: li.ScopeType,
			ServiceScopeAttrs: li.ScopeAttrs,
			PricingType:      li.PricingType,
			PricingAttrs:     li.PricingAttrs,
		}
		lineItem, err := converter.convertToLineItem(parsed)
		if err != nil {
			return nil, NewRegisterTariffError("CONVERSION_ERROR",
				fmt.Sprintf("failed to convert line item at index %d: %s", i, err.Error()))
		}
		if err := tariff.AddLineItem(*lineItem); err != nil {
			return nil, NewRegisterTariffError("ADD_ITEM_ERROR",
				fmt.Sprintf("failed to add line item at index %d: %s", i, err.Error()))
		}
	}

	// 5. 永続化
	if err := uc.tariffRepo.Save(ctx, tariff); err != nil {
		return nil, NewRegisterTariffError("SAVE_ERROR", err.Error()).
			WithDetail("tariffID", tariff.ID)
	}

	// 6. Tariff件数取得
	totalCount, _ := uc.tariffRepo.CountByContractID(ctx, input.ContractID)

	return &RegisterTariffDirectOutput{
		ContractID:       c.ID,
		ContractStatus:   string(c.Status()),
		TariffID:         tariff.ID,
		TariffName:       tariff.Name,
		EffectiveFrom:    tariff.EffectiveDate.From,
		EffectiveTo:      tariff.EffectiveDate.To,
		LineItemCount:    len(tariff.LineItems),
		TotalTariffCount: totalCount,
	}, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *RegisterTariffDirectUseCase) validateInput(input RegisterTariffDirectInput) error {
	if input.ContractID == uuid.Nil {
		return errors.New("contract ID is required")
	}
	if input.TariffName == "" {
		return errors.New("tariff name is required")
	}
	if input.EffectiveFrom.After(input.EffectiveTo) {
		return errors.New("effective date range is invalid: from must be before or equal to to")
	}
	if len(input.LineItems) == 0 {
		return errors.New("at least one line item is required")
	}
	return nil
}
