DROP INDEX IF EXISTS idx_rate_entries_line_item;
ALTER TABLE rate_entries
    DROP COLUMN IF EXISTS tariff_line_item_id,
    DROP COLUMN IF EXISTS charge_code,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS unit_price_amount,
    DROP COLUMN IF EXISTS unit_price_currency;
