# convertToTariff テストデータ仕様書

本ドキュメントは `convert_to_tariff_test.go` で使用するテストデータの業務背景を説明します。
各テストケースは国際物流における実際の料金表（Tariff）構造をモデル化しています。

---

## 共通条件

| 項目 | 値 |
|---|---|
| 有効期間 | 2026/04/01 〜 2027/03/31 |
| ルート | 日本 → 米国（東京港/成田空港 → ロサンゼルス） |

### テスト用ロケーションID

| 変数名 | 業務上の場所 |
|---|---|
| `tokyoPortID` | 東京港（海上輸送の起点CY） |
| `losAngelesID` | ロサンゼルス港（海上輸送の終点CY） |
| `naritaID` | 成田国際空港（航空輸送の起点） |
| `laxAirportID` | LAX国際空港（航空輸送の終点） |
| `warehouseID` | 内陸倉庫（ドレージの到着地/出発地） |
| `factoryID` | 工場（Door-to-Doorの起点） |
| `customsOffice` | 通関事務所 |
| `cfsWarehouseID` | CFS倉庫（混載貨物の集約/仕分け拠点） |

---

## 1. 海上輸送 FCL（TestConvertToTariff_OceanFreight_FCL）

**業務シナリオ**: 船会社から取得した東京港→ロサンゼルス港のFCL（コンテナ単位）料金表。
20ft/40ftコンテナ1本あたりの固定料金体系。

**元となる見積書のイメージ**:

```
ACME Shipping Co., Ltd.
OCEAN FREIGHT TARIFF - FCL (Tokyo → Los Angeles)
Valid: 2026/04/01 - 2027/03/31

[基本運賃・割増料 (TRANSPORTATION scope: Tokyo Port → LA Port, OCEAN)]
Charge Code   | Description                    | Rate     | Currency
------------- | ------------------------------ | -------- | --------
OFT           | Ocean Freight                  | 2,500.00 | USD
BAF           | Bunker Adjustment Factor       |   450.00 | USD
LSS           | Low Sulphur Surcharge          |   120.00 | USD
CAF           | Currency Adjustment Factor     |    80.00 | USD
CIC           | Container Imbalance Charge     |   200.00 | USD
PSS           | Peak Season Surcharge          |   300.00 | USD
WRS           | War Risk Surcharge             |    50.00 | USD

[起点ローカルチャージ (LOCATION scope: Tokyo Port)]
Charge Code          | Description                 | Rate    | Currency
-------------------- | --------------------------- | ------- | --------
THC                  | Terminal Handling Charge     | 35,000  | JPY
DOC_FEE              | Documentation Fee (B/L)     |  5,000  | JPY
SEAL_FEE             | Container Seal Fee          |  1,500  | JPY
CONTAINER_CLEANING   | Container Cleaning Fee      |  8,000  | JPY
TELEX_RELEASE        | Telex Release Fee           |  3,000  | JPY

[到着地ローカルチャージ (LOCATION scope: LA Port)]
Charge Code   | Description                    | Rate   | Currency
------------- | ------------------------------ | ------ | --------
THC_DEST      | Dest Terminal Handling Charge   | 350.00 | USD
```

**ポイント**:
- 基本運賃と割増料は同一区間のTRANSPORTATION scopeにまとめられる
- ローカルチャージは発生場所ごとにLOCATION scopeで分離
- 通貨が日本側（JPY）と米国側（USD）で異なる

---

## 2. 航空輸送（TestConvertToTariff_AirFreight）

**業務シナリオ**: フォワーダーから取得した成田→LAXの航空運賃。
航空貨物は重量帯別の段階的単価が特徴で、CEL式で表現。

**元となる見積書のイメージ**:

```
Global Air Logistics Inc.
AIR FREIGHT TARIFF (NRT → LAX)
Valid: 2026/04/01 - 2027/03/31

[航空運賃 (TRANSPORTATION scope: Narita → LAX Airport, AIR)]
Weight Band       | Rate/kg | Currency | 表現方式
----------------- | ------- | -------- | -----------
Normal (+45kg)    |   8.50  | USD      | CEL: 重量帯別条件式
+100kg            |   7.20  | USD      |
+300kg            |   6.00  | USD      |
+500kg            |   5.50  | USD      |
+1000kg           |   5.00  | USD      |

[割増料 (TRANSPORTATION scope: Narita → LAX Airport, AIR)]
Charge Code | Description         | Formula                  | Currency
----------- | ------------------- | ------------------------ | --------
FSC         | Fuel Surcharge      | chargeable_weight * 1.20 | USD
SSC         | Security Surcharge  | chargeable_weight * 0.10 | USD

[空港諸掛 (LOCATION scope: Narita)]
Charge Code      | Description          | Rate   | Currency
---------------- | -------------------- | ------ | --------
AWB_FEE          | Air Waybill Fee      |  3,000 | JPY
TERMINAL_CHARGE  | Terminal Charge      |  25.00 | USD
XRAY_FEE         | X-Ray Inspection Fee |  5,000 | JPY
```

