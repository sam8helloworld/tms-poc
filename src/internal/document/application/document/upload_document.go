package document

import (
	"context"
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// UploadDocumentInput: 書類アップロードの入力
type UploadDocumentInput struct {
	ShipmentID uuid.UUID
	DocType    shared.DocType
	Origin     domain.DocumentOrigin // SHIPPER or PROVIDER
	FileName   string
	MimeType   string
	StorageURI string
	FileSize   int64
	UploadedBy uuid.UUID
	Metadata   map[string]string        // 任意のメタデータ（後方互換）
	Content    domain.DocumentContent    // 構造化コンテンツ（任意）
}

// UploadDocumentOutput: 書類アップロードの出力
type UploadDocumentOutput struct {
	DocumentID uuid.UUID
	ShipmentID uuid.UUID
	DocType    shared.DocType
	FileName   string
	Status     string
	Version    int
	CreatedAt  time.Time
}

// UploadDocumentUseCase: 書類アップロードユースケース
// 書類をDRAFT状態で新規作成し、永続化する
type UploadDocumentUseCase struct {
	documentRepo domain.DocumentRepository
}

// NewUploadDocumentUseCase: コンストラクタ
func NewUploadDocumentUseCase(
	documentRepo domain.DocumentRepository,
) *UploadDocumentUseCase {
	return &UploadDocumentUseCase{
		documentRepo: documentRepo,
	}
}

// Execute: ユースケースを実行
func (uc *UploadDocumentUseCase) Execute(
	ctx context.Context,
	input UploadDocumentInput,
) (*UploadDocumentOutput, error) {
	// 1. Documentドメインオブジェクトを生成（DRAFT状態）
	doc, err := domain.NewDocument(
		input.ShipmentID,
		input.DocType,
		input.Origin,
		input.FileName,
		input.MimeType,
		input.StorageURI,
		input.FileSize,
		input.UploadedBy,
	)
	if err != nil {
		return nil, NewDocumentUseCaseError("INVALID_INPUT", err.Error())
	}

	// 2. 構造化コンテンツの設定（任意）
	if input.Content != nil {
		if err := doc.SetContent(input.Content); err != nil {
			return nil, NewDocumentUseCaseError("INVALID_CONTENT", err.Error())
		}
	}

	// 3. メタデータの設定
	for key, value := range input.Metadata {
		doc.UpdateMetadata(key, value)
	}

	// 4. 永続化
	if err := uc.documentRepo.Save(ctx, doc); err != nil {
		return nil, NewDocumentUseCaseError("SAVE_ERROR", "failed to save document").
			WithDetail("documentID", doc.ID)
	}

	// 5. 出力DTOの作成
	return &UploadDocumentOutput{
		DocumentID: doc.ID,
		ShipmentID: doc.ShipmentID,
		DocType:    doc.DocType,
		FileName:   doc.FileName,
		Status:     string(doc.Status()),
		Version:    doc.Version,
		CreatedAt:  doc.CreatedAt,
	}, nil
}
