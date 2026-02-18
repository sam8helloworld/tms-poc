package pricing

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// ParsedTariffData: ファイルから解析された料金表データ
// ドメインモデルへの変換前の中間データ構造（DTO）
//
// 設計上の注意:
// - これはドメインモデルではなく、外部入力の中間表現
// - Infrastructure層のParserが生成し、Application層がドメインモデルに変換する
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
	ServiceScopeType  ServiceScopeType    // ScopeLocation or ScopeTransportation
	ServiceScopeAttrs map[string]string   // LocationID, OriginID, DestinationID, Mode など
	PricingType       PricingStrategyType // PricingFlat, PricingExpression, PricingComposite
	PricingAttrs      map[string]any      // Amount, Currency, Formula, Steps など
	OperatorVendorID  *uuid.UUID          // 実際の作業業者（任意）
}

// TariffParser: 料金表ファイルを解析するインターフェース
//
// 設計上の注意:
// - このインターフェースはDomain層で定義し、Infrastructure層で実装する（DIP）
// - 複数の実装を切り替え可能（PDFParser, ExcelParser, AIParser など）
// - Parse処理は外部システムとの境界で行われるため、Anti-Corruption Layerとして機能
type TariffParser interface {
	// Parse: ファイルから料金表データを解析
	// 返り値の ParsedTariffData はドメインモデルではなく、中間データ構造
	Parse(reader io.Reader) (*ParsedTariffData, error)

	// SupportedFormats: このパーサーがサポートするファイル形式
	// 例: ["pdf", "xlsx", "csv"]
	SupportedFormats() []string
}

// TariffParserFactory: ファイル形式に応じた適切なパーサーを返すファクトリー
//
// 設計上の注意:
// - Factory Patternによる実装の切り替え
// - Infrastructure層で具象Factoryを実装
type TariffParserFactory interface {
	// GetParser: ファイル形式に応じたパーサーを取得
	// format例: "pdf", "xlsx", "csv", "ai"
	GetParser(format string) (TariffParser, error)
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
//
// 設計上の注意:
// - ドメインルールに基づくバリデーションを行う
// - Infrastructure層で実装（ファイル形式特有のバリデーション）
// - または Application層で実装（ドメインルール適用）
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