**ポイント**:
- 航空運賃は重量帯で変動するためCEL_EXPRESSION戦略を使用
- `chargeable_weight = max(実重量, 容積重量)` は荷主側の計算コンテキストで決まる
- FSC/SSCもkg単価のためCEL式

---

## 3-A. ドレージ（TestConvertToTariff_Drayage）

**業務シナリオ**: トラック会社から取得した東京港→内陸倉庫のコンテナドレージ料金。
FCLコンテナをシャーシに載せて港から倉庫まで往復輸送する。

**元となる見積書のイメージ**:

```
ABC Trucking Co.
DRAYAGE TARIFF (Tokyo Port → Warehouse)
Valid: 2026/04/01 - 2027/03/31

[ドレージ料金 (TRANSPORTATION scope: Tokyo Port → Warehouse, TRUCK)]
Charge Code      | Description           | Rate    | Currency
---------------- | --------------------- | ------- | --------
DRAYAGE          | Round Trip Drayage    | 65,000  | JPY
3AXLE_SURCHARGE  | 3-Axle Chassis Fee    | 15,000  | JPY
MG_FEE           | Motor Generator Fee   | 20,000  | JPY
TOLL_FEE         | Highway Toll Fee      |  5,500  | JPY
```

**ポイント**:
- ドレージはラウンドトリップ（往復）料金が一般的
- 3軸シャーシは重量貨物（20t超）の場合に発生
- MG Feeは冷凍コンテナ（Reefer）輸送時のみ発生
- すべてTRUCKモードのTRANSPORTATION scope

---

## 3-B. 一般トラック/混載便（TestConvertToTariff_Haulage）

**業務シナリオ**: LCLや航空貨物の陸送料金。コンテナではなくバラ貨物の輸送。

**元となる見積書のイメージ**:

```
XYZ Haulage Service
HAULAGE TARIFF (Warehouse → Tokyo Port)
Valid: 2026/04/01 - 2027/03/31

Charge Code      | Description         | Rate / Formula                    | Currency
---------------- | ------------------- | --------------------------------- | --------
CHARTER_4T       | 4t Truck Charter    | 45,000 (固定)                     | JPY
LTL_RATE         | LTL Consolidation   | max(重量kg, 容積m3×280) × 35      | JPY
WAITING_CHARGE   | Waiting Charge      | max(0, 待機分-30) / 30 × 3,000    | JPY
```

**ポイント**:
- チャーター便は固定料金（FLAT）
- 混載便はRevenue Ton計算（重量と容積の大きい方）のCEL式
- 待機料は30分超過後の追加課金（条件付きCEL式）

---

## 4. 通関・取扱（TestConvertToTariff_CustomsAndHandling）

**業務シナリオ**: 通関業者（乙仲）から取得した通関費用と付随する取扱手数料。

**元となる見積書のイメージ**:

```
Japan Customs Broker Inc.
CUSTOMS & HANDLING FEE SCHEDULE
Valid: 2026/04/01 - 2027/03/31

[通関・取扱料金 (LOCATION scope: Customs Office)]
Charge Code         | Description               | Rate    | Currency
------------------- | ------------------------- | ------- | --------
CUSTOMS_EXPORT      | Export Customs Clearance   | 11,800  | JPY
CUSTOMS_IMPORT      | Import Customs Clearance   | 11,800  | JPY
HANDLING_CHARGE     | Forwarder Handling Fee     | 15,000  | JPY
CUSTOMS_INSPECTION  | Customs Inspection Fee     | 25,000  | JPY
FOOD_QUARANTINE     | Food Quarantine App. Fee   |  8,000  | JPY
OTHER_LAW_APP       | Other Law Application Fee  | 10,000  | JPY
```

**ポイント**:
- すべてLOCATION scopeの固定料金
- 通関料（11,800円）は法定料金
- 検査料・検疫料は発生した場合のみ（見積書には含めるが実際の請求は条件付き）

---

## 5. 倉庫/CFS（TestConvertToTariff_Warehousing）

**業務シナリオ**: CFS（Container Freight Station）倉庫の料金表。
LCL貨物の集約・仕分けおよび一時保管に関する料金。

**元となる見積書のイメージ**:

```
Tokyo CFS Warehouse Co.
WAREHOUSE TARIFF
Valid: 2026/04/01 - 2027/03/31

[倉庫料金 (LOCATION scope: CFS Warehouse)]
Charge Code      | Description        | Rate / Formula           | ServiceType | Currency
---------------- | ------------------ | ------------------------ | ----------- | --------
CFS_CHARGE       | CFS Charge         | Revenue Ton × 3,500     | STORAGE     | JPY
STORAGE_FEE      | Storage Fee        | 保管日数 × 容積m3 × 150  | STORAGE     | JPY
IN_OUT_HANDLING   | In/Out Handling    | 8,000 (固定)             | HANDLING    | JPY
DEVANNING_FEE    | Devanning Fee      | 35,000 (固定)            | HANDLING    | JPY
VANNING_FEE      | Vanning Fee        | 35,000 (固定)            | HANDLING    | JPY
LABELING_FEE     | Labeling Fee       | カートン数 × 50          | HANDLING    | JPY
```

