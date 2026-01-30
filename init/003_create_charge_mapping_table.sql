-- 1. 内部標準マスタ (これがシステムの正)
CREATE TABLE charge_codes (
    code VARCHAR(50) PRIMARY KEY, -- 例: 'FREIGHT_BASIC', 'SURCHARGE_BAF'
    name VARCHAR(100),            -- 例: 'Basic Ocean Freight'
    category charge_category      -- 例: 'FREIGHT', 'SURCHARGE'
);

-- 2. 変換マッピング (辞書)
CREATE TABLE charge_code_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    partner_id UUID,              -- 特定の業者専用の呼び名なら指定。NULLなら全社共通
    input_text VARCHAR(100) NOT NULL, -- 外部から来る文字列 (例: 'OFT')
    target_code VARCHAR(50) NOT NULL REFERENCES charge_codes(code)
);

-- インデックス: 取り込み時に爆速で引くため
CREATE INDEX idx_mapping_lookup ON charge_code_mappings(partner_id, input_text);