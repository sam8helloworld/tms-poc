package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
)

// PostgresVendorRepo: Vendor集約のPostgreSQL実装
type PostgresVendorRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresVendorRepo(pool *pgxpool.Pool) *PostgresVendorRepo {
	return &PostgresVendorRepo{pool: pool}
}

func (r *PostgresVendorRepo) Save(ctx context.Context, v *contract.Vendor) error {
	capabilities, err := json.Marshal(v.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal capabilities: %w", err)
	}
	contacts, err := json.Marshal(v.Contacts)
	if err != nil {
		return fmt.Errorf("marshal contacts: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO vendors (id, name, type, credit_rating, payment_days, payment_currency,
			preferred_vendor, capabilities, contacts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, type = EXCLUDED.type,
			credit_rating = EXCLUDED.credit_rating,
			payment_days = EXCLUDED.payment_days,
			payment_currency = EXCLUDED.payment_currency,
			preferred_vendor = EXCLUDED.preferred_vendor,
			capabilities = EXCLUDED.capabilities,
			contacts = EXCLUDED.contacts,
			updated_at = EXCLUDED.updated_at`,
		v.ID, v.Name, string(v.Type), string(v.CreditRating),
		v.PaymentTerms.DaysFromInvoice, v.PaymentTerms.Currency,
		v.PreferredVendor, capabilities, contacts,
		v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (r *PostgresVendorRepo) FindByID(ctx context.Context, id uuid.UUID) (*contract.Vendor, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, type, credit_rating, payment_days, payment_currency,
			preferred_vendor, capabilities, contacts, created_at, updated_at
		FROM vendors WHERE id = $1`, id)

	return scanVendor(row)
}

func (r *PostgresVendorRepo) FindByName(ctx context.Context, name string) ([]*contract.Vendor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, credit_rating, payment_days, payment_currency,
			preferred_vendor, capabilities, contacts, created_at, updated_at
		FROM vendors WHERE name ILIKE '%' || $1 || '%' ORDER BY name`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vendors []*contract.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		vendors = append(vendors, v)
	}
	return vendors, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanVendor(s scannable) (*contract.Vendor, error) {
	var (
		v                contract.Vendor
		providerType     string
		creditRating     string
		paymentDays      int
		paymentCurrency  string
		capabilitiesJSON []byte
		contactsJSON     []byte
	)

	err := s.Scan(
		&v.ID, &v.Name, &providerType, &creditRating,
		&paymentDays, &paymentCurrency,
		&v.PreferredVendor, &capabilitiesJSON, &contactsJSON,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.NewDomainError(shared.ErrNotFound, "vendor not found")
		}
		return nil, err
	}

	v.Type = contract.ProviderType(providerType)
	v.CreditRating = contract.CreditRating(creditRating)
	v.PaymentTerms = contract.PaymentTerms{
		DaysFromInvoice: paymentDays,
		Currency:        paymentCurrency,
	}

	if err := json.Unmarshal(capabilitiesJSON, &v.Capabilities); err != nil {
		return nil, fmt.Errorf("unmarshal capabilities: %w", err)
	}
	if err := json.Unmarshal(contactsJSON, &v.Contacts); err != nil {
		return nil, fmt.Errorf("unmarshal contacts: %w", err)
	}

	return &v, nil
}
