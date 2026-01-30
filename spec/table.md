# 国際物流SCMプラットフォーム データベース仕様書 (v1.0)

## 1. 概要

本データベースは、物理的な物流ネットワーク（グラフ構造）と、その上で発生する商流（レート・契約・コスト）を管理するために設計されている。
PostgreSQLを採用し、地理空間検索にはPostGIS、複雑な料金ロジックや制約条件にはJSONBを活用する。

### 必須拡張機能

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

```

## 2. 共通定義 (ENUM Types)

テーブル間で共通して利用される列挙型定義。

| 型名 | 値の例 | 説明 |
| --- | --- | --- |
| `location_type` | `PORT`, `AIRPORT`, `WAREHOUSE`, `REGION`, `COUNTRY`, `FACTORY` | 拠点の種類 |
| `transport_mode` | `OCEAN_FCL`, `OCEAN_LCL`, `AIR`, `TRUCK`, `RAIL`, `WAREHOUSE` | 輸送手段または保管 |
| `currency_code` | `USD`, `JPY`, `EUR`, `CNY` | ISO 4217 通貨コード |
| `charge_unit` | `PER_CONTAINER`, `PER_SHIPMENT`, `PER_KG`, `PER_M3`, `PER_TRIP` | 課金単位 |
| `container_type` | `20DC`, `40DC`, `40HC`, `LCL` | コンテナサイズ・種類 |
| `charge_category` | `FREIGHT_BASIC`, `SURCHARGE`, `ORIGIN_LOCAL`, `DEST_LOCAL`, `DUTY_TAX` | 集計・比較用費目カテゴリ |

---

## 3. 物理ネットワーク層 (Physical Network Layer)

物流の「場所(Nodes)」と「経路(Edges)」を定義するレイヤー。

### 3.1. locations (拠点マスタ)

港、空港、倉庫、および論理的なエリア（関東、北米西岸など）を管理する。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `un_locode` | `VARCHAR(5)` | Index | UN/LOCODE (例: JPTYO)。倉庫等はNULL可。 |
| `name` | `VARCHAR(255)` | NOT NULL | 拠点名称 (English) |
| `type` | `location_type` | NOT NULL | 拠点の種類 |
| `country_code` | `VARCHAR(2)` | NOT NULL | ISO 3166-1 alpha-2 (JP, US) |
| `parent_location_id` | `UUID` | FK | 親拠点ID。階層構造を実現 (例: 大井埠頭 -> 東京港)。 |
| `geom` | `GEOMETRY(POINT)` | Index | 緯度経度 (SRID 4326)。半径検索等に使用。 |
| `attributes` | `JSONB` | Default `{}` | 固有属性 (水深、設備、TimeZoneなど)。 |
| `created_at` | `TIMESTAMPTZ` |  | 作成日時 |

### 3.2. transport_edges (輸送経路)

2つの拠点を結ぶ物理的・論理的なルート。経路探索（Routing）の基礎データとなる。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `source_location_id` | `UUID` | FK, Not Null | 出発地ID |
| `target_location_id` | `UUID` | FK, Not Null | 到着地ID |
| `mode` | `transport_mode` | Not Null | 輸送モード |
| `distance_km` | `NUMERIC(10,2)` |  | 物理距離 (km) |
| `default_transit_time` | `INTEGER` |  | 標準所要時間 (Hours) |
| `path_geom` | `GEOMETRY(LINESTRING)` |  | 実際のルート形状データ (地図描画用) |
| `attributes` | `JSONB` | Default `{}` | 経路属性 (CO2排出係数、重量制限など) |

* **Unique制約:** `(source_location_id, target_location_id, mode)`
* **備考:** 実際のルーティング計算には `pgRouting` を利用する想定。

---

## 4. 商流・レート層 (Commercial & Rates Layer)

入札、見積もり、契約レートを管理するレイヤー。

### 4.1. partners (取引先マスタ)

船会社、フォワーダー、トラック会社、倉庫業者など。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `name` | `VARCHAR(255)` | NOT NULL | 企業名 |
| `scac_code` | `VARCHAR(4)` |  | SCACコード (トラッキングAPI連携用) |
| `type` | `VARCHAR(50)` |  | CARRIER, FORWARDER, NVOCC 等 |

### 4.2. rate_cards (レートヘッダー)

「誰と、どの区間で、いつまで」という契約の定義。実際の金額は `rate_items` に保持する。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `partner_id` | `UUID` | FK, Not Null | 取引先ID |
| `origin_location_id` | `UUID` | FK, Not Null | 発地ID (Region等の広域指定も可) |
| `dest_location_id` | `UUID` | FK, Not Null | 着地ID |
| `mode` | `transport_mode` | NOT NULL | 輸送モード |
| `validity` | `DATERANGE` | Index, Not Null | 有効期間 (`[2026-01-01, 2026-03-31)`) |
| `contract_reference` | `VARCHAR(100)` |  | 契約番号、NACCSコード等 |
| `conditions` | `JSONB` | Default `[]` | 自然言語の制約条件を構造化した配列。<br>

<br>例: `[{"tag": "NO_HAZMAT", "is_blocking": true}]` |
| `is_active` | `BOOLEAN` | Default `true` | 論理削除用フラグ |

* **Index戦略:** `(origin, dest, partner)` および `validity` (GIST Index) を推奨。

### 4.3. rate_items (レート明細・計算ロジック)

具体的な金額、または計算ロジックを定義する。JSONBを活用し、距離制運賃表などの複雑な構造に対応する。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `rate_card_id` | `UUID` | FK, Not Null | 親レートカードID |
| `charge_name` | `VARCHAR(100)` | NOT NULL | 費目名 (Ocean Freight, BAF, Trucking Fee...) |
| `category` | `charge_category` | NOT NULL | 集計用カテゴリ (比較・正規化用) |
| `target_container` | `container_type` | Nullable | 特定コンテナ専用の場合に指定。共通ならNULL。 |
| `calculation_type` | `VARCHAR(20)` | Default `FIXED` | 計算タイプ: `FIXED`, `DISTANCE_TIER`, `WEIGHT_TIER` |
| `currency` | `currency_code` | NOT NULL | 通貨 (USD, JPY...) |
| `amount` | `NUMERIC(12,2)` | Nullable | 固定額の場合の値。 |
| `unit` | `charge_unit` | NOT NULL | 課金単位 |
| `tier_matrix` | `JSONB` | Nullable | 計算ロジック用マトリクスデータ。<br>

<br>距離制・重量制の場合の階段表を格納する。 |
| `notes` | `TEXT` |  | 備考 |

#### `tier_matrix` JSONB構造例 (距離制運賃)

```json
{
  "logic": "step",
  "steps": [
    {"max_km": 10, "price": 18060},
    {"max_km": 20, "price": 20160}
  ],
  "over_max_rule": {"unit_km": 20, "add_price": 4140}
}

