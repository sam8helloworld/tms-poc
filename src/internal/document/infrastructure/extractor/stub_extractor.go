package extractor

import (
	"context"
	"fmt"

	"github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// StubDocumentContentExtractor: 開発・テスト用のDocumentContentExtractorスタブ
type StubDocumentContentExtractor struct{}

// NewStubDocumentContentExtractor: StubDocumentContentExtractorのコンストラクタ
func NewStubDocumentContentExtractor() *StubDocumentContentExtractor {
	return &StubDocumentContentExtractor{}
}

// Extract: スタブ実装 - GenericContentを返す
func (e *StubDocumentContentExtractor) Extract(
	ctx context.Context,
	storageURI string,
	docType shared.DocType,
) (document.DocumentContent, error) {
	_ = ctx
	fmt.Printf("[STUB] Extract called for storageURI=%s, docType=%s (returning GenericContent)\n", storageURI, docType)
	return &document.GenericContent{
		DocTypeValue: docType,
		Fields: map[string]interface{}{
			"stub":       true,
			"storageURI": storageURI,
		},
	}, nil
}

// Verify interface compliance
var _ document.DocumentContentExtractor = (*StubDocumentContentExtractor)(nil)
