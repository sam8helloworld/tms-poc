CREATE TYPE document_status AS ENUM (
    'DRAFT', 'ISSUED', 'UNDER_REVIEW', 'REVISION_REQUESTED', 'CONFIRMED', 'ARCHIVED'
);

CREATE TYPE document_origin AS ENUM ('SHIPPER', 'PROVIDER');

-- Documents (aggregate root)
CREATE TABLE documents (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id  UUID NOT NULL,
    doc_type     VARCHAR(50) NOT NULL,
    origin       document_origin NOT NULL,
    file_name    VARCHAR(500) NOT NULL,
    mime_type    VARCHAR(100),
    storage_uri  TEXT NOT NULL,
    file_size    BIGINT NOT NULL,
    uploaded_by  UUID NOT NULL,
    status       document_status NOT NULL DEFAULT 'DRAFT',
    version      INT NOT NULL DEFAULT 1,
    metadata     JSONB NOT NULL DEFAULT '{}'::JSONB,
    content      JSONB,             -- polymorphic DocumentContent (nullable until extracted)
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_shipment ON documents(shipment_id);
CREATE INDEX idx_documents_shipment_type ON documents(shipment_id, doc_type);
CREATE INDEX idx_documents_status ON documents(status);

-- Document Reviews (part of aggregate, append-only)
CREATE TABLE document_reviews (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id   UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    reviewer_id   UUID NOT NULL,
    reviewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decision      VARCHAR(30) NOT NULL,   -- APPROVED, REJECTED, REVISION_REQUESTED
    comment       TEXT,
    discrepancies JSONB NOT NULL DEFAULT '[]'::JSONB
);

CREATE INDEX idx_doc_reviews_document ON document_reviews(document_id);
