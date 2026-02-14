package document

import (
	"context"
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ExtractDocumentContentInput: 書類コンテンツ抽出の入力
type ExtractDocumentContentInput struct {
	DocumentID uuid.UUID
}

// ExtractDocumentContentOutput: 書類コンテンツ抽出の出力
type ExtractDocumentContentOutput struct {
	DocumentID  uuid.UUID
	DocType     shared.DocType
	ExtractedAt time.Time
}

// ExtractDocumentContentUseCase: 書類コンテンツ抽出ユースケース
// DRAFT状態の書類に対してDocumentContentExtractorを実行し、構造化コンテンツを設定する
type ExtractDocumentContentUseCase struct {
	documentRepo domain.DocumentRepository
	extractor    domain.DocumentContentExtractor
}

// NewExtractDocumentContentUseCase: コンストラクタ
func NewExtractDocumentContentUseCase(
	documentRepo domain.DocumentRepository,
	extractor domain.DocumentContentExtractor,
) *ExtractDocumentContentUseCase {
	return &ExtractDocumentContentUseCase{
		documentRepo: documentRepo,
		extractor:    extractor,
	}
}

// Execute: ユースケースを実行
func (uc *ExtractDocumentContentUseCase) Execute(
	ctx context.Context,
	input ExtractDocumentContentInput,
) (*ExtractDocumentContentOutput, error) {
	// 1. Documentの取得
	doc, err := uc.documentRepo.FindByID(ctx, input.DocumentID)
	if err != nil {
		return nil, NewDocumentUseCaseError("NOT_FOUND", "document not found").
			WithDetail("documentID", input.DocumentID)
	}

	// 2. ステータス検証（DRAFTのみ許可）
	if doc.Status() != domain.DocumentStatusDraft {
		return nil, NewDocumentUseCaseError("INVALID_STATE", "only DRAFT documents can have content extracted").
			WithDetail("documentID", doc.ID).
			WithDetail("currentStatus", string(doc.Status()))
	}

	// 3. コンテンツ抽出
	content, err := uc.extractor.Extract(ctx, doc.StorageURI, doc.DocType)
	if err != nil {
		return nil, NewDocumentUseCaseError("EXTRACTION_ERROR", "failed to extract document content").
			WithDetail("documentID", doc.ID).
			WithDetail("storageURI", doc.StorageURI)
	}

	// 4. ドメインオブジェクトにコンテンツを設定
	if err := doc.SetContent(content); err != nil {
		return nil, NewDocumentUseCaseError("INVALID_CONTENT", err.Error()).
			WithDetail("documentID", doc.ID)
	}

	// 5. イベント記録
	doc.RecordEvent(domain.NewDocumentContentExtracted(doc.ID, doc.ShipmentID, doc.DocType))

	// 6. 永続化
	if err := uc.documentRepo.Save(ctx, doc); err != nil {
		return nil, NewDocumentUseCaseError("SAVE_ERROR", "failed to save document").
			WithDetail("documentID", doc.ID)
	}

	// 7. 出力DTOの作成
	return &ExtractDocumentContentOutput{
		DocumentID:  doc.ID,
		DocType:     doc.DocType,
		ExtractedAt: doc.UpdatedAt,
	}, nil
}
