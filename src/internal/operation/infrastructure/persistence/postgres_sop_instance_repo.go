package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/operation/domain/sop"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// PostgresSOPInstanceRepo: SOPInstance集約のPostgreSQL実装
type PostgresSOPInstanceRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSOPInstanceRepo(pool *pgxpool.Pool) *PostgresSOPInstanceRepo {
	return &PostgresSOPInstanceRepo{pool: pool}
}

func (r *PostgresSOPInstanceRepo) Save(ctx context.Context, inst *sop.SOPInstance) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO sop_instances (id, shipment_id, definition_id, definition_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status, updated_at = EXCLUDED.updated_at`,
		inst.ID, inst.ShipmentID, inst.DefinitionID, inst.DefinitionName,
		string(inst.Status()), inst.CreatedAt, inst.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert instance: %w", err)
	}

	// Upsert tasks (update existing or insert new)
	for _, task := range inst.Tasks() {
		reqDocTypesJSON, err := json.Marshal(task.RequiredDocTypes)
		if err != nil {
			return fmt.Errorf("marshal required doc types: %w", err)
		}

		linkedDocIDsJSON, err := json.Marshal(task.LinkedDocumentIDs)
		if err != nil {
			return fmt.Errorf("marshal linked doc ids: %w", err)
		}

		var genDocType *string
		if task.GeneratedDocType != nil {
			s := string(*task.GeneratedDocType)
			genDocType = &s
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO sop_tasks (
				id, sop_instance_id, step_definition_id, name, description,
				order_index, action_type, required_doc_types, generated_doc_type,
				status, assignee_id, linked_document_ids, completed_at, completed_by, note
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT (id) DO UPDATE SET
				status = EXCLUDED.status,
				assignee_id = EXCLUDED.assignee_id,
				linked_document_ids = EXCLUDED.linked_document_ids,
				completed_at = EXCLUDED.completed_at,
				completed_by = EXCLUDED.completed_by,
				note = EXCLUDED.note`,
			task.ID, inst.ID, task.StepDefinitionID, task.Name, task.Description,
			task.OrderIndex, string(task.ActionType), reqDocTypesJSON, genDocType,
			string(task.Status()), task.AssigneeID, linkedDocIDsJSON,
			task.CompletedAt, task.CompletedBy, task.Note,
		)
		if err != nil {
			return fmt.Errorf("upsert task: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresSOPInstanceRepo) FindByID(ctx context.Context, id uuid.UUID) (*sop.SOPInstance, error) {
	var (
		instID, shipmentID, defID uuid.UUID
		defName, status           string
		createdAt, updatedAt      time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, shipment_id, definition_id, definition_name, status, created_at, updated_at
		FROM sop_instances WHERE id = $1`, id).
		Scan(&instID, &shipmentID, &defID, &defName, &status, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "SOP instance not found")
		}
		return nil, err
	}

	tasks, err := r.loadTasks(ctx, instID)
	if err != nil {
		return nil, err
	}

	return sop.ReconstructSOPInstance(
		instID, shipmentID, defID, defName,
		tasks, sop.SOPInstanceStatus(status),
		createdAt, updatedAt,
	), nil
}

func (r *PostgresSOPInstanceRepo) FindByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*sop.SOPInstance, error) {
	var instID uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM sop_instances WHERE shipment_id = $1`, shipmentID).Scan(&instID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "SOP instance not found for shipment")
		}
		return nil, err
	}
	return r.FindByID(ctx, instID)
}

func (r *PostgresSOPInstanceRepo) loadTasks(ctx context.Context, instanceID uuid.UUID) ([]sop.SOPTask, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, step_definition_id, name, description,
			order_index, action_type, required_doc_types, generated_doc_type,
			status, assignee_id, linked_document_ids, completed_at, completed_by, note
		FROM sop_tasks WHERE sop_instance_id = $1 ORDER BY order_index`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []sop.SOPTask
	for rows.Next() {
		var (
			id, stepDefID          uuid.UUID
			name, description      string
			orderIndex             int
			actionType, status     string
			reqDocJSON             []byte
			genDocType             *string
			assigneeID             *uuid.UUID
			linkedDocIDsJSON       []byte
			completedAt            *time.Time
			completedBy            *uuid.UUID
			note                   string
		)
		err := rows.Scan(
			&id, &stepDefID, &name, &description,
			&orderIndex, &actionType, &reqDocJSON, &genDocType,
			&status, &assigneeID, &linkedDocIDsJSON, &completedAt, &completedBy, &note,
		)
		if err != nil {
			return nil, err
		}

		var reqDocTypes []shared.DocType
		if reqDocJSON != nil {
			if err := json.Unmarshal(reqDocJSON, &reqDocTypes); err != nil {
				return nil, fmt.Errorf("unmarshal required doc types: %w", err)
			}
		}

		var linkedDocIDs []uuid.UUID
		if linkedDocIDsJSON != nil {
			if err := json.Unmarshal(linkedDocIDsJSON, &linkedDocIDs); err != nil {
				return nil, fmt.Errorf("unmarshal linked doc ids: %w", err)
			}
		}

		var genDocTypePtr *shared.DocType
		if genDocType != nil {
			dt := shared.DocType(*genDocType)
			genDocTypePtr = &dt
		}

		tasks = append(tasks, sop.ReconstructSOPTask(
			id, stepDefID, name, description, orderIndex,
			sop.ActionType(actionType), reqDocTypes, genDocTypePtr,
			sop.TaskStatus(status), assigneeID, linkedDocIDs,
			completedAt, completedBy, note,
		))
	}
	return tasks, rows.Err()
}
