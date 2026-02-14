package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// PostgresDocumentRepo: Document集約のPostgreSQL実装
type PostgresDocumentRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresDocumentRepo(pool *pgxpool.Pool) *PostgresDocumentRepo {
	return &PostgresDocumentRepo{pool: pool}
}

func (r *PostgresDocumentRepo) Save(ctx context.Context, doc *document.Document) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var contentJSON []byte
	if doc.Content() != nil {
		contentJSON, err = json.Marshal(doc.Content())
		if err != nil {
			return fmt.Errorf("marshal content: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO documents (
			id, shipment_id, doc_type, origin, file_name, mime_type,
			storage_uri, file_size, uploaded_by, status, version,
			metadata, content, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, version = EXCLUDED.version,
			metadata = EXCLUDED.metadata, content = EXCLUDED.content,
			updated_at = EXCLUDED.updated_at`,
		doc.ID, doc.ShipmentID, string(doc.DocType), string(doc.Origin),
		doc.FileName, doc.MimeType, doc.StorageURI, doc.FileSize,
		doc.UploadedBy, string(doc.Status()), doc.Version,
		metadataJSON, contentJSON,
		doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert document: %w", err)
	}

	// Insert new reviews (append-only)
	for _, review := range doc.Reviews() {
		discrepanciesJSON, err := json.Marshal(review.Discrepancies)
		if err != nil {
			return fmt.Errorf("marshal discrepancies: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO document_reviews (id, document_id, reviewer_id, reviewed_at, decision, comment, discrepancies)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING`,
			review.ID, doc.ID, review.ReviewerID, review.ReviewedAt,
			string(review.Decision), review.Comment, discrepanciesJSON,
		)
		if err != nil {
			return fmt.Errorf("insert review: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresDocumentRepo) FindByID(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	doc, err := r.scanDocument(ctx, `
		SELECT id, shipment_id, doc_type, origin, file_name, mime_type,
			storage_uri, file_size, uploaded_by, status, version,
			metadata, content, created_at, updated_at
		FROM documents WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	reviews, err := r.loadReviews(ctx, doc.ID)
	if err != nil {
		return nil, err
	}

	return document.ReconstructDocument(
		doc.ID, doc.ShipmentID, doc.DocType, doc.Origin,
		doc.FileName, doc.MimeType, doc.StorageURI, doc.FileSize,
		doc.UploadedBy, doc.Status(), doc.Version, doc.Metadata,
		doc.Content(), reviews,
		doc.CreatedAt, doc.UpdatedAt,
	), nil
}

func (r *PostgresDocumentRepo) FindByShipmentID(ctx context.Context, shipmentID uuid.UUID) ([]*document.Document, error) {
	return r.queryDocuments(ctx, `
		SELECT id, shipment_id, doc_type, origin, file_name, mime_type,
			storage_uri, file_size, uploaded_by, status, version,
			metadata, content, created_at, updated_at
		FROM documents WHERE shipment_id = $1 ORDER BY created_at`, shipmentID)
}

func (r *PostgresDocumentRepo) FindByShipmentIDAndDocType(ctx context.Context, shipmentID uuid.UUID, docType shared.DocType) ([]*document.Document, error) {
	return r.queryDocuments(ctx, `
		SELECT id, shipment_id, doc_type, origin, file_name, mime_type,
			storage_uri, file_size, uploaded_by, status, version,
			metadata, content, created_at, updated_at
		FROM documents WHERE shipment_id = $1 AND doc_type = $2 ORDER BY version DESC`,
		shipmentID, string(docType))
}

// scanDocument scans a single document row (without reviews)
func (r *PostgresDocumentRepo) scanDocument(ctx context.Context, query string, args ...any) (*document.Document, error) {
	var (
		id, shipmentID, uploadedBy         uuid.UUID
		docType, origin, status            string
		fileName, mimeType, storageURI     string
		fileSize                           int64
		version                            int
		metadataJSON                       []byte
		contentJSON                        []byte
		createdAt, updatedAt               time.Time
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id, &shipmentID, &docType, &origin, &fileName, &mimeType,
		&storageURI, &fileSize, &uploadedBy, &status, &version,
		&metadataJSON, &contentJSON, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "document not found")
		}
		return nil, err
	}

	metadata := make(map[string]string)
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	// Note: content deserialization requires polymorphic mapping based on doc_type
	// For now, content remains nil unless explicitly deserialized
	_ = contentJSON

	return document.ReconstructDocument(
		id, shipmentID, shared.DocType(docType), document.DocumentOrigin(origin),
		fileName, mimeType, storageURI, fileSize,
		uploadedBy, document.DocumentStatus(status), version, metadata,
		nil, nil,
		createdAt, updatedAt,
	), nil
}

func (r *PostgresDocumentRepo) queryDocuments(ctx context.Context, query string, args ...any) ([]*document.Document, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*document.Document
	for rows.Next() {
		var (
			id, shipmentID, uploadedBy         uuid.UUID
			docType, origin, status            string
			fileName, mimeType, storageURI     string
			fileSize                           int64
			version                            int
			metadataJSON                       []byte
			contentJSON                        []byte
			createdAt, updatedAt               time.Time
		)

		err := rows.Scan(
			&id, &shipmentID, &docType, &origin, &fileName, &mimeType,
			&storageURI, &fileSize, &uploadedBy, &status, &version,
			&metadataJSON, &contentJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		metadata := make(map[string]string)
		if metadataJSON != nil {
			_ = json.Unmarshal(metadataJSON, &metadata)
		}

		reviews, err := r.loadReviews(ctx, id)
		if err != nil {
			return nil, err
		}

		_ = contentJSON

		doc := document.ReconstructDocument(
			id, shipmentID, shared.DocType(docType), document.DocumentOrigin(origin),
			fileName, mimeType, storageURI, fileSize,
			uploadedBy, document.DocumentStatus(status), version, metadata,
			nil, reviews,
			createdAt, updatedAt,
		)
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (r *PostgresDocumentRepo) loadReviews(ctx context.Context, documentID uuid.UUID) ([]document.DocumentReview, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, reviewer_id, reviewed_at, decision, comment, discrepancies
		FROM document_reviews WHERE document_id = $1 ORDER BY reviewed_at`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []document.DocumentReview
	for rows.Next() {
		var (
			review            document.DocumentReview
			decision          string
			discrepanciesJSON []byte
		)
		err := rows.Scan(
			&review.ID, &review.ReviewerID, &review.ReviewedAt,
			&decision, &review.Comment, &discrepanciesJSON,
		)
		if err != nil {
			return nil, err
		}
		review.Decision = document.ReviewDecision(decision)

		if discrepanciesJSON != nil {
			if err := json.Unmarshal(discrepanciesJSON, &review.Discrepancies); err != nil {
				return nil, fmt.Errorf("unmarshal discrepancies: %w", err)
			}
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}