```

---

## 5. マスタ・変換レイヤー (Normalization Layer)

表記ゆれの吸収、費目の標準化を行うためのレイヤー。

### 5.1. charge_codes (費目標準マスタ)

システム内で一意に定まる標準費目コード。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `code` | `VARCHAR(50)` | **PK** | 内部コード (例: `FREIGHT_BASIC`) |
| `name` | `VARCHAR(100)` | NOT NULL | 表示名称 (例: Basic Ocean Freight) |
| `category` | `charge_category` | NOT NULL | デフォルトカテゴリ |

### 5.2. charge_code_mappings (費目変換辞書)

外部データ（Excel等）の費目名と、内部標準コードの対応表。

| カラム名 | データ型 | 制約 | 説明 |
| --- | --- | --- | --- |
| `id` | `UUID` | **PK** | システム内部ID |
| `partner_id` | `UUID` | FK, Nullable | 特定業者専用のマッピングの場合に指定。 |
| `input_text` | `VARCHAR(100)` | NOT NULL | 外部入力テキスト (例: `OFT`, `Ocean Frt`) |
| `target_code` | `VARCHAR(50)` | FK, Not Null | 対応する標準コード (`charge_codes.code`) |

---

## 補足: 設計意図と拡張性

1. **JSONBの活用:**
* `locations.attributes`: 港ごとの特殊事情（ドラフト制限など）に対応。
* `rate_cards.conditions`: 「危険品不可」「スペース次第」等の定性条件をタグ管理。
* `rate_items.tier_matrix`: トラック運賃のような複雑なタリフ表を、レコードの大量生成なしに管理。


2. **階層構造:**
* `locations.parent_location_id`: 詳細な港（大井埠頭）へのレートがない場合、親（東京港）や祖父母（関東）のレートを検索するロジックを実装可能にする。


3. **比較・集計:**
* `rate_items.category`: All-in表記と内訳あり表記を、カテゴリレベルで合算して比較可能にするためのキー。