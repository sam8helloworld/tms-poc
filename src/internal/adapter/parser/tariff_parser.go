package parser

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// ParsedTariffData: ファイルから解析された料金表データ
// ドメインモデルへの変換前の中間データ構造
type ParsedTariffData struct {
	TariffName    string
	EffectiveFrom time.Time
	EffectiveTo   time.Time
	LineItems     []ParsedLineItem
}

// ParsedLineItem: 解析された料金明細1行のデータ
type ParsedLineItem struct {
	ChargeCode        string
	ChargeName        string // 外部システムでの料金名称
	Category          string
	ServiceScopeType  string            // "LOCATION" or "TRANSPORTATION"
	ServiceScopeAttrs map[string]string // LocationID, OriginID, DestinationID, Mode など
	PricingType       string            // "FLAT", "CEL_EXPRESSION", "COMPOSITE"
	PricingAttrs      map[string]any    // Amount, Currency, Formula など
}

// TariffParser: 料金表ファイルを解析するインターフェース
// 実装はインフラ層で行う（PDFParser, ExcelParser など）
type TariffParser interface {
	// Parse: ファイルから料金表データを解析
	Parse(reader io.Reader) (*ParsedTariffData, error)

	// SupportedFormats: このパーサーがサポートするファイル形式
	SupportedFormats() []string // ["pdf", "xlsx", "csv"]
}

// TariffParserFactory: ファイル形式に応じた適切なパーサーを返すファクトリー
type TariffParserFactory interface {
	// GetParser: ファイル形式に応じたパーサーを取得
	GetParser(format string) (TariffParser, error)
}

// TariffDataConverter: 解析データをドメインモデルに変換するインターフェース
// 本来はusecase層で実装されるが、複雑な変換ロジックの場合は専用のコンバーターを用意
type TariffDataConverter interface {
	// ConvertToLineItem: ParsedLineItemをドメインのTariffLineItemに変換
	// LocationやPricingStrategyの解決が必要なため、リポジトリへの依存が生じる
	// ConvertToLineItem(ctx context.Context, parsed ParsedLineItem) (*pricing.TariffLineItem, error)
}

// ValidationResult: 解析データのバリデーション結果
type ValidationResult struct {
	IsValid bool
	Errors  []ValidationError
}

// ValidationError: バリデーションエラー詳細
type ValidationError struct {
	Field   string // "LineItem[3].ChargeCode"
	Message string // "Charge code is required"
	LineNo  int    // 元ファイルの行番号（あれば）
}

// TariffDataValidator: 解析データのバリデーター
type TariffDataValidator interface {
	// Validate: 解析データの妥当性を検証
	Validate(data *ParsedTariffData) *ValidationResult
}

// FileMetadata: アップロードされたファイルのメタデータ
type FileMetadata struct {
	OriginalFilename string
	FileSize         int64
	ContentType      string // "application/pdf", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	UploadedBy       uuid.UUID
	UploadedAt       time.Time
}