**ポイント**:
- ServiceTypeがSTORAGE（保管系）とHANDLING（作業系）に分かれる
- CFS ChargeはRevenue Ton（重量tまたは容積m3の大きい方）ベース
- Storage Feeは日数×容積の変動料金（CEL式）
- Devanning/Vanningはコンテナ詰め/出し作業の固定料金

---

## 6. デマレージ/ディテンション（TestConvertToTariff_DemurrageDetention）

**業務シナリオ**: 船会社のフリータイム超過時の延滞料金。
金額そのものではなく「条件」に近いが、コスト影響が大きいため料金表に含める。

**元となる見積書のイメージ**:

```
ACME Shipping Co., Ltd.
DEMURRAGE & DETENTION SCHEDULE (Los Angeles)
Valid: 2026/04/01 - 2027/03/31

Type        | Free Time | Over Free Time Rate               | Currency
----------- | --------- | --------------------------------- | --------
DEMURRAGE   | 4 days    | max(0, 超過日数-4) × $150/day     | USD
DETENTION   | 7 days    | max(0, 超過日数-7) × $100/day     | USD
```

**ポイント**:
- Demurrage: 港内でのコンテナ保管延滞（CY内）
- Detention: 港外でのコンテナ返却遅延（荷主の倉庫等）
- フリータイム以内なら無料、超過分のみ課金（CEL式で表現）

---

## 7. フォワーダー All-in（TestConvertToTariff_ForwarderAllIn）

**業務シナリオ**: フォワーダーに一括で依頼したDoor-to-Door料金表。
工場出荷→輸出通関→海上輸送→輸入通関→倉庫搬入までの全工程を含む。

**元となる見積書のイメージ**:

```
International Freight Forwarder Corp.
ALL-IN DOOR-TO-DOOR TARIFF (Japan Factory → US Warehouse)
Valid: 2026/04/01 - 2027/03/31

Leg                   | Charge Code      | Category       | Scope          | Rate     | Currency
--------------------- | ---------------- | -------------- | -------------- | -------- | --------
工場→東京港 (TRUCK)    | PICKUP_DRAYAGE   | FREIGHT_BASIC  | TRANSPORTATION | 55,000   | JPY
東京港 輸出通関        | CUSTOMS_EXPORT   | DUTY_TAX       | LOCATION       | 11,800   | JPY
東京港 THC             | THC_ORIGIN       | ORIGIN_LOCAL   | LOCATION       | 35,000   | JPY
東京港→LA港 (OCEAN)    | OFT              | FREIGHT_BASIC  | TRANSPORTATION | 2,800.00 | USD
東京港→LA港 BAF        | BAF              | SURCHARGE_FUEL | TRANSPORTATION |   500.00 | USD
LA港 THC               | THC_DEST         | DEST_LOCAL     | LOCATION       |   380.00 | USD
LA港 輸入通関          | CUSTOMS_IMPORT   | DUTY_TAX       | LOCATION       |   150.00 | USD
LA港→倉庫 (TRUCK)      | DELIVERY_DRAYAGE | FREIGHT_BASIC  | TRANSPORTATION |   850.00 | USD
```

**ポイント**:
- 複数の輸送モード（TRUCK + OCEAN）が混在
- LOCATION scope（通関・THC）とTRANSPORTATION scope（輸送）が交互に出現
- 通貨も日本側（JPY）と米国側（USD）の混在
- フォワーダーのマージンが各項目に含まれているのが特徴

---

## 8. COMPOSITE 複合料金

### 8-A. 海上運賃パッケージ（TestConvertToTariff_CompositeOceanFreight）

**業務シナリオ**: OFT + BAF + CAF を1つのチャージコードにまとめたパッケージ料金。
船会社が「All-in Freight」として提示することがある。

```
OCEAN_FREIGHT_PKG = OFT($2,500) + BAF($450) + CAF($80)
→ CompositeStrategy(Steps: [Flat($2,500), Flat($450), Flat($80)])
```

### 8-B. 混合戦略（TestConvertToTariff_CompositeMixedStrategies）

**業務シナリオ**: 航空運賃の基本料金（固定）+ 重量帯別追加料金（CEL式）の組み合わせ。

```
AIR_FREIGHT_PKG = 基本料金($500) + 超過分(max(0, chargeable_weight - 45) × $3.50)
→ CompositeStrategy(Steps: [Flat($500), CelExpression("max(0, cw-45)*3.50")])
```

**ポイント**:
- FLAT同士の合成（8-A）と FLAT + CEL の混合合成（8-B）の両パターン
- CompositeStrategy内の各Stepは再帰的にbuildPricingStrategyで構築される
