package tariff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/shopspring/decimal"
)

// RegisterTariffUseCase: 料金表登録ユースケース
// 入札プロセスにおいて、物流企業から受け取った料金表（PDFやExcel）を解析し、
// 契約（DRAFT状態）に登録する処理を担当する
type RegisterTariffUseCase struct {
	parserFactory pricing.TariffParserFactory
	validator     pricing.TariffDataValidator
	contractRepo  contract.ServiceContractRepository
	tariffRepo    pricing.TariffRepository
}

// NewRegisterTariffUseCase: RegisterTariffUseCaseのコンストラクタ
func NewRegisterTariffUseCase(
	parserFactory pricing.TariffParserFactory,
	validator pricing.TariffDataValidator,
	tariffRepo pricing.TariffRepository,
	contractRepo contract.ServiceContractRepository,
) *RegisterTariffUseCase {
	return &RegisterTariffUseCase{
		parserFactory: parserFactory,
		validator:     validator,
		tariffRepo:    tariffRepo,
		contractRepo:  contractRepo,
	}
}

// Execute: ユースケースを実行
func (uc *RegisterTariffUseCase) Execute(
	ctx context.Context,
	input RegisterTariffInput,
) (*RegisterTariffOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewRegisterTariffError("INVALID_INPUT", err.Error())
	}

	// 2. 契約の取得または新規作成
	c, isNewContract, err := uc.getOrCreateContract(ctx, input)
	if err != nil {
		return nil, err
	}

	// 3. ファイルの解析
	parsedData, err := uc.parseFile(input)
	if err != nil {
		return nil, NewRegisterTariffError("FILE_PARSE_ERROR", err.Error()).
			WithDetail("fileName", input.FileName).
			WithDetail("fileFormat", input.FileFormat)
	}

	// 4. 解析データのバリデーション
	validationResult := uc.validator.Validate(parsedData)
	if !validationResult.IsValid {
		return nil, uc.buildValidationError(validationResult)
	}

	// 5. ドメインモデルへの変換
	tariff, err := uc.convertToTariff(parsedData, c.ID)
	if err != nil {
		return nil, NewRegisterTariffError("CONVERSION_ERROR", err.Error())
	}

	// 6. 重複チェック
	existingTariffs, _ := uc.tariffRepo.FindByContractID(ctx, c.ID)
	isUpdated := uc.tariffExistsInList(existingTariffs, tariff)

	// 7. Tariffを永続化
	if err := uc.tariffRepo.Save(ctx, tariff); err != nil {
		return nil, NewRegisterTariffError("SAVE_ERROR", err.Error()).
			WithDetail("tariffID", tariff.ID)
	}

	// 8. 新規契約の場合は契約を永続化
	if isNewContract {
		if err := uc.contractRepo.Save(ctx, c); err != nil {
			return nil, NewRegisterTariffError("SAVE_ERROR", err.Error()).
				WithDetail("contractID", c.ID)
		}
	}

	// 9. Tariff件数取得
	totalTariffCount, _ := uc.tariffRepo.CountByContractID(ctx, c.ID)

	// 10. 出力DTOの作成
	output := &RegisterTariffOutput{
		ContractID:       c.ID,
		ContractStatus:   string(c.Status()),
		TariffID:         tariff.ID,
		TariffName:       tariff.Name,
		EffectiveFrom:    tariff.EffectiveDate.From,
		EffectiveTo:      tariff.EffectiveDate.To,
		LineItemCount:    len(tariff.LineItems),
		CreatedAt:        time.Now(),
		IsNewContract:    isNewContract,
		IsUpdatedTariff:  isUpdated,
		TotalTariffCount: totalTariffCount,
	}

	return output, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *RegisterTariffUseCase) validateInput(input RegisterTariffInput) error {
	if input.FileReader == nil {
		return errors.New("file reader is required")
	}
	if input.FileFormat == "" {
		return errors.New("file format is required")
	}
	if input.FileName == "" {
		return errors.New("file name is required")
	}
	if input.ProviderID == uuid.Nil {
		return errors.New("provider ID is required")
	}
	if input.ShipperID == uuid.Nil {
		return errors.New("shipper ID is required")
	}
	if input.UploadedBy == uuid.Nil {
		return errors.New("uploaded by user ID is required")
	}

	// 新規契約作成時は有効期間が必須
	if input.ContractID == nil {
		if input.ContractValidFrom == nil || input.ContractValidTo == nil {
			return errors.New("contract valid period is required for new contracts")
		}
	}

	return nil
}

