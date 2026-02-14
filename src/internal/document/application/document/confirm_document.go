package document

import (
	"context"
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ConfirmDocumentInput: 書類確認完了の入力
type ConfirmDocumentInput struct {
	DocumentID uuid.UUID
}

// ConfirmDocumentOutput: 書類確認完了の出力
type ConfirmDocumentOutput struct {
	DocumentID  uuid.UUID
	ShipmentID  uuid.UUID
	DocType     shared.DocType
	FileName    string
	Status      string
	ConfirmedAt time.Time
}

// ConfirmDocumentUseCase: 書類確認完了ユースケース
// DRAFT状態の書類をCONFIRMED状態に遷移させる
type ConfirmDocumentUseCase struct {
	documentRepo domain.DocumentRepository
}

// NewConfirmDocumentUseCase: コンストラクタ
func NewConfirmDocumentUseCase(
	documentRepo domain.DocumentRepository,
) *ConfirmDocumentUseCase {
	return &ConfirmDocumentUseCase{
		documentRepo: documentRepo,
	}
}

// Execute: ユースケースを実行
func (uc *ConfirmDocumentUseCase) Execute(
	ctx context.Context,
	input ConfirmDocumentInput,
) (*ConfirmDocumentOutput, error) {
	// 1. Documentの取得
	doc, err := uc.documentRepo.FindByID(ctx, input.DocumentID)
	if err != nil {
		return nil, NewDocumentUseCaseError("NOT_FOUND", "document not found").
			WithDetail("documentID", input.DocumentID)
	}

	// 2. DRAFT → CONFIRMED への遷移
	if err := doc.Confirm(); err != nil {
		return nil, NewDocumentUseCaseError("INVALID_STATE", err.Error()).
			WithDetail("documentID", doc.ID).
			WithDetail("currentStatus", string(doc.Status()))
	}

	// 3. 永続化
	if err := uc.documentRepo.Save(ctx, doc); err != nil {
		return nil, NewDocumentUseCaseError("SAVE_ERROR", "failed to save document").
			WithDetail("documentID", doc.ID)
	}

	// 4. 出力DTOの作成
	return &ConfirmDocumentOutput{
		DocumentID:  doc.ID,
		ShipmentID:  doc.ShipmentID,
		DocType:     doc.DocType,
		FileName:    doc.FileName,
		Status:      string(doc.Status()),
		ConfirmedAt: doc.UpdatedAt,
	}, nil
}
