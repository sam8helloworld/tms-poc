package tariff

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/adapter/parser"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/pricing"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/scope"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
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
	contractRepo  commercial.ServiceContractRepository
	parserFactory parser.TariffParserFactory
}

// NewAmendContractTariffUseCase: コンストラクタ
func NewAmendContractTariffUseCase(
	contractRepo commercial.ServiceContractRepository,
	parserFactory parser.TariffParserFactory,
) *AmendContractTariffUseCase {
	return &AmendContractTariffUseCase{
		contractRepo:  contractRepo,
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
	var baseTariff *commercial.Tariff
	for _, tariff := range contract.Tariffs() {
		if tariff.ID == input.BaseTariffID {
			baseTariff = tariff
			break
		}
	}
	if baseTariff == nil {
		return nil, NewAmendTariffError("BASE_TARIFF_NOT_FOUND", "base tariff not found in contract").
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
	newTariff, err := commercial.NewTariffVersion(baseTariff, effectiveFrom, effectiveTo)
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

		lineItem := commercial.TariffLineItem{
			ChargeCode: parsedItem.ChargeCode,
			Category:   parsedItem.Category,
			Scope:      serviceScope,
			Logic:      logic,
		}

		if err := newTariff.AddLineItem(lineItem); err != nil {
			return nil, NewAmendTariffError("LINE_ITEM_ERROR", "failed to add line item").
				WithCause(err)
		}
	}

	// 8. 契約に改定版を追加
	if err := contract.AddTariffAmendment(newTariff); err != nil {
		return nil, NewAmendTariffError("AMENDMENT_ERROR", "failed to add tariff amendment to contract").
			WithCause(err)
	}

	// 9. 契約を保存
	if err := uc.contractRepo.Save(ctx, contract); err != nil {
		return nil, NewAmendTariffError("SAVE_ERROR", "failed to save contract").
			WithCause(err)
	}

	// 10. レスポンスを構築
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
		TotalTariffCount: contract.TariffCount(),
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
	parsed parser.ParsedLineItem,
) (scope.ServiceScope, error) {
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

		return scope.LocationService{
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

		return scope.TransportationService{
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
	parsed parser.ParsedLineItem,
) (pricing.PricingStrategy, error) {
	switch parsed.PricingType {
	case "FLAT":
		return nil, errors.New("FlatStrategy construction not implemented (placeholder)")

	case "CEL_EXPRESSION":
		return nil, errors.New("CelExpressionStrategy construction not implemented (placeholder)")

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