// getOrCreateContract: 契約を取得または新規作成
func (uc *RegisterTariffUseCase) getOrCreateContract(
	ctx context.Context,
	input RegisterTariffInput,
) (*contract.ServiceContract, bool, error) {
	// 既存の契約IDが指定されている場合は取得
	if input.ContractID != nil {
		c, err := uc.contractRepo.FindByID(ctx, *input.ContractID)
		if err != nil {
			return nil, false, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract not found").
				WithDetail("contractID", *input.ContractID)
		}
		if !c.IsDraft() {
			return nil, false, NewRegisterTariffError("CONTRACT_NOT_DRAFT", "contract is not in DRAFT status").
				WithDetail("contractID", *input.ContractID).
				WithDetail("status", string(c.Status()))
		}
		return c, false, nil
	}

	// 新規契約を作成（DRAFT状態）
	c, err := contract.NewServiceContract(
		input.ProviderID,
		input.ShipperID,
		*input.ContractValidFrom,
		*input.ContractValidTo,
	)
	if err != nil {
		return nil, false, NewRegisterTariffError("CONTRACT_CREATE_ERROR", err.Error())
	}

	return c, true, nil
}

// tariffExistsInList: Tariffリスト内に同じTariffが存在するかチェック
func (uc *RegisterTariffUseCase) tariffExistsInList(
	tariffs []*pricing.Tariff,
	tariff *pricing.Tariff,
) bool {
	for _, existing := range tariffs {
		if existing.Name == tariff.Name &&
			existing.EffectiveDate.From.Equal(tariff.EffectiveDate.From) &&
			existing.EffectiveDate.To.Equal(tariff.EffectiveDate.To) {
			return true
		}
	}
	return false
}

