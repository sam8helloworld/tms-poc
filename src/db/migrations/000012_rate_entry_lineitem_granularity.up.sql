ALTER TABLE rate_entries
    ADD COLUMN tariff_line_item_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    ADD COLUMN charge_code VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN category VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN unit_price_amount NUMERIC(20,4) NOT NULL DEFAULT 0,
    ADD COLUMN unit_price_currency VARCHAR(3) NOT NULL DEFAULT 'USD';

ALTER TABLE rate_entries
    ALTER COLUMN tariff_line_item_id DROP DEFAULT,
    ALTER COLUMN charge_code DROP DEFAULT,
    ALTER COLUMN category DROP DEFAULT,
    ALTER COLUMN unit_price_amount DROP DEFAULT,
    ALTER COLUMN unit_price_currency DROP DEFAULT;

CREATE INDEX idx_rate_entries_line_item ON rate_entries(tariff_line_item_id);
