CREATE TYPE sop_definition_status AS ENUM ('DRAFT', 'ACTIVE', 'ARCHIVED');
CREATE TYPE sop_instance_status AS ENUM ('IN_PROGRESS', 'COMPLETED', 'CANCELLED');
CREATE TYPE task_status AS ENUM ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'SKIPPED', 'ERROR');
CREATE TYPE action_type AS ENUM (
    'DOCUMENT_UPLOAD', 'DOCUMENT_GENERATION', 'EMAIL_SEND',
    'APPROVAL_REQUEST', 'EXTERNAL_API_CALL', 'MANUAL_CHECK'
);

-- SOP Definitions (aggregate root / template)
CREATE TABLE sop_definitions (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    direction     VARCHAR(10) NOT NULL,   -- EXPORT / IMPORT
    transport_mode transport_mode NOT NULL,
    origin_country VARCHAR(2),
    dest_country   VARCHAR(2),
    status        sop_definition_status NOT NULL DEFAULT 'DRAFT',
    version       INT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sop_defs_status ON sop_definitions(status);

-- SOP Step Definitions (part of SOPDefinition aggregate)
CREATE TABLE sop_step_definitions (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sop_definition_id  UUID NOT NULL REFERENCES sop_definitions(id) ON DELETE CASCADE,
    name               VARCHAR(255) NOT NULL,
    description        TEXT,
    order_index        INT NOT NULL,
    required_doc_types JSONB NOT NULL DEFAULT '[]'::JSONB,  -- array of DocType strings
    generated_doc_type VARCHAR(50),
    action_type        action_type NOT NULL,
    is_automatable     BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT uq_sop_step_order UNIQUE (sop_definition_id, order_index)
);

CREATE INDEX idx_sop_steps_def ON sop_step_definitions(sop_definition_id);

-- SOP Instances (aggregate root / execution)
CREATE TABLE sop_instances (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id     UUID NOT NULL,
    definition_id   UUID NOT NULL,
    definition_name VARCHAR(255) NOT NULL,
    status          sop_instance_status NOT NULL DEFAULT 'IN_PROGRESS',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sop_instances_shipment ON sop_instances(shipment_id);

-- SOP Tasks (part of SOPInstance aggregate)
CREATE TABLE sop_tasks (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sop_instance_id     UUID NOT NULL REFERENCES sop_instances(id) ON DELETE CASCADE,
    step_definition_id  UUID NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    order_index         INT NOT NULL,
    action_type         action_type NOT NULL,
    required_doc_types  JSONB NOT NULL DEFAULT '[]'::JSONB,
    generated_doc_type  VARCHAR(50),
    status              task_status NOT NULL DEFAULT 'PENDING',
    assignee_id         UUID,
    linked_document_ids JSONB NOT NULL DEFAULT '[]'::JSONB,
    completed_at        TIMESTAMPTZ,
    completed_by        UUID,
    note                TEXT
);

CREATE INDEX idx_sop_tasks_instance ON sop_tasks(sop_instance_id);