// parseFile: ファイルを解析
func (uc *RegisterTariffUseCase) parseFile(input RegisterTariffInput) (*pricing.ParsedTariffData, error) {
	p, err := uc.parserFactory.GetParser(input.FileFormat)
	if err != nil {
		return nil, fmt.Errorf("unsupported file format: %s", input.FileFormat)
	}

	parsedData, err := p.Parse(input.FileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	if input.OverrideTariffName != nil {
		parsedData.TariffName = *input.OverrideTariffName
	}
	if input.OverrideEffectiveFrom != nil {
		parsedData.EffectiveFrom = *input.OverrideEffectiveFrom
	}
	if input.OverrideEffectiveTo != nil {
		parsedData.EffectiveTo = *input.OverrideEffectiveTo
	}

	return parsedData, nil
}

// buildValidationError: バリデーションエラーを構築
func (uc *RegisterTariffUseCase) buildValidationError(result *pricing.ValidationResult) *RegisterTariffError {
	err := NewRegisterTariffError("VALIDATION_ERROR", "parsed data validation failed")
	for _, ve := range result.Errors {
		err.WithDetail(ve.Field, ve.Message)
	}
	return err
}

// convertToTariff: 解析データをドメインモデルに変換
func (uc *RegisterTariffUseCase) convertToTariff(
	data *pricing.ParsedTariffData,
	contractID uuid.UUID,
) (*pricing.Tariff, error) {
	tariff, err := pricing.NewTariff(
		data.TariffName,
		contractID,
		data.EffectiveFrom,
		data.EffectiveTo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tariff: %w", err)
	}

	for i, parsedItem := range data.LineItems {
		lineItem, err := uc.convertToLineItem(parsedItem)
		if err != nil {
			return nil, fmt.Errorf("failed to convert line item at index %d: %w", i, err)
		}

		if err := tariff.AddLineItem(*lineItem); err != nil {
			return nil, fmt.Errorf("failed to add line item at index %d: %w", i, err)
		}
	}

	return tariff, nil
}

// convertToLineItem: ParsedLineItemをドメインのTariffLineItemに変換
func (uc *RegisterTariffUseCase) convertToLineItem(
	parsed pricing.ParsedLineItem,
) (*pricing.TariffLineItem, error) {
	serviceScope, err := uc.buildServiceScope(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to build service scope: %w", err)
	}

	pricingLogic, err := uc.buildPricingStrategy(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to build pricing strategy: %w", err)
	}

	return &pricing.TariffLineItem{
		ID:         uuid.New(),
		ChargeCode: parsed.ChargeCode,
		Category:   parsed.Category,
		Scope:      serviceScope,
		Logic:      pricingLogic,
	}, nil
}

// buildServiceScope: ServiceScopeを構築
func (uc *RegisterTariffUseCase) buildServiceScope(
	parsed pricing.ParsedLineItem,
) (pricing.ServiceScope, error) {
	switch parsed.ServiceScopeType {
	case pricing.ScopeLocation:
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

	case pricing.ScopeTransportation:
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

// buildPricingStrategy: PricingStrategyを構築
func (uc *RegisterTariffUseCase) buildPricingStrategy(
	parsed pricing.ParsedLineItem,
) (pricing.PricingStrategy, error) {
	switch parsed.PricingType {
	case pricing.PricingFlat:
		return uc.buildFlatStrategy(parsed.PricingAttrs)

	case pricing.PricingExpression:
		return uc.buildExpressionStrategy(parsed.PricingAttrs)

	case pricing.PricingComposite:
		return uc.buildCompositeStrategy(parsed.PricingAttrs)

	default:
		return nil, fmt.Errorf("unsupported pricing type: %s", parsed.PricingType)
	}
}

// buildFlatStrategy: FlatStrategyを構築
func (uc *RegisterTariffUseCase) buildFlatStrategy(
	attrs map[string]any,
) (*pricing.FlatStrategy, error) {
	amountRaw, ok := attrs["Amount"]
	if !ok {
		return nil, errors.New("Amount is required for FLAT pricing")
	}
	amount, err := toDecimal(amountRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid Amount: %w", err)
	}

	currencyRaw, ok := attrs["Currency"]
	if !ok {
		return nil, errors.New("Currency is required for FLAT pricing")
	}
	currency, ok := currencyRaw.(string)
	if !ok {
		return nil, errors.New("Currency must be a string")
	}

	money, err := shared.NewMoney(amount, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to create money: %w", err)
	}

	return &pricing.FlatStrategy{Amount: money}, nil
}

// buildExpressionStrategy: ExpressionStrategyを構築
func (uc *RegisterTariffUseCase) buildExpressionStrategy(
	attrs map[string]any,
) (*pricing.ExpressionStrategy, error) {
	formulaRaw, ok := attrs["Formula"]
	if !ok {
		return nil, errors.New("Formula is required for EXPRESSION pricing")
	}
	formula, ok := formulaRaw.(string)
	if !ok {
		return nil, errors.New("Formula must be a string")
	}

	currencyRaw, ok := attrs["Currency"]
	if !ok {
		return nil, errors.New("Currency is required for EXPRESSION pricing")
	}
	currency, ok := currencyRaw.(string)
	if !ok {
		return nil, errors.New("Currency must be a string")
	}

	return &pricing.ExpressionStrategy{Formula: formula, Currency: currency}, nil
}

// buildCompositeStrategy: CompositeStrategyを構築
func (uc *RegisterTariffUseCase) buildCompositeStrategy(
	attrs map[string]any,
) (*pricing.CompositeStrategy, error) {
	stepsRaw, ok := attrs["Steps"]
	if !ok {
		return nil, errors.New("Steps is required for COMPOSITE pricing")
	}

	var stepMaps []map[string]any
	switch v := stepsRaw.(type) {
	case []map[string]any:
		stepMaps = v
	case []any:
		for i, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("step[%d] must be a map", i)
			}
			stepMaps = append(stepMaps, m)
		}
	default:
		return nil, errors.New("Steps must be an array of pricing definitions")
	}

	if len(stepMaps) == 0 {
		return nil, errors.New("Steps must contain at least one pricing definition")
	}

	var strategies []pricing.PricingStrategy
	for i, stepAttrs := range stepMaps {
		typeRaw, ok := stepAttrs["Type"]
		if !ok {
			return nil, fmt.Errorf("step[%d]: Type is required", i)
		}
		typeName, ok := typeRaw.(string)
		if !ok {
			return nil, fmt.Errorf("step[%d]: Type must be a string", i)
		}

		subParsed := pricing.ParsedLineItem{
			PricingType:  pricing.PricingStrategyType(typeName),
			PricingAttrs: stepAttrs,
		}
		strategy, err := uc.buildPricingStrategy(subParsed)
		if err != nil {
			return nil, fmt.Errorf("step[%d]: %w", i, err)
		}
		strategies = append(strategies, strategy)
	}

	return &pricing.CompositeStrategy{Steps: strategies}, nil
}

// parseTransportMode: 文字列からTransportModeに変換
func (uc *RegisterTariffUseCase) parseTransportMode(modeStr string) (shared.TransportMode, error) {
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

// toDecimal: any型の値をdecimal.Decimalに変換するヘルパー
func toDecimal(v any) (decimal.Decimal, error) {
	switch val := v.(type) {
	case decimal.Decimal:
		return val, nil
	case float64:
		return decimal.NewFromFloat(val), nil
	case float32:
		return decimal.NewFromFloat32(val), nil
	case int:
		return decimal.NewFromInt(int64(val)), nil
	case int64:
		return decimal.NewFromInt(val), nil
	case string:
		return decimal.NewFromString(val)
	default:
		return decimal.Decimal{}, fmt.Errorf("unsupported type for decimal conversion: %T", v)
	}
}
