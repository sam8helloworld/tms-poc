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

// ApplyContractToRateUseCase: 契約反映ユースケース
// CONTRACTED状態の契約から料金表を選択し、DRAFT状態のレートにRateEntryとして追加する
type ApplyContractToRateUseCase struct {
	rateRepo     domainrate.RateRepository
	contractRepo contract.ServiceContractRepository
	tariffRepo   pricing.TariffRepository
}

// NewApplyContractToRateUseCase: ApplyContractToRateUseCaseのコンストラクタ
func NewApplyContractToRateUseCase(
	rateRepo domainrate.RateRepository,
	contractRepo contract.ServiceContractRepository,
	tariffRepo pricing.TariffRepository,
) *ApplyContractToRateUseCase {
	return &ApplyContractToRateUseCase{
		rateRepo:     rateRepo,
		contractRepo: contractRepo,
		tariffRepo:   tariffRepo,
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
	r, err := uc.getRate(ctx, input.RateID)
	if err != nil {
		return nil, err
	}

	// 3. 契約を取得し、CONTRACTED状態であることを確認
	c, err := uc.getContractedContract(ctx, input.ContractID)
	if err != nil {
		return nil, err
	}

	// 4. 対象の料金表を特定
	targetTariffs, err := uc.resolveTargetTariffs(ctx, c, input.TariffIDs)
	if err != nil {
		return nil, err
	}

	// 5. 各料金表に対してRateEntryを作成しレートに追加
	addedEntries, err := uc.addEntriesToRate(r, c, targetTariffs, input.TariffLineItemIDs)
	if err != nil {
		return nil, err
	}

	// 6. 永続化
	if err := uc.rateRepo.Save(ctx, r); err != nil {
		return nil, NewApplyContractToRateError("SAVE_ERROR", "failed to save rate").
			WithDetail("rateID", r.ID)
	}

	// 7. 出力DTOの作成
	return &ApplyContractToRateOutput{
		RateID:          r.ID,
		RateStatus:      string(r.Status()),
		ContractID:      c.ID,
		ProviderID:      c.ProviderID,
		AddedEntries:    addedEntries,
		TotalEntryCount: len(r.Entries()),
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
	r, err := uc.rateRepo.FindByID(ctx, rateID)
	if err != nil {
		return nil, NewApplyContractToRateError("RATE_NOT_FOUND", "rate not found").
			WithDetail("rateID", rateID)
	}
	return r, nil
}

// getContractedContract: CONTRACTED状態の契約を取得
func (uc *ApplyContractToRateUseCase) getContractedContract(
	ctx context.Context,
	contractID uuid.UUID,
) (*contract.ServiceContract, error) {
	c, err := uc.contractRepo.FindByID(ctx, contractID)
	if err != nil {
		return nil, NewApplyContractToRateError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", contractID)
	}

	if !c.IsActive() {
		return nil, NewApplyContractToRateError(
			"CONTRACT_NOT_CONTRACTED",
			"only CONTRACTED contracts can be applied to a rate",
		).
			WithDetail("contractID", contractID).
			WithDetail("status", string(c.Status()))
	}

	return c, nil
}

// resolveTargetTariffs: 対象の料金表を特定
func (uc *ApplyContractToRateUseCase) resolveTargetTariffs(
	ctx context.Context,
	c *contract.ServiceContract,
	tariffIDs []uuid.UUID,
) ([]*pricing.Tariff, error) {
	// TariffIDs未指定: 契約内の全Tariffを対象
	if len(tariffIDs) == 0 {
		allTariffs, err := uc.tariffRepo.FindByContractID(ctx, c.ID)
		if err != nil {
			return nil, NewApplyContractToRateError("TARIFF_FETCH_ERROR", "failed to fetch tariffs").
				WithDetail("contractID", c.ID)
		}
		if len(allTariffs) == 0 {
			return nil, NewApplyContractToRateError(
				"NO_TARIFFS",
				"contract has no tariffs",
			).WithDetail("contractID", c.ID)
		}
		return allTariffs, nil
	}

	// TariffIDs指定: 指定されたTariffのみ対象
	result := make([]*pricing.Tariff, 0, len(tariffIDs))
	for _, id := range tariffIDs {
		tariff, err := uc.tariffRepo.FindByID(ctx, id)
		if err != nil {
			return nil, NewApplyContractToRateError(
				"TARIFF_NOT_FOUND",
				fmt.Sprintf("tariff %s not found", id),
			).
				WithDetail("tariffID", id).
				WithDetail("contractID", c.ID)
		}
		if tariff.ContractID != c.ID {
			return nil, NewApplyContractToRateError(
				"TARIFF_NOT_IN_CONTRACT",
				fmt.Sprintf("tariff %s does not belong to contract %s", id, c.ID),
			).
				WithDetail("tariffID", id).
				WithDetail("contractID", c.ID)
		}
		result = append(result, tariff)
	}

	return result, nil
}

// addEntriesToRate: 料金表のLineItem単位でRateEntryを作成しレートに追加（ACL経由）
func (uc *ApplyContractToRateUseCase) addEntriesToRate(
	r *domainrate.Rate,
	c *contract.ServiceContract,
	tariffs []*pricing.Tariff,
	lineItemFilter []uuid.UUID,
) ([]AddedEntryDetail, error) {
	// LineItemIDフィルタをmapに変換
	filterSet := make(map[uuid.UUID]bool, len(lineItemFilter))
	for _, id := range lineItemFilter {
		filterSet[id] = true
	}

	addedEntries := make([]AddedEntryDetail, 0)

	for _, tariff := range tariffs {
		// ACL経由でLineItem情報を抽出
		rateableItems := pricing.ExtractRateableItems(tariff)

		for _, item := range rateableItems {
			// フィルタが指定されている場合、一致するもののみ処理
			if len(filterSet) > 0 && !filterSet[item.LineItemID] {
				continue
			}

			// RouteScopeの構築
			var routeScope domainrate.RouteScope
			if item.OriginID != nil {
				lid := route.LocationID(*item.OriginID)
				routeScope.OriginID = &lid
			}
			if item.DestinationID != nil {
				lid := route.LocationID(*item.DestinationID)
				routeScope.DestinationID = &lid
			}
			if item.TransportMode != nil {
				m := shared.TransportMode(*item.TransportMode)
				routeScope.TransportMode = &m
			}

			// UnitPriceの設定（nilの場合はゼロ値USD）
			unitPrice := shared.ZeroMoney("USD")
			if item.UnitPrice != nil {
				unitPrice = *item.UnitPrice
			}

			entry := &domainrate.RateEntry{
				ProviderID:       c.ProviderID,
				ContractID:       c.ID,
				TariffID:         tariff.ID,
				TariffLineItemID: item.LineItemID,
				RouteScope:       routeScope,
				ChargeCode:       item.ChargeCode,
				Category:         item.Category,
				UnitPrice:        unitPrice,
			}

			if err := r.AddEntry(entry); err != nil {
				return nil, NewApplyContractToRateError(
					"ADD_ENTRY_ERROR",
					fmt.Sprintf("failed to add entry for tariff %s line item %s: %s", tariff.ID, item.LineItemID, err.Error()),
				).
					WithDetail("tariffID", tariff.ID).
					WithDetail("lineItemID", item.LineItemID).
					WithDetail("rateID", r.ID)
			}

			addedEntries = append(addedEntries, AddedEntryDetail{
				EntryID:          entry.ID,
				TariffID:         tariff.ID,
				TariffLineItemID: item.LineItemID,
				TariffName:       tariff.Name,
				ChargeCode:       item.ChargeCode,
				Category:         item.Category,
				UnitPrice:        unitPrice,
				OriginID:         item.OriginID,
				DestinationID:    item.DestinationID,
				TransportMode:    item.TransportMode,
			})
		}
	}

	return addedEntries, nil
}
