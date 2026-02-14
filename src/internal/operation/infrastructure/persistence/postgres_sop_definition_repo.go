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

// PostgresSOPDefinitionRepo: SOPDefinition集約のPostgreSQL実装
type PostgresSOPDefinitionRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSOPDefinitionRepo(pool *pgxpool.Pool) *PostgresSOPDefinitionRepo {
	return &PostgresSOPDefinitionRepo{pool: pool}
}

func (r *PostgresSOPDefinitionRepo) Save(ctx context.Context, def *sop.SOPDefinition) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO sop_definitions (
			id, name, description, direction, transport_mode,
			origin_country, dest_country, status, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description,
			status = EXCLUDED.status, version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at`,
		def.ID, def.Name, def.Description,
		string(def.TargetScope.Direction), string(def.TargetScope.TransportMode),
		def.TargetScope.OriginCountryCode, def.TargetScope.DestCountryCode,
		string(def.Status()), def.Version,
		def.CreatedAt, def.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert definition: %w", err)
	}

	// Replace steps
	_, err = tx.Exec(ctx, `DELETE FROM sop_step_definitions WHERE sop_definition_id = $1`, def.ID)
	if err != nil {
		return fmt.Errorf("delete steps: %w", err)
	}

	for _, step := range def.Steps() {
		reqDocTypesJSON, err := json.Marshal(step.RequiredDocTypes)
		if err != nil {
			return fmt.Errorf("marshal required doc types: %w", err)
		}

		var genDocType *string
		if step.GeneratedDocType != nil {
			s := string(*step.GeneratedDocType)
			genDocType = &s
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO sop_step_definitions (
				id, sop_definition_id, name, description, order_index,
				required_doc_types, generated_doc_type, action_type, is_automatable
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			step.ID, def.ID, step.Name, step.Description, step.OrderIndex,
			reqDocTypesJSON, genDocType, string(step.ActionType), step.IsAutomatable,
		)
		if err != nil {
			return fmt.Errorf("insert step: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresSOPDefinitionRepo) FindByID(ctx context.Context, id uuid.UUID) (*sop.SOPDefinition, error) {
	var (
		defID                              uuid.UUID
		name, description, direction, mode string
		status                             string
		originCountry, destCountry         *string
		version                            int
		createdAt, updatedAt               time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, direction, transport_mode,
			origin_country, dest_country, status, version, created_at, updated_at
		FROM sop_definitions WHERE id = $1`, id).
		Scan(&defID, &name, &description, &direction, &mode,
			&originCountry, &destCountry, &status, &version, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "SOP definition not found")
		}
		return nil, err
	}

	steps, err := r.loadSteps(ctx, defID)
	if err != nil {
		return nil, err
	}

	return sop.ReconstructSOPDefinition(
		defID, name, description,
		sop.ScopeCriteria{
			Direction:         shared.TradeDirection(direction),
			TransportMode:     shared.TransportMode(mode),
			OriginCountryCode: originCountry,
			DestCountryCode:   destCountry,
		},
		steps, sop.SOPDefinitionStatus(status),
		version, createdAt, updatedAt,
	), nil
}

func (r *PostgresSOPDefinitionRepo) FindActiveByScope(
	ctx context.Context,
	direction shared.TradeDirection,
	mode shared.TransportMode,
	originCountry, destCountry *string,
) ([]*sop.SOPDefinition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, direction, transport_mode,
			origin_country, dest_country, status, version, created_at, updated_at
		FROM sop_definitions
		WHERE status = 'ACTIVE'
			AND direction = $1
			AND transport_mode = $2
			AND (origin_country IS NULL OR origin_country = $3)
			AND (dest_country IS NULL OR dest_country = $4)
		ORDER BY name`,
		string(direction), string(mode), originCountry, destCountry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []*sop.SOPDefinition
	for rows.Next() {
		var (
			defID                              uuid.UUID
			name, description, dir, m, st      string
			origC, destC                       *string
			version                            int
			createdAt, updatedAt               time.Time
		)
		err := rows.Scan(&defID, &name, &description, &dir, &m,
			&origC, &destC, &st, &version, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		steps, err := r.loadSteps(ctx, defID)
		if err != nil {
			return nil, err
		}

		defs = append(defs, sop.ReconstructSOPDefinition(
			defID, name, description,
			sop.ScopeCriteria{
				Direction:         shared.TradeDirection(dir),
				TransportMode:     shared.TransportMode(m),
				OriginCountryCode: origC,
				DestCountryCode:   destC,
			},
			steps, sop.SOPDefinitionStatus(st),
			version, createdAt, updatedAt,
		))
	}
	return defs, rows.Err()
}

func (r *PostgresSOPDefinitionRepo) loadSteps(ctx context.Context, defID uuid.UUID) ([]sop.SOPStepDefinition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, order_index,
			required_doc_types, generated_doc_type, action_type, is_automatable
		FROM sop_step_definitions WHERE sop_definition_id = $1 ORDER BY order_index`, defID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []sop.SOPStepDefinition
	for rows.Next() {
		var (
			step         sop.SOPStepDefinition
			reqDocJSON   []byte
			genDocType   *string
			actionType   string
		)
		err := rows.Scan(
			&step.ID, &step.Name, &step.Description, &step.OrderIndex,
			&reqDocJSON, &genDocType, &actionType, &step.IsAutomatable,
		)
		if err != nil {
			return nil, err
		}

		step.ActionType = sop.ActionType(actionType)
		if genDocType != nil {
			dt := shared.DocType(*genDocType)
			step.GeneratedDocType = &dt
		}
		if reqDocJSON != nil {
			if err := json.Unmarshal(reqDocJSON, &step.RequiredDocTypes); err != nil {
				return nil, fmt.Errorf("unmarshal required doc types: %w", err)
			}
		}

		steps = append(steps, step)
	}
	return steps, rows.Err()
}
