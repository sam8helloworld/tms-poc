-- ==========================================
-- 0. Prerequisites (区分値・マスタ)
-- ==========================================

-- 通貨コード (ISO 4217)
CREATE TYPE currency_code AS ENUM ('USD', 'JPY', 'EUR', 'CNY');

-- 課金単位
CREATE TYPE charge_unit AS ENUM (
    'PER_CONTAINER', -- コンテナ単位
    'PER_SHIPMENT',  -- 船積み単位 (Doc Feeなど)
    'PER_KG',        -- 重量単位 (航空貨物)
    'PER_M3'         -- 容積単位 (LCL/Warehouse)
);

-- コンテナサイズ/タイプ
CREATE TYPE container_type AS ENUM (
    '20DC', '40DC', '40HC', 'LCL', 'BULK'
);

-- 費目カテゴリー (集計・比較用)
CREATE TYPE charge_category AS ENUM (
    'FREIGHT_BASIC', -- 基本運賃 (Ocean Freight)
    'SURCHARGE_FUEL',-- 燃料サーチャージ (BAF)
    'SURCHARGE_CCY', -- 通貨サーチャージ (CAF)
    'ORIGIN_LOCAL',  -- 積地ローカル費用 (THC/Doc)
    'DEST_LOCAL',    -- 揚地ローカル費用
    'DUTY_TAX'       -- 税金
);

-- 簡易的なパートナーマスタ (船会社など)
CREATE TABLE partners (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    scac_code VARCHAR(4), -- Standard Carrier Alpha Code (例: MAEU)
    type VARCHAR(50)      -- CARRIER, FORWARDER, TRUCKER
);

-- ==========================================
-- 1. Rate Cards (Header: 契約・ルート定義)
-- ==========================================
-- ※データ量が増えるため、有効期限(validity)などでパーティショニング推奨

CREATE TABLE rate_cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- 誰のレートか
    partner_id UUID NOT NULL REFERENCES partners(id),
    
    -- 区間 (Locationsテーブルと紐づけ)
    -- ※Edge IDではなくLocation IDで持つのが一般的 (物理ルートは問わない契約が多いため)
    origin_location_id UUID NOT NULL REFERENCES locations(id),
    dest_location_id UUID NOT NULL REFERENCES locations(id),
    
    -- 輸送モード
    mode transport_mode NOT NULL,
    
    -- 有効期間 (PostgreSQL固有のRange型を使用)
    -- "&&" 演算子で「期間重複」や「特定日の包含」を爆速で検索可能
    validity DATERANGE NOT NULL,
    
    -- 契約参照番号 (船会社のContract Noなど)
    contract_reference VARCHAR(100),
    
    -- 自然言語の制約条件 (前述のConditionモデルをJSONBで格納)
    -- 例: [{"tag": "NO_HAZMAT", "is_blocking": true}]
    conditions JSONB DEFAULT '[]'::JSONB,
    
    -- 管理情報
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 検索用インデックス
-- 「ある期間に有効な、東京発・LA行きのレート」を引くための複合インデックス
CREATE INDEX idx_rate_cards_search ON rate_cards (origin_location_id, dest_location_id, partner_id);
-- 期間検索用 (GISTインデックス)
CREATE INDEX idx_rate_cards_validity ON rate_cards USING GIST (validity);


-- ==========================================
-- 2. Rate Items (Detail: 金額明細)
-- ==========================================
-- 1つのRateCardに対して、複数の費目が紐づく (1:N)
-- 例: RateID_1 -> { OceanFreight: $2000, BAF: $500, THC: ¥32000 }

CREATE TABLE rate_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rate_card_id UUID NOT NULL REFERENCES rate_cards(id),
    
    charge_name VARCHAR(100) NOT NULL,
    category charge_category NOT NULL,
    
    -- ★計算ロジックタイプを追加
    calculation_type VARCHAR(20) DEFAULT 'FIXED' CHECK (calculation_type IN ('FIXED', 'DISTANCE_TIER', 'WEIGHT_TIER', 'PERCENTAGE')),
    
    currency currency_code NOT NULL,
    amount NUMERIC(12, 2), -- FIXEDの場合はここが入る
    
    unit charge_unit NOT NULL,
    
    -- ★複雑な計算ロジック（階段状の運賃表など）を格納するフィールド
    tier_matrix JSONB DEFAULT NULL, 
    
    notes TEXT
);

-- 集計用インデックス
CREATE INDEX idx_rate_items_card ON rate_items(rate_card_id);