package document

import (
	"context"

	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// DocumentContentExtractor: 書類ファイルから構造化コンテンツを抽出するインターフェース
//
// Domain層で定義し、Infrastructure層で実装する（DIP）
// 実装例: AI OCR、ルールベースパーサー等
//
// 設計上の注意:
// - TariffParser（sourcing/domain/pricing）と同様のACLパターン
// - ファイルは既にストレージに保存済みのため、storageURIを受け取る
// - Infrastructure実装がストレージアクセス + AI OCR呼び出しを担う
// - docTypeに応じた抽出ロジックの切り替えを可能にする
type DocumentContentExtractor interface {
	// Extract: ストレージ上のファイルから構造化コンテンツを抽出
	Extract(ctx context.Context, storageURI string, docType shared.DocType) (DocumentContent, error)
}
