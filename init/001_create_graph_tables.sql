-- 1. 拡張機能の有効化
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ==========================================
-- 2. Master Lookups (区分値)
-- ==========================================

-- Location Type: 場所の種類
CREATE TYPE location_type AS ENUM (
    'PORT',         -- 港
    'AIRPORT',      -- 空港
    'RAIL_YARD',    -- 鉄道ターミナル
    'WAREHOUSE',    -- 倉庫
    'FACTORY',      -- 工場
    'REGION',       -- エリア/ゾーン (例: 北米西岸)
    'COUNTRY',      -- 国
    'CITY'          -- 都市
);

-- Transport Mode: 輸送手段
CREATE TYPE transport_mode AS ENUM (
    'OCEAN_FCL',    -- 海上コンテナ
    'OCEAN_LCL',    -- 海上混載
    'AIR',          -- 航空
    'TRUCK',        -- トラック
    'RAIL',         -- 鉄道
    'BARGE'         -- はしけ
);

-- ==========================================
-- 3. Locations (Nodes)
-- ==========================================

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- UN/LOCODEは物流の共通言語 (例: JPTYO, USLGB)
    -- ただし、倉庫などコードがない場所もあるためNullable
    un_locode VARCHAR(5),
    
    name VARCHAR(255) NOT NULL,
    type location_type NOT NULL,
    country_code VARCHAR(2) NOT NULL, -- ISO 3166-1 alpha-2 (JP, US)
    
    -- 階層構造管理 (Adjacency List Model)
    -- 例: "大井埠頭" -> parent -> "東京港" -> parent -> "関東エリア"
    parent_location_id UUID REFERENCES locations(id),
    
    -- 地理空間情報 (PostGIS)
    -- 半径検索や地図描画に使用 (SRID 4326 = WGS84 経度緯度)
    geom GEOMETRY(POINT, 4326),
    
    -- 拡張属性 (JSONB)
    -- Timezone, 通関コード, 港湾の特性など、場所による属性違いを吸収
    attributes JSONB DEFAULT '{}'::JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 検索用インデックス
CREATE INDEX idx_locations_un_locode ON locations(un_locode);
CREATE INDEX idx_locations_parent ON locations(parent_location_id);
CREATE INDEX idx_locations_geom ON locations USING GIST(geom); -- 空間インデックス


-- ==========================================
-- 4. Transport Edges (Edges / Lanes)
-- ==========================================

CREATE TABLE transport_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- どこからどこへ
    source_location_id UUID NOT NULL REFERENCES locations(id),
    target_location_id UUID NOT NULL REFERENCES locations(id),
    
    -- どうやって (海、空、陸)
    mode transport_mode NOT NULL,
    
    -- 物理的な距離と標準リードタイム
    distance_km NUMERIC(10, 2),
    default_transit_time_hours INTEGER, -- 標準所要時間
    
    -- ルートの形状 (直生ではなく、航路や道路の形状を持つ場合)
    -- PostGISのLineString型で保持し、Deck.gl等で描画する
    path_geom GEOMETRY(LINESTRING, 4326),
    
    -- CO2排出係数などのメタデータ
    attributes JSONB DEFAULT '{}'::JSONB,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- ユニーク制約: 同じ場所・同じ手段のルートは1つ
    CONSTRAINT uq_transport_edges UNIQUE (source_location_id, target_location_id, mode)
);

-- 検索用インデックス
CREATE INDEX idx_edges_source ON transport_edges(source_location_id);
CREATE INDEX idx_edges_target ON transport_edges(target_location_id);
-- 経路探索時に双方向検索を高速化するための複合インデックス
CREATE INDEX idx_edges_route ON transport_edges(source_location_id, target_location_id, mode);