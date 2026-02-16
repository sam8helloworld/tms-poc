package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/shopspring/decimal"
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
		item.Scope = deserializeScope(scopeJSON)
		item.Logic = deserializeLogic(logicJSON)
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

// deserializeScope: JSONB からServiceScopeを復元
// TransportationService: {"OriginID":[bytes],"DestinationID":[bytes],"Mode":"OCEAN"}
// LocationService: {"LocationID":[bytes],"ServiceType":"HANDLING"}
// Note: route.LocationID (type LocationID uuid.UUID) はjson.Marshalでバイト配列として保存される
func deserializeScope(data []byte) pricing.ServiceScope {
	if len(data) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	// TransportationService の判定: OriginID フィールドの存在
	if _, ok := raw["OriginID"]; ok {
		var s struct {
			OriginID      route.LocationID `json:"OriginID"`
			DestinationID route.LocationID `json:"DestinationID"`
			Mode          string           `json:"Mode"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil
		}
		return &pricing.TransportationService{
			OriginID:      s.OriginID,
			DestinationID: s.DestinationID,
			Mode:          shared.TransportMode(s.Mode),
		}
	}

	// LocationService の判定: LocationID フィールドの存在
	if _, ok := raw["LocationID"]; ok {
		var s struct {
			LocationID  route.LocationID `json:"LocationID"`
			ServiceType string           `json:"ServiceType"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil
		}
		return &pricing.LocationService{
			LocationID:  s.LocationID,
			ServiceType: s.ServiceType,
		}
	}

	return nil
}

// deserializeLogic: JSONB からPricingStrategyを復元
func deserializeLogic(data []byte) pricing.PricingStrategy {
	if len(data) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	// FlatStrategy の判定: Amount フィールドの存在
	if _, ok := raw["Amount"]; ok {
		var s struct {
			Amount struct {
				Amount   decimal.Decimal `json:"Amount"`
				Currency string          `json:"Currency"`
			} `json:"Amount"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil
		}
		return &pricing.FlatStrategy{
			Amount: shared.Money{Amount: s.Amount.Amount, Currency: s.Amount.Currency},
		}
	}

	// ExpressionStrategy の判定: Formula フィールドの存在
	if _, ok := raw["Formula"]; ok {
		var s struct {
			Formula  string `json:"Formula"`
			Currency string `json:"Currency"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil
		}
		return &pricing.ExpressionStrategy{
			Formula:  s.Formula,
			Currency: s.Currency,
		}
	}

	// CompositeStrategy の判定: Steps フィールドの存在
	if _, ok := raw["Steps"]; ok {
		var s struct {
			Steps []json.RawMessage `json:"Steps"`
		}
		if err := json.Unmarshal(data, &s); err != nil {
			return nil
		}

		var steps []pricing.PricingStrategy
		for _, stepData := range s.Steps {
			if step := deserializeLogic(stepData); step != nil {
				steps = append(steps, step)
			}
		}
		return &pricing.CompositeStrategy{Steps: steps}
	}

	return nil
}
