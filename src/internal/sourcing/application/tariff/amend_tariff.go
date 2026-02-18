package tariff

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// AmendContractTariffUseCase: CONTRACTED状態の契約に対して料金表の改定版を追加するユースケース
// BAF等の市況変動や料金表の誤りにより、既に成立した契約の料金表を改定する
//
// 処理の流れ:
// 1. 入力バリデーション
// 2. 契約を取得し、CONTRACTED状態であることを確認
// 3. ベースとなるTariffを契約内から検索
// 4. ファイルをパースして新Tariff内容を取得
// 5. NewTariffVersion()で新バージョン作成
// 6. LineItemを構築・追加
// 7. contract.AddTariffAmendment(newTariff)で契約に追加
// 8. 永続化
// 9. 出力DTOを返却
type AmendContractTariffUseCase struct {
	contractRepo  contract.ServiceContractRepository
	tariffRepo    pricing.TariffRepository
	parserFactory pricing.TariffParserFactory
}

// NewAmendContractTariffUseCase: コンストラクタ
func NewAmendContractTariffUseCase(
	contractRepo contract.ServiceContractRepository,
	tariffRepo pricing.TariffRepository,
	parserFactory pricing.TariffParserFactory,
) *AmendContractTariffUseCase {
	return &AmendContractTariffUseCase{
		contractRepo:  contractRepo,
		tariffRepo:    tariffRepo,
		parserFactory: parserFactory,
	}
}

// Execute: ユースケースの実行
func (uc *AmendContractTariffUseCase) Execute(
	ctx context.Context,
	input AmendContractTariffInput,
) (*AmendContractTariffOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, err
	}

	// 2. 契約を取得
	contract, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewAmendTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// CONTRACTED状態のみ改定可能
	if !contract.IsActive() {
		return nil, NewAmendTariffError(
			"INVALID_CONTRACT_STATE",
			"tariff amendments can only be added to CONTRACTED contracts",
		).WithDetail("currentStatus", string(contract.Status()))
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

	// 4. ファイルをパース
	p, err := uc.parserFactory.GetParser(input.FileFormat)
	if err != nil {
		return nil, NewAmendTariffError("UNSUPPORTED_FORMAT", "unsupported file format").
			WithDetail("format", input.FileFormat)
	}

	parsedData, err := p.Parse(input.FileReader)
	if err != nil {
		return nil, NewAmendTariffError("PARSE_ERROR", "failed to parse tariff file").
			WithCause(err)
	}

	// 5. 有効期間の決定（入力 > ファイル）
	effectiveFrom := parsedData.EffectiveFrom
	if input.EffectiveFrom != nil {
		effectiveFrom = *input.EffectiveFrom
	}
	effectiveTo := parsedData.EffectiveTo
	if input.EffectiveTo != nil {
		effectiveTo = *input.EffectiveTo
	}

	// 6. 新バージョンのTariffを作成
	newTariff, err := pricing.NewTariffVersion(baseTariff, effectiveFrom, effectiveTo)
	if err != nil {
		return nil, NewAmendTariffError("TARIFF_CREATE_ERROR", "failed to create tariff version").
			WithCause(err)
	}

	// 7. LineItemsを追加
	for _, parsedItem := range parsedData.LineItems {
		serviceScope, err := uc.buildServiceScope(parsedItem)
		if err != nil {
			return nil, NewAmendTariffError("SCOPE_BUILD_ERROR", "failed to build service scope").
				WithCause(err)
		}

		logic, err := uc.buildPricingStrategy(parsedItem)
		if err != nil {
			return nil, NewAmendTariffError("LOGIC_BUILD_ERROR", "failed to build pricing logic").
				WithCause(err)
		}

		lineItem := pricing.TariffLineItem{
			ChargeCode:       parsedItem.ChargeCode,
			Category:         parsedItem.Category,
			Scope:            serviceScope,
			Logic:            logic,
			OperatorVendorID: parsedItem.OperatorVendorID,
		}

		if err := newTariff.AddLineItem(lineItem); err != nil {
			return nil, NewAmendTariffError("LINE_ITEM_ERROR", "failed to add line item").
				WithCause(err)
		}
	}

	// 8. IsNewVersionチェック（UseCase層で実施）
	if !newTariff.IsNewVersion() {
		return nil, NewAmendTariffError("NOT_NEW_VERSION", "only new versions of existing tariffs can be added as amendments")
	}

	// 9. Tariffを保存
	if err := uc.tariffRepo.Save(ctx, newTariff); err != nil {
		return nil, NewAmendTariffError("SAVE_ERROR", "failed to save tariff").
			WithCause(err)
	}

	// 10. Tariff件数を取得
	totalTariffCount, err := uc.tariffRepo.CountByContractID(ctx, contract.ID)
	if err != nil {
		return nil, NewAmendTariffError("COUNT_ERROR", "failed to count tariffs").
			WithCause(err)
	}

	// 11. レスポンスを構築
	return &AmendContractTariffOutput{
		ContractID:       contract.ID,
		ContractStatus:   string(contract.Status()),
		TariffID:         newTariff.ID,
		TariffName:       newTariff.Name,
		TariffVersion:    newTariff.Version,
		BaseTariffID:     *newTariff.BaseVersionID,
		EffectiveFrom:    newTariff.EffectiveDate.From,
		EffectiveTo:      newTariff.EffectiveDate.To,
		LineItemCount:    len(newTariff.LineItems),
		TotalTariffCount: totalTariffCount,
		Message:          "Tariff amendment added successfully to contracted contract.",
	}, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *AmendContractTariffUseCase) validateInput(input AmendContractTariffInput) error {
	if input.FileReader == nil {
		return NewAmendTariffError("INVALID_INPUT", "file reader is required")
	}
	if input.FileFormat == "" {
		return NewAmendTariffError("INVALID_INPUT", "file format is required")
	}
	if input.FileName == "" {
		return NewAmendTariffError("INVALID_INPUT", "file name is required")
	}
	if input.ContractID == uuid.Nil {
		return NewAmendTariffError("INVALID_INPUT", "contractID is required")
	}
	if input.BaseTariffID == uuid.Nil {
		return NewAmendTariffError("INVALID_INPUT", "baseTariffID is required")
	}
	if input.UploadedBy == uuid.Nil {
		return NewAmendTariffError("INVALID_INPUT", "uploadedBy is required")
	}
	return nil
}

