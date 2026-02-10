package tariff

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// AddTariffVersionInput: 料金表バージョン追加の入力
type AddTariffVersionInput struct {
	FileReader         io.Reader
	FileFormat         string    // "csv", "excel", "json"
	FileName           string
	ContractID         uuid.UUID // 料金表を追加する契約のID
	BaseTariffID       uuid.UUID // バージョンアップ元のTariffID
	UploadedBy         uuid.UUID
	EffectiveFrom      *time.Time // 新バージョンの有効期間開始（オプション：ファイルから取得可能）
	EffectiveTo        *time.Time // 新バージョンの有効期間終了（オプション：ファイルから取得可能）
}

// AddTariffVersionOutput: 料金表バージョン追加の出力
type AddTariffVersionOutput struct {
	ContractID       uuid.UUID
	ContractStatus   string
	TariffID         uuid.UUID
	TariffName       string
	TariffVersion    int
	BaseTariffID     uuid.UUID
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	LineItemCount    int
	TotalTariffCount int
	Message          string
}

// AddTariffVersionUseCase: 既存料金表の新バージョンを追加するユースケース
// 同じ名前のTariffの改定版を受け取った場合、履歴を保持しつつ新バージョンとして追加する
type AddTariffVersionUseCase struct {
	contractRepo  contract.ServiceContractRepository
	tariffRepo    pricing.TariffRepository
	parserFactory pricing.TariffParserFactory
}

// NewAddTariffVersionUseCase: コンストラクタ
func NewAddTariffVersionUseCase(
	contractRepo contract.ServiceContractRepository,
	tariffRepo pricing.TariffRepository,
	parserFactory pricing.TariffParserFactory,
) *AddTariffVersionUseCase {
	return &AddTariffVersionUseCase{
		contractRepo:  contractRepo,
		tariffRepo:    tariffRepo,
		parserFactory: parserFactory,
	}
}

// Execute: ユースケースの実行
func (uc *AddTariffVersionUseCase) Execute(
	ctx context.Context,
	input AddTariffVersionInput,
) (*AddTariffVersionOutput, error) {
	// 1. 契約を取得
	contract, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// DRAFT状態のみ料金表追加可能
	if !contract.IsDraft() {
		return nil, NewRegisterTariffError(
			"INVALID_CONTRACT_STATE",
			"tariffs can only be added to DRAFT contracts",
		).WithDetail("currentStatus", string(contract.Status()))
	}

	// 2. ベースとなるTariffを取得
	baseTariff, err := uc.tariffRepo.FindByID(ctx, input.BaseTariffID)
	if err != nil || baseTariff == nil {
		return nil, NewRegisterTariffError("BASE_TARIFF_NOT_FOUND", "base tariff not found").
			WithDetail("baseTariffID", input.BaseTariffID).
			WithDetail("contractID", input.ContractID)
	}
	if baseTariff.ContractID != input.ContractID {
		return nil, NewRegisterTariffError("BASE_TARIFF_NOT_IN_CONTRACT", "base tariff does not belong to the specified contract").
			WithDetail("baseTariffID", input.BaseTariffID).
			WithDetail("contractID", input.ContractID)
	}

	// 3. ファイルをパース
	parser, err := uc.parserFactory.GetParser(input.FileFormat)
	if err != nil {
		return nil, NewRegisterTariffError("UNSUPPORTED_FORMAT", "unsupported file format").
			WithDetail("format", input.FileFormat)
	}

	parsedData, err := parser.Parse(input.FileReader)
	if err != nil {
		return nil, NewRegisterTariffError("PARSE_ERROR", "failed to parse tariff file").
			WithCause(err)
	}

	// 4. 有効期間の決定（入力 > ファイル）
	effectiveFrom := parsedData.EffectiveFrom
	if input.EffectiveFrom != nil {
		effectiveFrom = *input.EffectiveFrom
	}
	effectiveTo := parsedData.EffectiveTo
	if input.EffectiveTo != nil {
		effectiveTo = *input.EffectiveTo
	}

	// 5. 新バージョンのTariffを作成
	newTariff, err := pricing.NewTariffVersion(baseTariff, effectiveFrom, effectiveTo)
	if err != nil {
		return nil, NewRegisterTariffError("TARIFF_CREATE_ERROR", "failed to create tariff version").
			WithCause(err)
	}

	// 6. LineItemsを追加
	for _, parsedItem := range parsedData.LineItems {
		scope, err := uc.buildServiceScope(parsedItem)
		if err != nil {
			return nil, NewRegisterTariffError("SCOPE_BUILD_ERROR", "failed to build service scope").
				WithCause(err)
		}

		logic, err := uc.buildPricingStrategy(parsedItem)
		if err != nil {
			return nil, NewRegisterTariffError("LOGIC_BUILD_ERROR", "failed to build pricing logic").
				WithCause(err)
		}

		lineItem := pricing.TariffLineItem{
			ChargeCode: parsedItem.ChargeCode,
			Category:   parsedItem.Category,
			Scope:      scope,
			Logic:      logic,
		}

		if err := newTariff.AddLineItem(lineItem); err != nil {
			return nil, NewRegisterTariffError("LINE_ITEM_ERROR", "failed to add line item").
				WithCause(err)
		}
	}

	// 7. Tariffを保存
	if err := uc.tariffRepo.Save(ctx, newTariff); err != nil {
		return nil, NewRegisterTariffError("SAVE_ERROR", "failed to save tariff").
			WithCause(err)
	}

	// 8. Tariff件数を取得
	totalTariffCount, err := uc.tariffRepo.CountByContractID(ctx, contract.ID)
	if err != nil {
		return nil, NewRegisterTariffError("COUNT_ERROR", "failed to count tariffs").
			WithCause(err)
	}

	// 9. レスポンスを構築
	return &AddTariffVersionOutput{
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
		Message:          "New tariff version added successfully. Old version is retained for history.",
	}, nil
}

// buildServiceScope: ParsedLineItemからServiceScopeを構築
// （RegisterTariffUseCaseと同じロジック）
func (uc *AddTariffVersionUseCase) buildServiceScope(
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
// （RegisterTariffUseCaseと同じロジック）
func (uc *AddTariffVersionUseCase) buildPricingStrategy(
	parsed pricing.ParsedLineItem,
) (pricing.PricingStrategy, error) {
	switch parsed.PricingType {
	case "FLAT":
		// FlatStrategyの構築（プレースホルダー）
		return nil, errors.New("FlatStrategy construction not implemented (placeholder)")

	case "CEL_EXPRESSION":
		// CelExpressionStrategyの構築（プレースホルダー）
		return nil, errors.New("CelExpressionStrategy construction not implemented (placeholder)")

	default:
		return nil, fmt.Errorf("unsupported pricing type: %s", parsed.PricingType)
	}
}

// parseTransportMode: 文字列からTransportModeに変換
func (uc *AddTariffVersionUseCase) parseTransportMode(modeStr string) (shared.TransportMode, error) {
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
