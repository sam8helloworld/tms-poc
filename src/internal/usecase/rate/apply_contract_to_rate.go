package rate

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
	domainrate "github.com/sam8helloworld/tms-poc/internal/domain/rate"
)

// ApplyContractToRateUseCase: 契約反映ユースケース
// CONTRACTED状態の契約から料金表を選択し、DRAFT状態のレートにRateEntryとして追加する
//
// 処理の流れ:
// 1. レートを取得
// 2. 契約を取得し、CONTRACTED状態であることを確認
// 3. 対象の料金表を特定（TariffIDs指定時はそれらのみ、未指定時は契約内の全Tariff）
// 4. 各料金表に対してRateEntryを作成しレートに追加
// 5. レートを永続化
type ApplyContractToRateUseCase struct {
	// 本来利用すべきリポジトリ（コメントアウト）
	// rateRepo     domainrate.RateRepository
	// contractRepo commercial.ServiceContractRepository
}

// NewApplyContractToRateUseCase: ApplyContractToRateUseCaseのコンストラクタ
func NewApplyContractToRateUseCase(
// rateRepo domainrate.RateRepository,
// contractRepo commercial.ServiceContractRepository,
) *ApplyContractToRateUseCase {
	return &ApplyContractToRateUseCase{
		// rateRepo:     rateRepo,
		// contractRepo: contractRepo,
	}
}

// Execute: ユースケースを実行
func (uc *ApplyContractToRateUseCase) Execute(
	ctx context.Context,
	input ApplyContractToRateInput,
) (*ApplyContractToRateOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, err
	}

	// 2. レートを取得
	rate, err := uc.getRate(ctx, input.RateID)
	if err != nil {
		return nil, err
	}

	// 3. 契約を取得し、CONTRACTED状態であることを確認
	contract, err := uc.getContractedContract(ctx, input.ContractID)
	if err != nil {
		return nil, err
	}

	// 4. 対象の料金表を特定
	targetTariffs, err := uc.resolveTargetTariffs(contract, input.TariffIDs)
	if err != nil {
		return nil, err
	}

	// 5. 各料金表に対してRateEntryを作成しレートに追加
	addedEntries, err := uc.addEntriesToRate(rate, contract, targetTariffs, input.RouteScope)
	if err != nil {
		return nil, err
	}

	// 6. 永続化（コメントアウト）
	// if err := uc.rateRepo.Save(ctx, rate); err != nil {
	// 	return nil, NewApplyContractToRateError("SAVE_ERROR", "failed to save rate").
	// 		WithDetail("rateID", rate.ID)
	// }

	// 7. 出力DTOの作成
	return &ApplyContractToRateOutput{
		RateID:          rate.ID,
		RateStatus:      string(rate.Status()),
		ContractID:      contract.ID,
		ProviderID:      contract.ProviderID,
		AddedEntries:    addedEntries,
		TotalEntryCount: len(rate.Entries()),
	}, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *ApplyContractToRateUseCase) validateInput(input ApplyContractToRateInput) error {
	if input.RateID == uuid.Nil {
		return NewApplyContractToRateError("INVALID_INPUT", "rateID is required")
	}
	if input.ContractID == uuid.Nil {
		return NewApplyContractToRateError("INVALID_INPUT", "contractID is required")
	}
	return nil
}

// getRate: レートを取得
func (uc *ApplyContractToRateUseCase) getRate(
	ctx context.Context,
	rateID uuid.UUID,
) (*domainrate.Rate, error) {
	// rate, err := uc.rateRepo.FindByID(ctx, rateID)
	// if err != nil {
	// 	return nil, NewApplyContractToRateError("RATE_NOT_FOUND", "rate not found").
	// 		WithDetail("rateID", rateID)
	// }
	// return rate, nil

	// コメントアウト中のため、ダミーのDRAFTレートを返す
	_ = ctx
	dummyRate, _ := domainrate.NewRate(
		uuid.New(), // shipperID
		"dummy rate",
		defaultTime(),
		defaultTime().AddDate(1, 0, 0),
	)
	dummyRate.ID = rateID
	return dummyRate, nil
}