// buildServiceScope: ParsedLineItemからServiceScopeを構築
func (uc *AmendContractTariffUseCase) buildServiceScope(
	parsed pricing.ParsedLineItem,
) (pricing.ServiceScope, error) {
	switch parsed.ServiceScopeType {
	case "LOCATION":
		locationIDStr, ok := parsed.ServiceScopeAttrs["LocationID"]
		if !ok {
			return nil, errors.New("LocationID is required for LOCATION scope")
		}
		locationID, err := uuid.Parse(locationIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid LocationID: %w", err)
		}

		serviceType, ok := parsed.ServiceScopeAttrs["ServiceType"]
		if !ok {
			serviceType = "HANDLING"
		}

		return pricing.LocationService{
			LocationID:  route.LocationID(locationID),
			ServiceType: serviceType,
		}, nil

	case "TRANSPORTATION":
		originIDStr, ok := parsed.ServiceScopeAttrs["OriginID"]
		if !ok {
			return nil, errors.New("OriginID is required for TRANSPORTATION scope")
		}
		originID, err := uuid.Parse(originIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid OriginID: %w", err)
		}

		destIDStr, ok := parsed.ServiceScopeAttrs["DestinationID"]
		if !ok {
			return nil, errors.New("DestinationID is required for TRANSPORTATION scope")
		}
		destID, err := uuid.Parse(destIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DestinationID: %w", err)
		}

		modeStr, ok := parsed.ServiceScopeAttrs["Mode"]
		if !ok {
			return nil, errors.New("Mode is required for TRANSPORTATION scope")
		}
		mode, err := uc.parseTransportMode(modeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid transport mode: %w", err)
		}

		return pricing.TransportationService{
			OriginID:      route.LocationID(originID),
			DestinationID: route.LocationID(destID),
			Mode:          mode,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported service scope type: %s", parsed.ServiceScopeType)
	}
}

// buildPricingStrategy: ParsedLineItemからPricingStrategyを構築
func (uc *AmendContractTariffUseCase) buildPricingStrategy(
	parsed pricing.ParsedLineItem,
) (pricing.PricingStrategy, error) {
	switch parsed.PricingType {
	case "FLAT":
		return nil, errors.New("FlatStrategy construction not implemented (placeholder)")

	case "EXPRESSION":
		return nil, errors.New("ExpressionStrategy construction not implemented (placeholder)")

	default:
		return nil, fmt.Errorf("unsupported pricing type: %s", parsed.PricingType)
	}
}

// parseTransportMode: 文字列からTransportModeに変換
func (uc *AmendContractTariffUseCase) parseTransportMode(modeStr string) (shared.TransportMode, error) {
	validModes := map[string]shared.TransportMode{
		"OCEAN":   shared.ModeOcean,
		"AIR":     shared.ModeAir,
		"TRUCK":   shared.ModeTruck,
		"Railway": shared.ModeRailway,
	}

	mode, ok := validModes[modeStr]
	if !ok {
		return "", fmt.Errorf("invalid transport mode: %s", modeStr)
	}

	return mode, nil
}
