package tariff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/adapter/parser"
	"github.com/sam8helloworld/tms-poc/internal/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/domain/pricing"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// RegisterTariffUseCase: 料金表登録ユースケース
// 入札プロセスにおいて、物流企業から受け取った料金表（PDFやExcel）を解析し、
// 契約（DRAFT状態）に登録する処理を担当する
//
// 処理の流れ:
// 1. 既存のDRAFT契約を取得、または新規作成
// 2. ファイルを解析してTariffドメインモデルに変換
// 3. 契約にTariffを追加または更新
// 4. 契約を永続化
type RegisterTariffUseCase struct {
	parserFactory parser.TariffParserFactory
	validator     parser.TariffDataValidator

	// 本来利用すべきリポジトリ（コメントアウト）
	// contractRepo contract.ServiceContractRepository
	// tariffRepo   pricing.TariffRepository
}

// NewRegisterTariffUseCase: RegisterTariffUseCaseのコンストラクタ
func NewRegisterTariffUseCase(
	parserFactory parser.TariffParserFactory,
	validator parser.TariffDataValidator,
	// contractRepo contract.ServiceContractRepository,
	// tariffRepo pricing.TariffRepository,
) *RegisterTariffUseCase {
	return &RegisterTariffUseCase{
		parserFactory: parserFactory,
		validator:     validator,
		// contractRepo:  contractRepo,
		// tariffRepo:    tariffRepo,
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
	contract, isNewContract, err := uc.getOrCreateContract(ctx, input)
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
	tariff, err := uc.convertToTariff(parsedData, contract.ID)
	if err != nil {
		return nil, NewRegisterTariffError("CONVERSION_ERROR", err.Error())
	}

	// 6. 重複チェック（コメントアウト）
	// existingTariffs, _ := uc.tariffRepo.FindByContractID(ctx, contract.ID)
	// isUpdated := uc.tariffExistsInList(existingTariffs, tariff)
	isUpdated := false

	// 7. Tariffを永続化（コメントアウト）
	// if err := uc.tariffRepo.Save(ctx, tariff); err != nil {
	// 	return nil, NewRegisterTariffError("SAVE_ERROR", err.Error()).
	// 		WithDetail("tariffID", tariff.ID)
	// }

	// 8. 契約を永続化（コメントアウト）
	// if err := uc.contractRepo.Save(ctx, contract); err != nil {
	// 	return nil, NewRegisterTariffError("SAVE_ERROR", err.Error()).
	// 		WithDetail("contractID", contract.ID)
	// }

	// 9. Tariff件数取得（コメントアウト）
	// totalTariffCount, _ := uc.tariffRepo.CountByContractID(ctx, contract.ID)
	totalTariffCount := 1

	// 10. 出力DTOの作成
	output := &RegisterTariffOutput{
		ContractID:       contract.ID,
		ContractStatus:   string(contract.Status()),
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
		// contract, err := uc.contractRepo.FindByID(ctx, *input.ContractID)
		// if err != nil {
		// 	return nil, false, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract not found").
		// 		WithDetail("contractID", *input.ContractID)
		// }
		// if contract == nil {
		// 	return nil, false, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract does not exist").
		// 		WithDetail("contractID", *input.ContractID)
		// }
		// if !contract.IsDraft() {
		// 	return nil, false, NewRegisterTariffError("CONTRACT_NOT_DRAFT", "contract is not in DRAFT status").
		// 		WithDetail("contractID", *input.ContractID).
		// 		WithDetail("status", string(contract.Status))
		// }
		// return contract, false, nil

		// コメントアウト中のため、ダミーのDRAFT契約を返す
		contract, err := contract.NewServiceContract(
			input.ProviderID,
			input.ShipperID,
			*input.ContractValidFrom,
			*input.ContractValidTo,
		)
		if err != nil {
			return nil, false, NewRegisterTariffError("CONTRACT_CREATE_ERROR", err.Error())
		}
		contract.ID = *input.ContractID
		return contract, false, nil
	}

	// 新規契約を作成（DRAFT状態）
	contract, err := contract.NewServiceContract(
		input.ProviderID,
		input.ShipperID,
		*input.ContractValidFrom,
		*input.ContractValidTo,
	)
	if err != nil {
		return nil, false, NewRegisterTariffError("CONTRACT_CREATE_ERROR", err.Error())
	}

	return contract, true, nil
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
func (uc *RegisterTariffUseCase) parseFile(input RegisterTariffInput) (*parser.ParsedTariffData, error) {
	// パーサーを取得
	p, err := uc.parserFactory.GetParser(input.FileFormat)
	if err != nil {
		return nil, fmt.Errorf("unsupported file format: %s", input.FileFormat)
	}

	// ファイルを解析
	parsedData, err := p.Parse(input.FileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// オーバーライド処理
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
func (uc *RegisterTariffUseCase) buildValidationError(result *parser.ValidationResult) *RegisterTariffError {
	err := NewRegisterTariffError("VALIDATION_ERROR", "parsed data validation failed")
	for _, ve := range result.Errors {
		err.WithDetail(ve.Field, ve.Message)
	}
	return err
}

// convertToTariff: 解析データをドメインモデルに変換
func (uc *RegisterTariffUseCase) convertToTariff(
	data *parser.ParsedTariffData,
	contractID uuid.UUID,
) (*pricing.Tariff, error) {
	// Tariff生成（独立した集約ルートとしてContractIDで契約を参照）
	tariff, err := pricing.NewTariff(
		data.TariffName,
		contractID,
		data.EffectiveFrom,
		data.EffectiveTo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tariff: %w", err)
	}

	// LineItemsの変換と追加
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
	parsed parser.ParsedLineItem,
) (*pricing.TariffLineItem, error) {
	// ServiceScopeの構築
	serviceScope, err := uc.buildServiceScope(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to build service scope: %w", err)
	}

	// PricingStrategyの構築
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
	parsed parser.ParsedLineItem,
) (pricing.ServiceScope, error) {
	switch parsed.ServiceScopeType {
	case "LOCATION":
		// LocationServiceの構築
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
			serviceType = "HANDLING" // デフォルト
		}

		return pricing.LocationService{
			LocationID:  route.LocationID(locationID),
			ServiceType: serviceType,
		}, nil

	case "TRANSPORTATION":
		// TransportationServiceの構築
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

		// TransportModeの解決
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
	parsed parser.ParsedLineItem,
) (pricing.PricingStrategy, error) {
	switch parsed.PricingType {
	case "FLAT":
		// FlatStrategyの構築
		amountRaw, ok := parsed.PricingAttrs["Amount"]
		if !ok {
			return nil, errors.New("Amount is required for FLAT pricing")
		}
		// 型アサーション（簡略化）
		// 本来はdecimal.Decimalへの変換が必要
		_ = amountRaw

		currencyRaw, ok := parsed.PricingAttrs["Currency"]
		if !ok {
			return nil, errors.New("Currency is required for FLAT pricing")
		}
		currency, ok := currencyRaw.(string)
		if !ok {
			return nil, errors.New("Currency must be a string")
		}

		// TODO: FlatStrategyの生成（実装は省略）
		_ = currency
		return nil, errors.New("FlatStrategy construction not implemented (placeholder)")

	case "CEL_EXPRESSION":
		// CelExpressionStrategyの構築
		formulaRaw, ok := parsed.PricingAttrs["Formula"]
		if !ok {
			return nil, errors.New("Formula is required for CEL_EXPRESSION pricing")
		}
		formula, ok := formulaRaw.(string)
		if !ok {
			return nil, errors.New("Formula must be a string")
		}

		currencyRaw, ok := parsed.PricingAttrs["Currency"]
		if !ok {
			return nil, errors.New("Currency is required for CEL_EXPRESSION pricing")
		}
		currency, ok := currencyRaw.(string)
		if !ok {
			return nil, errors.New("Currency must be a string")
		}

		// TODO: CelExpressionStrategyの生成（実装は省略）
		_, _ = formula, currency
		return nil, errors.New("CelExpressionStrategy construction not implemented (placeholder)")

	default:
		return nil, fmt.Errorf("unsupported pricing type: %s", parsed.PricingType)
	}
}

// parseTransportMode: 文字列からTransportModeに変換
func (uc *RegisterTariffUseCase) parseTransportMode(modeStr string) (shared.TransportMode, error) {
	// shared.TransportMode 型に変換
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