// getContractedContract: CONTRACTED状態の契約を取得
func (uc *ApplyContractToRateUseCase) getContractedContract(
	ctx context.Context,
	contractID uuid.UUID,
) (*commercial.ServiceContract, error) {
	// contract, err := uc.contractRepo.FindByID(ctx, contractID)
	// if err != nil {
	// 	return nil, NewApplyContractToRateError("CONTRACT_NOT_FOUND", "contract not found").
	// 		WithDetail("contractID", contractID)
	// }

	// コメントアウト中のため、ダミー契約を返す
	_ = ctx
	contract, _ := commercial.NewServiceContract(
		uuid.New(), // providerID
		uuid.New(), // shipperID
		defaultTime(),
		defaultTime().AddDate(1, 0, 0),
	)
	contract.ID = contractID

	// CONTRACTED状態チェック
	if !contract.IsActive() {
		return nil, NewApplyContractToRateError(
			"CONTRACT_NOT_CONTRACTED",
			"only CONTRACTED contracts can be applied to a rate",
		).
			WithDetail("contractID", contractID).
			WithDetail("status", string(contract.Status()))
	}

	return contract, nil
}

// resolveTargetTariffs: 対象の料金表を特定
// TariffIDsが空の場合は契約内の全Tariffを対象とする
func (uc *ApplyContractToRateUseCase) resolveTargetTariffs(
	contract *commercial.ServiceContract,
	tariffIDs []uuid.UUID,
) ([]*commercial.Tariff, error) {
	allTariffs := contract.Tariffs()

	if len(allTariffs) == 0 {
		return nil, NewApplyContractToRateError(
			"NO_TARIFFS",
			"contract has no tariffs",
		).WithDetail("contractID", contract.ID)
	}

	// TariffIDs未指定: 全Tariffを対象
	if len(tariffIDs) == 0 {
		return allTariffs, nil
	}

	// TariffIDs指定: 指定されたTariffのみ対象
	tariffMap := make(map[uuid.UUID]*commercial.Tariff, len(allTariffs))
	for _, t := range allTariffs {
		tariffMap[t.ID] = t
	}

	result := make([]*commercial.Tariff, 0, len(tariffIDs))
	for _, id := range tariffIDs {
		tariff, ok := tariffMap[id]
		if !ok {
			return nil, NewApplyContractToRateError(
				"TARIFF_NOT_FOUND",
				fmt.Sprintf("tariff %s not found in contract %s", id, contract.ID),
			).
				WithDetail("tariffID", id).
				WithDetail("contractID", contract.ID)
		}
		result = append(result, tariff)
	}

	return result, nil
}

// addEntriesToRate: 料金表ごとにRateEntryを作成しレートに追加
func (uc *ApplyContractToRateUseCase) addEntriesToRate(
	r *domainrate.Rate,
	contract *commercial.ServiceContract,
	tariffs []*commercial.Tariff,
	scopeInput RouteScopeInput,
) ([]AddedEntryDetail, error) {
	routeScope := domainrate.RouteScope{
		OriginID:      scopeInput.OriginID,
		DestinationID: scopeInput.DestinationID,
		TransportMode: scopeInput.TransportMode,
	}

	addedEntries := make([]AddedEntryDetail, 0, len(tariffs))

	for _, tariff := range tariffs {
		entry := &domainrate.RateEntry{
			ProviderID: contract.ProviderID,
			ContractID: contract.ID,
			TariffID:   tariff.ID,
			RouteScope: routeScope,
		}

		if err := r.AddEntry(entry); err != nil {
			return nil, NewApplyContractToRateError(
				"ADD_ENTRY_ERROR",
				fmt.Sprintf("failed to add entry for tariff %s: %s", tariff.ID, err.Error()),
			).
				WithDetail("tariffID", tariff.ID).
				WithDetail("rateID", r.ID)
		}

		addedEntries = append(addedEntries, AddedEntryDetail{
			EntryID:    entry.ID,
			TariffID:   tariff.ID,
			TariffName: tariff.Name,
		})
	}

	return addedEntries, nil
}

// defaultTime: ダミーデータ用のデフォルト時刻
func defaultTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
