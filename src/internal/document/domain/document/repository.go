package document

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// DocumentRepository: Document集約のリポジトリインターフェース
type DocumentRepository interface {
	// Save: Documentを保存（新規作成または更新）
	Save(ctx context.Context, doc *Document) error

	// FindByID: IDでDocumentを取得
	FindByID(ctx context.Context, id uuid.UUID) (*Document, error)

	// FindByShipmentID: ShipmentIDで関連Documentを全件取得
	FindByShipmentID(ctx context.Context, shipmentID uuid.UUID) ([]*Document, error)

	// FindByShipmentIDAndDocType: ShipmentIDと書類種別でDocumentを検索
	FindByShipmentIDAndDocType(ctx context.Context, shipmentID uuid.UUID, docType shared.DocType) ([]*Document, error)
}
