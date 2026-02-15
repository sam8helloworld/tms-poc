package rate

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
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

	// 4. 新TariffIDが契約内に存在することを確認
	if err := uc.validateTariffExistsInContract(ctx, c, input.NewTariffID); err != nil {
		return nil, err
	}

	// 5. 差し替え前のTariffIDを記録
	var oldTariffID uuid.UUID
	for _, entry := range r.Entries() {
		if entry.ID == input.EntryID {
			oldTariffID = entry.TariffID
			break
		}
	}

	// 6. エントリのTariffIDを差し替え
	if err := r.ReplaceEntryTariff(input.EntryID, input.NewTariffID); err != nil {
		return nil, NewUpdateRateEntryTariffError(
			"REPLACE_ERROR",
			"failed to replace entry tariff",
		).
			WithDetail("entryID", input.EntryID).
			WithDetail("rateID", input.RateID)
	}

	// 7. 永続化
	if err := uc.rateRepo.Save(ctx, r); err != nil {
		return nil, NewUpdateRateEntryTariffError("SAVE_ERROR", "failed to save rate").
			WithDetail("rateID", r.ID)
	}

	// 8. 出力DTOの作成
	return &UpdateRateEntryTariffOutput{
		RateID:          r.ID,
		RateStatus:      string(r.Status()),
		EntryID:         input.EntryID,
		OldTariffID:     oldTariffID,
		NewTariffID:     input.NewTariffID,
		TotalEntryCount: len(r.Entries()),
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
	return nil
}

// validateTariffExistsInContract: 新TariffIDが契約内に存在することを確認
func (uc *UpdateRateEntryTariffUseCase) validateTariffExistsInContract(
	ctx context.Context,
	c *contract.ServiceContract,
	tariffID uuid.UUID,
) error {
	tariff, err := uc.tariffRepo.FindByID(ctx, tariffID)
	if err != nil {
		return NewUpdateRateEntryTariffError(
			"TARIFF_NOT_FOUND",
			"new tariff not found",
		).
			WithDetail("tariffID", tariffID)
	}
	if tariff.ContractID != c.ID {
		return NewUpdateRateEntryTariffError(
			"TARIFF_NOT_IN_CONTRACT",
			"new tariff does not belong to the specified contract",
		).
			WithDetail("tariffID", tariffID).
			WithDetail("contractID", c.ID)
	}
	return nil
}
