package rate

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/contract"
	domainrate "github.com/sam8helloworld/tms-poc/internal/domain/rate"
)

// UpdateRateEntryTariffUseCase: DRAFT状態のRateのエントリのTariffIDを新しいTariffIDに差し替えるユースケース
// 契約アメンドメントで新バージョンのTariffが追加された後、レートのエントリを更新する
//
// 処理の流れ:
// 1. 入力バリデーション
// 2. レートを取得し、DRAFT状態であることを確認
// 3. 契約を取得し、新TariffIDが契約内に存在することを確認
// 4. rate.ReplaceEntryTariff(entryID, newTariffID) でエントリを更新
// 5. 永続化
// 6. 出力DTOを返却
type UpdateRateEntryTariffUseCase struct {
	// 本来利用すべきリポジトリ（コメントアウト）
	// rateRepo     domainrate.RateRepository
	// contractRepo contract.ServiceContractRepository
	// tariffRepo   pricing.TariffRepository
}

// NewUpdateRateEntryTariffUseCase: コンストラクタ
func NewUpdateRateEntryTariffUseCase(
// rateRepo domainrate.RateRepository,
// contractRepo contract.ServiceContractRepository,
// tariffRepo pricing.TariffRepository,
) *UpdateRateEntryTariffUseCase {
	return &UpdateRateEntryTariffUseCase{
		// rateRepo:     rateRepo,
		// contractRepo: contractRepo,
		// tariffRepo:   tariffRepo,
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
	rate, err := uc.getRate(ctx, input.RateID)
	if err != nil {
		return nil, err
	}

	// DRAFT状態の確認
	if rate.Status() != domainrate.RateStatusDraft {
		return nil, NewUpdateRateEntryTariffError(
			"RATE_NOT_DRAFT",
			"entry tariffs can only be replaced in DRAFT rates",
		).
			WithDetail("rateID", input.RateID).
			WithDetail("status", string(rate.Status()))
	}

	// 3. 契約を取得し、新TariffIDが存在することを確認
	contract, err := uc.getContract(ctx, input.ContractID)
	if err != nil {
		return nil, err
	}

	if err := uc.validateTariffExistsInContract(contract, input.NewTariffID); err != nil {
		return nil, err
	}

	// 4. 差し替え前のTariffIDを記録
	var oldTariffID uuid.UUID
	for _, entry := range rate.Entries() {
		if entry.ID == input.EntryID {
			oldTariffID = entry.TariffID
			break
		}
	}

	// 5. エントリのTariffIDを差し替え
	if err := rate.ReplaceEntryTariff(input.EntryID, input.NewTariffID); err != nil {
		return nil, NewUpdateRateEntryTariffError(
			"REPLACE_ERROR",
			"failed to replace entry tariff",
		).
			WithDetail("entryID", input.EntryID).
			WithDetail("rateID", input.RateID)
	}

	// 6. 永続化（コメントアウト）
	// if err := uc.rateRepo.Save(ctx, rate); err != nil {
	// 	return nil, NewUpdateRateEntryTariffError("SAVE_ERROR", "failed to save rate").
	// 		WithDetail("rateID", rate.ID)
	// }

	// 7. 出力DTOの作成
	return &UpdateRateEntryTariffOutput{
		RateID:          rate.ID,
		RateStatus:      string(rate.Status()),
		EntryID:         input.EntryID,
		OldTariffID:     oldTariffID,
		NewTariffID:     input.NewTariffID,
		TotalEntryCount: len(rate.Entries()),
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

// getRate: レートを取得
func (uc *UpdateRateEntryTariffUseCase) getRate(
	ctx context.Context,
	rateID uuid.UUID,
) (*domainrate.Rate, error) {
	// rate, err := uc.rateRepo.FindByID(ctx, rateID)
	// if err != nil {
	// 	return nil, NewUpdateRateEntryTariffError("RATE_NOT_FOUND", "rate not found").
	// 		WithDetail("rateID", rateID)
	// }
	// return rate, nil

	// コメントアウト中のため、ダミーのDRAFTレートを返す
	_ = ctx
	dummyRate, _ := domainrate.NewRate(
		uuid.New(),
		"dummy rate",
		defaultTime(),
		defaultTime().AddDate(1, 0, 0),
	)
	dummyRate.ID = rateID
	return dummyRate, nil
}

// getContract: 契約を取得
func (uc *UpdateRateEntryTariffUseCase) getContract(
	ctx context.Context,
	contractID uuid.UUID,
) (*contract.ServiceContract, error) {
	// contract, err := uc.contractRepo.FindByID(ctx, contractID)
	// if err != nil {
	// 	return nil, NewUpdateRateEntryTariffError("CONTRACT_NOT_FOUND", "contract not found").
	// 		WithDetail("contractID", contractID)
	// }
	// return contract, nil

	// コメントアウト中のため、ダミー契約を返す
	_ = ctx
	dummyContract, _ := contract.NewServiceContract(
		uuid.New(),
		uuid.New(),
		defaultTime(),
		defaultTime().AddDate(1, 0, 0),
	)
	dummyContract.ID = contractID
	return dummyContract, nil
}

// validateTariffExistsInContract: 新TariffIDが契約内に存在することを確認
// 注: 本来はtariffRepoを使用する（コメントアウト）
func (uc *UpdateRateEntryTariffUseCase) validateTariffExistsInContract(
	contract *contract.ServiceContract,
	tariffID uuid.UUID,
) error {
	// 本来の実装:
	// tariff, err := uc.tariffRepo.FindByID(ctx, tariffID)
	// if err != nil || tariff == nil {
	// 	return NewUpdateRateEntryTariffError(
	// 		"TARIFF_NOT_FOUND",
	// 		"new tariff not found",
	// 	).
	// 		WithDetail("tariffID", tariffID)
	// }
	// if tariff.ContractID != contract.ID {
	// 	return NewUpdateRateEntryTariffError(
	// 		"TARIFF_NOT_IN_CONTRACT",
	// 		"new tariff does not belong to the specified contract",
	// 	).
	// 		WithDetail("tariffID", tariffID).
	// 		WithDetail("contractID", contract.ID)
	// }
	// return nil

	// コメントアウト中のためダミーチェック（常にnilを返す）
	_ = contract
	_ = tariffID
	return nil
}
