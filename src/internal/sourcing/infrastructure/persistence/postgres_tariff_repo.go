package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// PostgresTariffRepo: Tariff集約のPostgreSQL実装
type PostgresTariffRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTariffRepo(pool *pgxpool.Pool) *PostgresTariffRepo {
	return &PostgresTariffRepo{pool: pool}
}

func (r *PostgresTariffRepo) Save(ctx context.Context, t *pricing.Tariff) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Upsert tariff header
	_, err = tx.Exec(ctx, `
		INSERT INTO tariffs (id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, version = EXCLUDED.version,
			base_version_id = EXCLUDED.base_version_id,
			effective_from = EXCLUDED.effective_from,
			effective_to = EXCLUDED.effective_to,
			updated_at = EXCLUDED.updated_at`,
		t.ID, t.ContractID, t.Name, t.Version, t.BaseVersionID,
		t.EffectiveDate.From, t.EffectiveDate.To,
		t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert tariff: %w", err)
	}

	// Replace line items
	_, err = tx.Exec(ctx, `DELETE FROM tariff_line_items WHERE tariff_id = $1`, t.ID)
	if err != nil {
		return fmt.Errorf("delete line items: %w", err)
	}

	for _, item := range t.LineItems {
		scopeJSON, err := json.Marshal(item.Scope)
		if err != nil {
			return fmt.Errorf("marshal scope: %w", err)
		}
		logicJSON, err := json.Marshal(item.Logic)
		if err != nil {
			return fmt.Errorf("marshal logic: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO tariff_line_items (id, tariff_id, charge_code, category, scope, logic, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			item.ID, t.ID, item.ChargeCode, item.Category,
			scopeJSON, logicJSON, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("insert line item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresTariffRepo) FindByID(ctx context.Context, id uuid.UUID) (*pricing.Tariff, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at
		FROM tariffs WHERE id = $1`, id)

	t, err := scanTariffHeader(row)
	if err != nil {
		return nil, err
	}

	lineItems, err := r.loadLineItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.LineItems = lineItems

	return t, nil
}

func (r *PostgresTariffRepo) FindByContractID(ctx context.Context, contractID uuid.UUID) ([]*pricing.Tariff, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at
		FROM tariffs WHERE contract_id = $1 ORDER BY name, version`, contractID)
	if err != nil {
		return nil, err
	}
	return r.collectTariffsWithLineItems(ctx, rows)
}

func (r *PostgresTariffRepo) FindByContractIDAndName(ctx context.Context, contractID uuid.UUID, name string) ([]*pricing.Tariff, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at
		FROM tariffs WHERE contract_id = $1 AND name = $2 ORDER BY version`, contractID, name)
	if err != nil {
		return nil, err
	}
	return r.collectTariffsWithLineItems(ctx, rows)
}

func (r *PostgresTariffRepo) FindLatestVersionByContractIDAndName(ctx context.Context, contractID uuid.UUID, name string) (*pricing.Tariff, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, contract_id, name, version, base_version_id, effective_from, effective_to, created_at, updated_at
		FROM tariffs WHERE contract_id = $1 AND name = $2 ORDER BY version DESC LIMIT 1`, contractID, name)

	t, err := scanTariffHeader(row)
	if err != nil {
		return nil, err
	}

	lineItems, err := r.loadLineItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.LineItems = lineItems
	return t, nil
}

func (r *PostgresTariffRepo) CountByContractID(ctx context.Context, contractID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tariffs WHERE contract_id = $1`, contractID).Scan(&count)
	return count, err
}

func (r *PostgresTariffRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tariffs WHERE id = $1`, id)
	return err
}

func scanTariffHeader(s scannable) (*pricing.Tariff, error) {
	var t pricing.Tariff
	var baseVersionID *uuid.UUID
	var effectiveFrom, effectiveTo time.Time

	err := s.Scan(
		&t.ID, &t.ContractID, &t.Name, &t.Version, &baseVersionID,
		&effectiveFrom, &effectiveTo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "tariff not found")
		}
		return nil, err
	}

	t.BaseVersionID = baseVersionID
	t.EffectiveDate = shared.DateRange{From: effectiveFrom, To: effectiveTo}
	t.LineItems = make([]pricing.TariffLineItem, 0)

	return &t, nil
}

func (r *PostgresTariffRepo) loadLineItems(ctx context.Context, tariffID uuid.UUID) ([]pricing.TariffLineItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, charge_code, category, scope, logic
		FROM tariff_line_items WHERE tariff_id = $1`, tariffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []pricing.TariffLineItem
	for rows.Next() {
		var item pricing.TariffLineItem
		var scopeJSON, logicJSON []byte
		err := rows.Scan(&item.ID, &item.ChargeCode, &item.Category, &scopeJSON, &logicJSON)
		if err != nil {
			return nil, err
		}
		// Note: Scope and Logic deserialization requires polymorphic mapping
		// This will be implemented in a separate mapper when concrete types are known
		_ = scopeJSON
		_ = logicJSON
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresTariffRepo) collectTariffsWithLineItems(ctx context.Context, rows pgx.Rows) ([]*pricing.Tariff, error) {
	defer rows.Close()
	var tariffs []*pricing.Tariff
	for rows.Next() {
		t, err := scanTariffHeader(rows)
		if err != nil {
			return nil, err
		}
		tariffs = append(tariffs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load line items for each tariff
	for _, t := range tariffs {
		lineItems, err := r.loadLineItems(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		t.LineItems = lineItems
	}
	return tariffs, nil
}
