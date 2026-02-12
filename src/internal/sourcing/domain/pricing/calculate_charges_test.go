package pricing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// ============================================================
// テストヘルパー
// ============================================================

var (
	// ロケーションID（testdata.mdの定義に準拠）
	calcTokyoPortID    = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	calcLosAngelesID   = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000002"))
	calcNaritaID       = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000003"))
	calcLaxAirportID   = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000004"))
	calcWarehouseID    = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000005"))
	calcFactoryID      = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000006"))
	calcCustomsOffice  = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000007"))
	calcCfsWarehouseID = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000008"))
	// 無関係なロケーション（スコープ不一致テスト用）
	calcShanghaiPortID = route.LocationID(uuid.MustParse("10000000-0000-0000-0000-000000000099"))
)

// newTestTariff: テスト用Tariff生成ヘルパー
func newTestTariff(name string) *Tariff {
	now := time.Now()
	return &Tariff{
		ID:         uuid.New(),
		ContractID: uuid.New(),
		Name:       name,
		Version:    1,
		EffectiveDate: shared.DateRange{
			From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		LineItems: make([]TariffLineItem, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// mustMoney: テスト用Money生成ヘルパー（panic on error）
func mustMoney(amount float64, currency string) shared.Money {
	m, err := shared.NewMoney(decimal.NewFromFloat(amount), currency)
	if err != nil {
		panic(err)
	}
	return m
}

// addFlatLineItem: FLAT料金のLineItemをTariffに追加するヘルパー
func addFlatLineItem(t *testing.T, tariff *Tariff, chargeCode, category string, scope ServiceScope, amount float64, currency string) {
	t.Helper()
	err := tariff.AddLineItem(TariffLineItem{
		ID:         uuid.New(),
		ChargeCode: chargeCode,
		Category:   category,
		Scope:      scope,
		Logic:      &FlatStrategy{Amount: mustMoney(amount, currency)},
	})
	if err != nil {
		t.Fatalf("failed to add line item %s: %v", chargeCode, err)
	}
}

// addExprLineItem: Expression式料金のLineItemをTariffに追加するヘルパー
func addExprLineItem(t *testing.T, tariff *Tariff, chargeCode, category string, scope ServiceScope, formula, currency string) {
	t.Helper()
	err := tariff.AddLineItem(TariffLineItem{
		ID:         uuid.New(),
		ChargeCode: chargeCode,
		Category:   category,
		Scope:      scope,
		Logic:      &ExpressionStrategy{Formula: formula, Currency: currency},
	})
	if err != nil {
		t.Fatalf("failed to add line item %s: %v", chargeCode, err)
	}
}

// addCompositeLineItem: COMPOSITE料金のLineItemをTariffに追加するヘルパー
func addCompositeLineItem(t *testing.T, tariff *Tariff, chargeCode, category string, scope ServiceScope, steps []PricingStrategy) {
	t.Helper()
	err := tariff.AddLineItem(TariffLineItem{
		ID:         uuid.New(),
		ChargeCode: chargeCode,
		Category:   category,
		Scope:      scope,
		Logic:      &CompositeStrategy{Steps: steps},
	})
	if err != nil {
		t.Fatalf("failed to add line item %s: %v", chargeCode, err)
	}
}

// oceanRoute: 東京港→LA港の海上ルートを生成
func oceanRoute() route.PhysicalRoute {
	return route.PhysicalRoute{
		ID:            route.PhysicalRouteID(uuid.New()),
		OriginID:      calcTokyoPortID,
		DestinationID: calcLosAngelesID,
		Segments: []route.RouteSegment{
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    0,
				OriginLocationID: calcTokyoPortID,
				OriginType:       shared.LocTypePort,
				DestLocationID:   calcLosAngelesID,
				DestType:         shared.LocTypePort,
				Mode:             shared.ModeOcean,
				DistanceKm:       decimal.NewFromInt(8800),
			},
		},
	}
}

// airRoute: 成田→LAXの航空ルートを生成
func airRoute() route.PhysicalRoute {
	return route.PhysicalRoute{
		ID:            route.PhysicalRouteID(uuid.New()),
		OriginID:      calcNaritaID,
		DestinationID: calcLaxAirportID,
		Segments: []route.RouteSegment{
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    0,
				OriginLocationID: calcNaritaID,
				OriginType:       shared.LocTypeAirport,
				DestLocationID:   calcLaxAirportID,
				DestType:         shared.LocTypeAirport,
				Mode:             shared.ModeAir,
				DistanceKm:       decimal.NewFromInt(8800),
			},
		},
	}
}

// drayageRoute: 東京港→倉庫のドレージルートを生成
func drayageRoute() route.PhysicalRoute {
	return route.PhysicalRoute{
		ID:            route.PhysicalRouteID(uuid.New()),
		OriginID:      calcTokyoPortID,
		DestinationID: calcWarehouseID,
		Segments: []route.RouteSegment{
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    0,
				OriginLocationID: calcTokyoPortID,
				OriginType:       shared.LocTypePort,
				DestLocationID:   calcWarehouseID,
				DestType:         shared.LocTypeWarehouse,
				Mode:             shared.ModeTruck,
				DistanceKm:       decimal.NewFromInt(50),
			},
		},
	}
}

// doorToDoorRoute: 工場→東京港→LA港→倉庫のDoor-to-Doorルートを生成
func doorToDoorRoute() route.PhysicalRoute {
	return route.PhysicalRoute{
		ID:            route.PhysicalRouteID(uuid.New()),
		OriginID:      calcFactoryID,
		DestinationID: calcWarehouseID,
		Segments: []route.RouteSegment{
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    0,
				OriginLocationID: calcFactoryID,
				OriginType:       shared.LocTypeDoor,
				DestLocationID:   calcTokyoPortID,
				DestType:         shared.LocTypePort,
				Mode:             shared.ModeTruck,
				DistanceKm:       decimal.NewFromInt(30),
			},
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    1,
				OriginLocationID: calcTokyoPortID,
				OriginType:       shared.LocTypePort,
				DestLocationID:   calcLosAngelesID,
				DestType:         shared.LocTypePort,
				Mode:             shared.ModeOcean,
				DistanceKm:       decimal.NewFromInt(8800),
			},
			{
				ID:               route.RouteSegmentID(uuid.New()),
				SequenceOrder:    2,
				OriginLocationID: calcLosAngelesID,
				OriginType:       shared.LocTypePort,
				DestLocationID:   calcWarehouseID,
				DestType:         shared.LocTypeWarehouse,
				Mode:             shared.ModeTruck,
				DistanceKm:       decimal.NewFromInt(40),
			},
		},
	}
}

// simpleCargoItems: テスト用CargoItem（20DC コンテナ2本分）
func simpleCargoItems() []CargoItem {
	return []CargoItem{
		{
			ID:          uuid.New(),
			ProductName: "Electronics",
			HSCode:      "8471.30",
			Quantity:    decimal.NewFromInt(2),
			WeightKG:    decimal.NewFromInt(18000),
			VolumeM3:    decimal.NewFromInt(30),
			PackageType: "PALLET",
		},
	}
}

// simpleConditions: テスト用の計算条件
func simpleConditions() CalculationConditions {
	return CalculationConditions{
		Incoterms:  "FOB",
		Attributes: map[string]string{},
	}
}

// findAppliedByChargeCode: AppliedItemsから指定ChargeCodeの項目を検索
func findAppliedByChargeCode(items []AppliedChargeItem, code string) *AppliedChargeItem {
	for _, item := range items {
		if item.ChargeCode == code {
			return &item
		}
	}
	return nil
}

// findSkippedByChargeCode: SkippedItemsから指定ChargeCodeの項目を検索
func findSkippedByChargeCode(items []SkippedChargeItem, code string) *SkippedChargeItem {
	for _, item := range items {
		if item.ChargeCode == code {
			return &item
		}
	}
	return nil
}

// findTotalByCurrency: TotalAmountsから指定通貨の合計を検索
func findTotalByCurrency(totals []CurrencyTotal, currency string) *CurrencyTotal {
	for _, t := range totals {
		if t.Currency == currency {
			return &t
		}
	}
	return nil
}

// ============================================================
// 1. 海上輸送FCL: 基本ケース
//    複数FLAT費目、TRANSPORTATION+LOCATION scope混在、JPY/USD混在
// ============================================================

func TestCalculateCharges_OceanFCL(t *testing.T) {
	tariff := newTestTariff("2026 Japan-US Ocean FCL Rate")

	oceanTransport := TransportationService{
		OriginID:      calcTokyoPortID,
		DestinationID: calcLosAngelesID,
		Mode:          shared.ModeOcean,
	}
	originLocation := LocationService{LocationID: calcTokyoPortID, ServiceType: "HANDLING"}
	destLocation := LocationService{LocationID: calcLosAngelesID, ServiceType: "HANDLING"}

	// 基本運賃・割増料（USD, TRANSPORTATION scope）
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2500.00, "USD")
	addFlatLineItem(t, tariff, "BAF", "SURCHARGE_FUEL", oceanTransport, 450.00, "USD")
	addFlatLineItem(t, tariff, "LSS", "SURCHARGE_FUEL", oceanTransport, 120.00, "USD")
	addFlatLineItem(t, tariff, "CAF", "SURCHARGE_CCY", oceanTransport, 80.00, "USD")
	addFlatLineItem(t, tariff, "CIC", "SURCHARGE_FUEL", oceanTransport, 200.00, "USD")
	addFlatLineItem(t, tariff, "PSS", "SURCHARGE_FUEL", oceanTransport, 300.00, "USD")
	addFlatLineItem(t, tariff, "WRS", "SURCHARGE_FUEL", oceanTransport, 50.00, "USD")
	// 起点ローカルチャージ（JPY, LOCATION scope: Tokyo Port）
	addFlatLineItem(t, tariff, "THC", "ORIGIN_LOCAL", originLocation, 35000, "JPY")
	addFlatLineItem(t, tariff, "DOC_FEE", "ORIGIN_LOCAL", originLocation, 5000, "JPY")
	addFlatLineItem(t, tariff, "SEAL_FEE", "ORIGIN_LOCAL", originLocation, 1500, "JPY")
	addFlatLineItem(t, tariff, "CONTAINER_CLEANING", "ORIGIN_LOCAL", originLocation, 8000, "JPY")
	addFlatLineItem(t, tariff, "TELEX_RELEASE", "ORIGIN_LOCAL", originLocation, 3000, "JPY")
	// 到着地ローカルチャージ（USD, LOCATION scope: LA Port）
	addFlatLineItem(t, tariff, "THC_DEST", "DEST_LOCAL", destLocation, 350.00, "USD")

	// CalculationRequest: コンテナ2本
	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// 全13件が適用されるはず
	if result.AppliedCount() != 13 {
		t.Errorf("AppliedCount = %d, want 13", result.AppliedCount())
	}
	if result.SkippedCount() != 0 {
		t.Errorf("SkippedCount = %d, want 0", result.SkippedCount())
	}
	if !result.HasAppliedItems() {
		t.Error("HasAppliedItems should be true")
	}

	// 個別費目の金額検証（FlatStrategy: 単価 × Quantity）
	// Quantity = TotalQuantity = 2（CargoItemのQuantityが2）
	qty := decimal.NewFromInt(2)

	oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
	if oft == nil {
		t.Fatal("OFT not found in applied items")
	}
	expectedOFT := decimal.NewFromFloat(2500.00).Mul(qty)
	if !oft.Amount.Amount.Equal(expectedOFT) {
		t.Errorf("OFT amount = %v, want %v (2500 * 2)", oft.Amount.Amount, expectedOFT)
	}
	if oft.Amount.Currency != "USD" {
		t.Errorf("OFT currency = %q, want USD", oft.Amount.Currency)
	}
	if oft.Category != "FREIGHT_BASIC" {
		t.Errorf("OFT category = %q, want FREIGHT_BASIC", oft.Category)
	}
	if oft.ScopeDescription == "" {
		t.Error("OFT ScopeDescription should not be empty")
	}

	thc := findAppliedByChargeCode(result.AppliedItems, "THC")
	if thc == nil {
		t.Fatal("THC not found in applied items")
	}
	expectedTHC := decimal.NewFromFloat(35000).Mul(qty)
	if !thc.Amount.Amount.Equal(expectedTHC) {
		t.Errorf("THC amount = %v, want %v (35000 * 2)", thc.Amount.Amount, expectedTHC)
	}
	if thc.Amount.Currency != "JPY" {
		t.Errorf("THC currency = %q, want JPY", thc.Amount.Currency)
	}

	// 通貨別合計の検証
	// USD: (2500+450+120+80+200+300+50+350) * 2 = 4050 * 2 = 8100
	usdTotal := findTotalByCurrency(result.TotalAmounts, "USD")
	if usdTotal == nil {
		t.Fatal("USD total not found")
	}
	expectedUSD := decimal.NewFromFloat(4050.00).Mul(qty)
	if !usdTotal.Amount.Amount.Equal(expectedUSD) {
		t.Errorf("USD total = %v, want %v", usdTotal.Amount.Amount, expectedUSD)
	}

	// JPY: (35000+5000+1500+8000+3000) * 2 = 52500 * 2 = 105000
	jpyTotal := findTotalByCurrency(result.TotalAmounts, "JPY")
	if jpyTotal == nil {
		t.Fatal("JPY total not found")
	}
	expectedJPY := decimal.NewFromFloat(52500).Mul(qty)
	if !jpyTotal.Amount.Amount.Equal(expectedJPY) {
		t.Errorf("JPY total = %v, want %v", jpyTotal.Amount.Amount, expectedJPY)
	}

	// TariffID, TariffNameが正しくセットされていること
	if result.TariffID != tariff.ID {
		t.Errorf("TariffID = %v, want %v", result.TariffID, tariff.ID)
	}
	if result.TariffName != "2026 Japan-US Ocean FCL Rate" {
		t.Errorf("TariffName = %q, want %q", result.TariffName, "2026 Japan-US Ocean FCL Rate")
	}
}

// ============================================================
// 2. 航空輸送: Expression式 + FLAT混在
//    Expression式で実際に計算される
// ============================================================

func TestCalculateCharges_AirFreight(t *testing.T) {
	tariff := newTestTariff("2026 NRT-LAX Air Freight Rate")

	airTransport := TransportationService{
		OriginID:      calcNaritaID,
		DestinationID: calcLaxAirportID,
		Mode:          shared.ModeAir,
	}
	naritaLocation := LocationService{LocationID: calcNaritaID, ServiceType: "HANDLING"}

	// 航空運賃（Expression式）
	// weight=100kg → 100*5.00 = 500 (重量が45kgを超えるので5.00/kg)
	addExprLineItem(t, tariff, "AIR_FREIGHT", "FREIGHT_BASIC", airTransport,
		"weight <= 45 ? weight * 8.50 : weight * 5.00", "USD")
	// 燃料割増（Expression式）
	// chargeable_weight = max(100, 0.5*1000) = 500 → 500*1.20 = 600
	addExprLineItem(t, tariff, "FSC", "SURCHARGE_FUEL", airTransport,
		"chargeable_weight * 1.20", "USD")
	// 保安料（Expression式）
	// chargeable_weight = 500 → 500*0.10 = 50
	addExprLineItem(t, tariff, "SSC", "SURCHARGE_FUEL", airTransport,
		"chargeable_weight * 0.10", "USD")
	// AWB Fee（FLAT, LOCATION）
	addFlatLineItem(t, tariff, "AWB_FEE", "ORIGIN_LOCAL", naritaLocation, 3000, "JPY")
	// Terminal Charge（FLAT, LOCATION）
	addFlatLineItem(t, tariff, "TERMINAL_CHARGE", "ORIGIN_LOCAL", naritaLocation, 25.00, "USD")
	// X-Ray Fee（FLAT, LOCATION）
	addFlatLineItem(t, tariff, "XRAY_FEE", "ORIGIN_LOCAL", naritaLocation, 5000, "JPY")

	// 航空貨物: 100kg, 0.5m3
	items := []CargoItem{
		{
			ID:          uuid.New(),
			ProductName: "Precision Parts",
			Quantity:    decimal.NewFromInt(1),
			WeightKG:    decimal.NewFromInt(100),
			VolumeM3:    decimal.NewFromFloat(0.5),
			PackageType: "CARTON",
		},
	}
	req, err := NewCalculationRequest(airRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// 全6件が適用（ルートが一致）
	if result.AppliedCount() != 6 {
		t.Errorf("AppliedCount = %d, want 6", result.AppliedCount())
	}
	if result.SkippedCount() != 0 {
		t.Errorf("SkippedCount = %d, want 0", result.SkippedCount())
	}

	// AIR_FREIGHT: weight=100kg, 100>45 なので 100*5.00 = 500
	airFreight := findAppliedByChargeCode(result.AppliedItems, "AIR_FREIGHT")
	if airFreight == nil {
		t.Fatal("AIR_FREIGHT not found")
	}
	if airFreight.Amount.Currency != "USD" {
		t.Errorf("AIR_FREIGHT currency = %q, want USD", airFreight.Amount.Currency)
	}
	expectedAirFreight := decimal.NewFromFloat(500.0)
	if !airFreight.Amount.Amount.Equal(expectedAirFreight) {
		t.Errorf("AIR_FREIGHT amount = %v, want %v (100kg * $5.00)", airFreight.Amount.Amount, expectedAirFreight)
	}

	// FLAT項目の検証（Quantity=1なので単価そのまま）
	awb := findAppliedByChargeCode(result.AppliedItems, "AWB_FEE")
	if awb == nil {
		t.Fatal("AWB_FEE not found")
	}
	if !awb.Amount.Amount.Equal(decimal.NewFromFloat(3000)) {
		t.Errorf("AWB_FEE amount = %v, want 3000", awb.Amount.Amount)
	}

	// 通貨別合計
	// USD: AIR_FREIGHT(500) + FSC(600) + SSC(50) + TERMINAL_CHARGE(25) = 1175
	usdTotal := findTotalByCurrency(result.TotalAmounts, "USD")
	if usdTotal == nil {
		t.Fatal("USD total not found")
	}
	expectedUSD := decimal.NewFromFloat(1175.0)
	if !usdTotal.Amount.Amount.Equal(expectedUSD) {
		t.Errorf("USD total = %v, want %v (500+600+50+25)", usdTotal.Amount.Amount, expectedUSD)
	}

	// JPY: FLAT(3000*1) + FLAT(5000*1) = 8000
	jpyTotal := findTotalByCurrency(result.TotalAmounts, "JPY")
	if jpyTotal == nil {
		t.Fatal("JPY total not found")
	}
	expectedJPY := decimal.NewFromFloat(8000)
	if !jpyTotal.Amount.Amount.Equal(expectedJPY) {
		t.Errorf("JPY total = %v, want %v", jpyTotal.Amount.Amount, expectedJPY)
	}
}

// ============================================================
// 3. ドレージ: 全FLAT、単一TRANSPORTATION scope
// ============================================================

func TestCalculateCharges_Drayage(t *testing.T) {
	tariff := newTestTariff("2026 Tokyo Port Drayage Rate")

	truckTransport := TransportationService{
		OriginID:      calcTokyoPortID,
		DestinationID: calcWarehouseID,
		Mode:          shared.ModeTruck,
	}

	addFlatLineItem(t, tariff, "DRAYAGE", "FREIGHT_BASIC", truckTransport, 65000, "JPY")
	addFlatLineItem(t, tariff, "3AXLE_SURCHARGE", "SURCHARGE_FUEL", truckTransport, 15000, "JPY")
	addFlatLineItem(t, tariff, "MG_FEE", "SURCHARGE_FUEL", truckTransport, 20000, "JPY")
	addFlatLineItem(t, tariff, "TOLL_FEE", "SURCHARGE_FUEL", truckTransport, 5500, "JPY")

	// コンテナ1本のドレージ
	items := []CargoItem{
		{
			ID:          uuid.New(),
			ProductName: "General Cargo",
			Quantity:    decimal.NewFromInt(1),
			WeightKG:    decimal.NewFromInt(15000),
			VolumeM3:    decimal.NewFromInt(28),
			PackageType: "CONTAINER",
		},
	}
	req, err := NewCalculationRequest(drayageRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 4 {
		t.Errorf("AppliedCount = %d, want 4", result.AppliedCount())
	}

	// 全費目がJPY, Quantity=1
	// 合計: 65000+15000+20000+5500 = 105500
	jpyTotal := findTotalByCurrency(result.TotalAmounts, "JPY")
	if jpyTotal == nil {
		t.Fatal("JPY total not found")
	}
	expected := decimal.NewFromFloat(105500)
	if !jpyTotal.Amount.Amount.Equal(expected) {
		t.Errorf("JPY total = %v, want %v", jpyTotal.Amount.Amount, expected)
	}

	// 通貨は1種類のみ
	if len(result.TotalAmounts) != 1 {
		t.Errorf("TotalAmounts count = %d, want 1", len(result.TotalAmounts))
	}
}

// ============================================================
// 4. フォワーダーAll-in Door-to-Door:
//    複数輸送モード(TRUCK+OCEAN)、LOCATION+TRANSPORTATION混在、JPY/USD混在
// ============================================================

func TestCalculateCharges_ForwarderAllIn(t *testing.T) {
	tariff := newTestTariff("2026 FWD All-in Japan-US Door-to-Door")

	pickupTransport := TransportationService{
		OriginID: calcFactoryID, DestinationID: calcTokyoPortID, Mode: shared.ModeTruck,
	}
	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	deliveryTransport := TransportationService{
		OriginID: calcLosAngelesID, DestinationID: calcWarehouseID, Mode: shared.ModeTruck,
	}
	tokyoLocation := LocationService{LocationID: calcTokyoPortID, ServiceType: "HANDLING"}
	laLocation := LocationService{LocationID: calcLosAngelesID, ServiceType: "HANDLING"}

	// Leg 1: 工場→東京港（TRUCK）
	addFlatLineItem(t, tariff, "PICKUP_DRAYAGE", "FREIGHT_BASIC", pickupTransport, 55000, "JPY")
	// 東京港: 輸出通関
	addFlatLineItem(t, tariff, "CUSTOMS_EXPORT", "DUTY_TAX", tokyoLocation, 11800, "JPY")
	// 東京港: THC
	addFlatLineItem(t, tariff, "THC_ORIGIN", "ORIGIN_LOCAL", tokyoLocation, 35000, "JPY")
	// Leg 2: 東京港→LA港（OCEAN）
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2800.00, "USD")
	// BAF
	addFlatLineItem(t, tariff, "BAF", "SURCHARGE_FUEL", oceanTransport, 500.00, "USD")
	// LA港: THC
	addFlatLineItem(t, tariff, "THC_DEST", "DEST_LOCAL", laLocation, 380.00, "USD")
	// LA港: 輸入通関
	addFlatLineItem(t, tariff, "CUSTOMS_IMPORT", "DUTY_TAX", laLocation, 150.00, "USD")
	// Leg 3: LA港→倉庫（TRUCK）
	addFlatLineItem(t, tariff, "DELIVERY_DRAYAGE", "FREIGHT_BASIC", deliveryTransport, 850.00, "USD")

	items := simpleCargoItems() // Quantity=2
	req, err := NewCalculationRequest(doorToDoorRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// 全8件が適用（Door-to-Doorルートですべてのスコープに一致）
	if result.AppliedCount() != 8 {
		t.Errorf("AppliedCount = %d, want 8", result.AppliedCount())
	}
	if result.SkippedCount() != 0 {
		t.Errorf("SkippedCount = %d, want 0", result.SkippedCount())
	}

	qty := decimal.NewFromInt(2)

	// 通貨別合計
	// JPY: (55000+11800+35000) * 2 = 101800 * 2 = 203600
	jpyTotal := findTotalByCurrency(result.TotalAmounts, "JPY")
	if jpyTotal == nil {
		t.Fatal("JPY total not found")
	}
	expectedJPY := decimal.NewFromFloat(101800).Mul(qty)
	if !jpyTotal.Amount.Amount.Equal(expectedJPY) {
		t.Errorf("JPY total = %v, want %v", jpyTotal.Amount.Amount, expectedJPY)
	}

	// USD: (2800+500+380+150+850) * 2 = 4680 * 2 = 9360
	usdTotal := findTotalByCurrency(result.TotalAmounts, "USD")
	if usdTotal == nil {
		t.Fatal("USD total not found")
	}
	expectedUSD := decimal.NewFromFloat(4680).Mul(qty)
	if !usdTotal.Amount.Amount.Equal(expectedUSD) {
		t.Errorf("USD total = %v, want %v", usdTotal.Amount.Amount, expectedUSD)
	}

	// 2通貨のみ
	if len(result.TotalAmounts) != 2 {
		t.Errorf("TotalAmounts count = %d, want 2", len(result.TotalAmounts))
	}
}

// ============================================================
// 5. COMPOSITE: OFT+BAF+CAF を1チャージコードにまとめたパッケージ
// ============================================================

func TestCalculateCharges_CompositeOceanFreight(t *testing.T) {
	tariff := newTestTariff("2026 Composite Ocean Freight")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}

	// COMPOSITE: OFT($2,500) + BAF($450) + CAF($80)
	addCompositeLineItem(t, tariff, "OCEAN_FREIGHT_PKG", "FREIGHT_BASIC", oceanTransport, []PricingStrategy{
		&FlatStrategy{Amount: mustMoney(2500.00, "USD")},
		&FlatStrategy{Amount: mustMoney(450.00, "USD")},
		&FlatStrategy{Amount: mustMoney(80.00, "USD")},
	})

	items := simpleCargoItems() // Quantity=2
	req, err := NewCalculationRequest(oceanRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 1 {
		t.Errorf("AppliedCount = %d, want 1", result.AppliedCount())
	}

	pkg := findAppliedByChargeCode(result.AppliedItems, "OCEAN_FREIGHT_PKG")
	if pkg == nil {
		t.Fatal("OCEAN_FREIGHT_PKG not found")
	}

	// CompositeStrategy: 各Stepが Quantity倍 された結果が合算
	// Step1: 2500*2=5000, Step2: 450*2=900, Step3: 80*2=160
	// total = 5000+900+160 = 6060
	expected := decimal.NewFromFloat(6060)
	if !pkg.Amount.Amount.Equal(expected) {
		t.Errorf("OCEAN_FREIGHT_PKG amount = %v, want %v", pkg.Amount.Amount, expected)
	}
	if pkg.Amount.Currency != "USD" {
		t.Errorf("OCEAN_FREIGHT_PKG currency = %q, want USD", pkg.Amount.Currency)
	}
}

// ============================================================
// 6. COMPOSITE: FLAT + Expression混合（航空運賃パッケージ）
// ============================================================

func TestCalculateCharges_CompositeMixedStrategies(t *testing.T) {
	tariff := newTestTariff("2026 Composite Air Rate")

	airTransport := TransportationService{
		OriginID: calcNaritaID, DestinationID: calcLaxAirportID, Mode: shared.ModeAir,
	}

	// COMPOSITE: 基本運賃($500 FLAT) + 重量帯別追加(Expression式)
	// chargeable_weight = max(200, 1.0*1000) = 1000
	// (1000-45)*3.50 = 955*3.50 = 3342.5
	addCompositeLineItem(t, tariff, "AIR_FREIGHT_PKG", "FREIGHT_BASIC", airTransport, []PricingStrategy{
		&FlatStrategy{Amount: mustMoney(500.00, "USD")},
		&ExpressionStrategy{Formula: "max(0, chargeable_weight - 45) * 3.50", Currency: "USD"},
	})

	items := []CargoItem{
		{
			ID:       uuid.New(),
			Quantity: decimal.NewFromInt(1),
			WeightKG: decimal.NewFromInt(200),
			VolumeM3: decimal.NewFromFloat(1.0),
		},
	}
	req, err := NewCalculationRequest(airRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 1 {
		t.Fatalf("AppliedCount = %d, want 1", result.AppliedCount())
	}

	pkg := result.AppliedItems[0]
	// FLAT: 500*1 = 500, Expression: (1000-45)*3.50 = 3342.5 → 合計 3842.5
	expected := decimal.NewFromFloat(3842.5)
	if !pkg.Amount.Amount.Equal(expected) {
		t.Errorf("AIR_FREIGHT_PKG amount = %v, want %v (FLAT:500 + Expression:3342.5)", pkg.Amount.Amount, expected)
	}
}

// ============================================================
// 7. Scope不一致によるスキップ:
//    ルートに含まれないロケーション/区間の費目はSkippedに分類
// ============================================================

func TestCalculateCharges_ScopeFiltering(t *testing.T) {
	tariff := newTestTariff("Scope Filtering Test")

	// ルートに一致するスコープ
	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	tokyoLocation := LocationService{LocationID: calcTokyoPortID, ServiceType: "HANDLING"}

	// ルートに一致しないスコープ
	unmatchedTransport := TransportationService{
		OriginID: calcNaritaID, DestinationID: calcLaxAirportID, Mode: shared.ModeAir,
	}
	unmatchedLocation := LocationService{LocationID: calcShanghaiPortID, ServiceType: "HANDLING"}
	unmatchedCustoms := LocationService{LocationID: calcCustomsOffice, ServiceType: "HANDLING"}

	// 一致する費目
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2500, "USD")
	addFlatLineItem(t, tariff, "THC", "ORIGIN_LOCAL", tokyoLocation, 35000, "JPY")
	// 一致しない費目（航空ルート区間 — 海上ルートには含まれない）
	addFlatLineItem(t, tariff, "AIR_FREIGHT", "FREIGHT_BASIC", unmatchedTransport, 8000, "USD")
	// 一致しない費目（上海港 — ルートに含まれない）
	addFlatLineItem(t, tariff, "SHANGHAI_THC", "ORIGIN_LOCAL", unmatchedLocation, 5000, "CNY")
	// 一致しない費目（通関事務所 — ルートに含まれない）
	addFlatLineItem(t, tariff, "CUSTOMS_EXPORT", "DUTY_TAX", unmatchedCustoms, 11800, "JPY")

	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// 2件適用、3件スキップ
	if result.AppliedCount() != 2 {
		t.Errorf("AppliedCount = %d, want 2", result.AppliedCount())
	}
	if result.SkippedCount() != 3 {
		t.Errorf("SkippedCount = %d, want 3", result.SkippedCount())
	}

	// スキップされた費目のChargeCode検証
	airSkipped := findSkippedByChargeCode(result.SkippedItems, "AIR_FREIGHT")
	if airSkipped == nil {
		t.Error("AIR_FREIGHT should be in skipped items")
	} else if airSkipped.Reason == "" {
		t.Error("skipped AIR_FREIGHT should have a reason")
	}

	shanghaiSkipped := findSkippedByChargeCode(result.SkippedItems, "SHANGHAI_THC")
	if shanghaiSkipped == nil {
		t.Error("SHANGHAI_THC should be in skipped items")
	}

	customsSkipped := findSkippedByChargeCode(result.SkippedItems, "CUSTOMS_EXPORT")
	if customsSkipped == nil {
		t.Error("CUSTOMS_EXPORT should be in skipped items")
	}

	// 適用された費目の通貨別合計（スキップ分は含まれない）
	usdTotal := findTotalByCurrency(result.TotalAmounts, "USD")
	if usdTotal == nil {
		t.Fatal("USD total not found")
	}
	// OFT: 2500 * 2 = 5000
	if !usdTotal.Amount.Amount.Equal(decimal.NewFromFloat(5000)) {
		t.Errorf("USD total = %v, want 5000", usdTotal.Amount.Amount)
	}

	// CNYはスキップされたので合計に含まれない
	cnyTotal := findTotalByCurrency(result.TotalAmounts, "CNY")
	if cnyTotal != nil {
		t.Errorf("CNY total should not exist, got %v", cnyTotal.Amount.Amount)
	}
}

// ============================================================
// 8. 全費目スキップ: ルートと完全に不一致なTariff
// ============================================================

func TestCalculateCharges_AllSkipped(t *testing.T) {
	tariff := newTestTariff("Unmatched Tariff")

	// 航空区間のみの料金表に対して海上ルートで計算
	airTransport := TransportationService{
		OriginID: calcNaritaID, DestinationID: calcLaxAirportID, Mode: shared.ModeAir,
	}
	naritaLocation := LocationService{LocationID: calcNaritaID, ServiceType: "HANDLING"}

	addFlatLineItem(t, tariff, "AIR_FREIGHT", "FREIGHT_BASIC", airTransport, 5000, "USD")
	addFlatLineItem(t, tariff, "AWB_FEE", "ORIGIN_LOCAL", naritaLocation, 3000, "JPY")

	// 海上ルートで計算→全スキップ
	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 0 {
		t.Errorf("AppliedCount = %d, want 0", result.AppliedCount())
	}
	if result.SkippedCount() != 2 {
		t.Errorf("SkippedCount = %d, want 2", result.SkippedCount())
	}
	if result.HasAppliedItems() {
		t.Error("HasAppliedItems should be false when all items are skipped")
	}
	if len(result.TotalAmounts) != 0 {
		t.Errorf("TotalAmounts should be empty, got %d entries", len(result.TotalAmounts))
	}
}

// ============================================================
// 9. エラーケース: LineItemsが空のTariff
// ============================================================

func TestCalculateCharges_EmptyLineItems(t *testing.T) {
	tariff := newTestTariff("Empty Tariff")
	// LineItemsを追加しない

	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	_, err = tariff.CalculateCharges(*req)
	if err == nil {
		t.Fatal("expected error for empty line items, got nil")
	}
	if !shared.IsCode(err, shared.ErrBusinessRuleViolation) {
		t.Errorf("error code = %v, want BUSINESS_RULE_VIOLATION", err)
	}
}

// ============================================================
// 10. Quantity乗算の検証: コンテナ数を変えた場合の金額変動
// ============================================================

func TestCalculateCharges_QuantityMultiplication(t *testing.T) {
	tariff := newTestTariff("Quantity Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2500.00, "USD")

	tests := []struct {
		name     string
		quantity int64
		expected float64
	}{
		{"1本", 1, 2500.00},
		{"2本", 2, 5000.00},
		{"5本", 5, 12500.00},
		{"10本", 10, 25000.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []CargoItem{
				{
					ID:       uuid.New(),
					Quantity: decimal.NewFromInt(tt.quantity),
					WeightKG: decimal.NewFromInt(10000 * tt.quantity),
					VolumeM3: decimal.NewFromInt(15 * tt.quantity),
				},
			}
			req, err := NewCalculationRequest(oceanRoute(), items, simpleConditions())
			if err != nil {
				t.Fatalf("NewCalculationRequest failed: %v", err)
			}

			result, err := tariff.CalculateCharges(*req)
			if err != nil {
				t.Fatalf("CalculateCharges failed: %v", err)
			}

			oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
			if oft == nil {
				t.Fatal("OFT not found")
			}
			expected := decimal.NewFromFloat(tt.expected)
			if !oft.Amount.Amount.Equal(expected) {
				t.Errorf("OFT amount = %v, want %v", oft.Amount.Amount, expected)
			}
		})
	}
}

// ============================================================
// 11. Summary直接指定（概算用ファクトリ）での計算
// ============================================================

func TestCalculateCharges_WithSummaryDirect(t *testing.T) {
	tariff := newTestTariff("Summary Direct Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2500.00, "USD")
	addFlatLineItem(t, tariff, "BAF", "SURCHARGE_FUEL", oceanTransport, 450.00, "USD")

	// CargoItemsではなくSummaryを直接指定
	summary := CargoSummary{
		TotalQuantity:      decimal.NewFromInt(3),
		TotalWeightKG:      decimal.NewFromInt(30000),
		TotalVolumeM3:      decimal.NewFromInt(45),
		ChargeableWeightKG: decimal.NewFromInt(45000), // 容積重量が勝つ
	}
	req, err := NewCalculationRequestWithSummary(oceanRoute(), summary, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequestWithSummary failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 2 {
		t.Errorf("AppliedCount = %d, want 2", result.AppliedCount())
	}

	// Quantity=3 なので単価×3
	oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
	if oft == nil {
		t.Fatal("OFT not found")
	}
	expected := decimal.NewFromFloat(7500.00) // 2500 * 3
	if !oft.Amount.Amount.Equal(expected) {
		t.Errorf("OFT amount = %v, want %v", oft.Amount.Amount, expected)
	}
}

// ============================================================
// 12. ContainerRequirements指定時のQuantity変換
//    ContainerRequirementsがある場合、Quantityはコンテナ合計数になる
// ============================================================

func TestCalculateCharges_ContainerRequirementsQuantity(t *testing.T) {
	tariff := newTestTariff("Container Quantity Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 2500.00, "USD")

	// ContainerRequirementsを指定 → Quantity = 2 + 1 = 3
	summary := CargoSummary{
		TotalQuantity:      decimal.NewFromInt(100), // CargoItem由来の数量（無視される）
		TotalWeightKG:      decimal.NewFromInt(40000),
		TotalVolumeM3:      decimal.NewFromInt(60),
		ChargeableWeightKG: decimal.NewFromInt(60000),
		ContainerRequirements: []ContainerRequirement{
			{ContainerType: "20DC", Count: decimal.NewFromInt(2)},
			{ContainerType: "40HC", Count: decimal.NewFromInt(1)},
		},
	}
	req, err := NewCalculationRequestWithSummary(oceanRoute(), summary, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequestWithSummary failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
	if oft == nil {
		t.Fatal("OFT not found")
	}
	// Quantity = 2 + 1 = 3 → 2500 * 3 = 7500
	expected := decimal.NewFromFloat(7500.00)
	if !oft.Amount.Amount.Equal(expected) {
		t.Errorf("OFT amount = %v, want %v (2500 * 3 containers)", oft.Amount.Amount, expected)
	}
}

// ============================================================
// 13. 通関・倉庫フル料金表: Door-to-Doorルートでの部分一致
//    通関事務所やCFS倉庫はルートのSegment端点に含まれない場合スキップ
// ============================================================

func TestCalculateCharges_CustomsAndWarehouse_PartialMatch(t *testing.T) {
	tariff := newTestTariff("Customs + Warehouse Mixed")

	// 通関料金（customsOffice）
	customsLocation := LocationService{LocationID: calcCustomsOffice, ServiceType: "HANDLING"}
	addFlatLineItem(t, tariff, "CUSTOMS_EXPORT", "DUTY_TAX", customsLocation, 11800, "JPY")

	// CFS倉庫料金（cfsWarehouseID）
	cfsLocation := LocationService{LocationID: calcCfsWarehouseID, ServiceType: "STORAGE"}
	addFlatLineItem(t, tariff, "CFS_CHARGE", "ORIGIN_LOCAL", cfsLocation, 35000, "JPY")

	// 東京港のTHC（ルートに含まれるのでApplied）
	tokyoLocation := LocationService{LocationID: calcTokyoPortID, ServiceType: "HANDLING"}
	addFlatLineItem(t, tariff, "THC", "ORIGIN_LOCAL", tokyoLocation, 35000, "JPY")

	// Door-to-Doorルート: factory→tokyoPort→LA→warehouse
	// customsOfficeとcfsWarehouseIDはSegment端点に含まれない
	req, err := NewCalculationRequest(doorToDoorRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// THCのみ適用、通関とCFSはスキップ
	if result.AppliedCount() != 1 {
		t.Errorf("AppliedCount = %d, want 1", result.AppliedCount())
	}
	if result.SkippedCount() != 2 {
		t.Errorf("SkippedCount = %d, want 2", result.SkippedCount())
	}

	thc := findAppliedByChargeCode(result.AppliedItems, "THC")
	if thc == nil {
		t.Error("THC should be applied (Tokyo Port is in route)")
	}

	customsSkipped := findSkippedByChargeCode(result.SkippedItems, "CUSTOMS_EXPORT")
	if customsSkipped == nil {
		t.Error("CUSTOMS_EXPORT should be skipped (customs office not in route)")
	}
	cfsSkipped := findSkippedByChargeCode(result.SkippedItems, "CFS_CHARGE")
	if cfsSkipped == nil {
		t.Error("CFS_CHARGE should be skipped (CFS warehouse not in route)")
	}
}

// ============================================================
// 14. デマレージ/ディテンション: Expression式によるフリータイム超過計算
//    detention_days属性が未設定の場合、式はエラーになるためスキップ可能性あり
// ============================================================

func TestCalculateCharges_DemurrageDetention(t *testing.T) {
	tariff := newTestTariff("Demurrage & Detention")

	laStorage := LocationService{LocationID: calcLosAngelesID, ServiceType: "STORAGE"}
	addExprLineItem(t, tariff, "DEMURRAGE", "DEST_LOCAL", laStorage,
		"max(0, detention_days - 4) * 150", "USD")
	addExprLineItem(t, tariff, "DETENTION", "DEST_LOCAL", laStorage,
		"max(0, detention_days - 7) * 100", "USD")

	// detention_daysを含むConditionsを作成
	conditions := CalculationConditions{
		Incoterms: "FOB",
		Attributes: map[string]string{
			"detention_days": "10", // 10日間の滞留
		},
	}
	// 海上ルート（LA港を含む→LOCATION scopeに一致）
	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), conditions)
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// LA港がルートに含まれるため両方適用される
	if result.AppliedCount() != 2 {
		t.Errorf("AppliedCount = %d, want 2", result.AppliedCount())
	}

	dem := findAppliedByChargeCode(result.AppliedItems, "DEMURRAGE")
	if dem == nil {
		t.Fatal("DEMURRAGE not found")
	}
	if dem.Category != "DEST_LOCAL" {
		t.Errorf("DEMURRAGE category = %q, want DEST_LOCAL", dem.Category)
	}
	// DEMURRAGE: max(0, 10 - 4) * 150 = 6 * 150 = 900
	expectedDem := decimal.NewFromFloat(900.0)
	if !dem.Amount.Amount.Equal(expectedDem) {
		t.Errorf("DEMURRAGE amount = %v, want %v (max(0, 10-4)*150)", dem.Amount.Amount, expectedDem)
	}

	det := findAppliedByChargeCode(result.AppliedItems, "DETENTION")
	if det == nil {
		t.Fatal("DETENTION not found")
	}
	// DETENTION: max(0, 10 - 7) * 100 = 3 * 100 = 300
	expectedDet := decimal.NewFromFloat(300.0)
	if !det.Amount.Amount.Equal(expectedDet) {
		t.Errorf("DETENTION amount = %v, want %v (max(0, 10-7)*100)", det.Amount.Amount, expectedDet)
	}
}

// ============================================================
// 15. CalculationConditionsの伝播:
//    Incoterms、DesiredShipDateがShipmentContext.Attributesに渡されること
// ============================================================

func TestCalculateCharges_ConditionsPassthrough(t *testing.T) {
	tariff := newTestTariff("Conditions Test")

	// カスタム属性を検証するため、適用可能なスコープで1つだけ追加
	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 1000, "USD")

	shipDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	conditions := CalculationConditions{
		DesiredShipDate:     &shipDate,
		Incoterms:           "CIF",
		SpecialRequirements: []string{"NO_HAZMAT", "TEMPERATURE_CONTROL"},
		Attributes: map[string]string{
			"customKey": "customValue",
		},
	}

	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), conditions)
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	// toShipmentContextの内容を間接的に検証
	ctx := req.toShipmentContext()
	if ctx.Attributes["incoterms"] != "CIF" {
		t.Errorf("incoterms = %v, want CIF", ctx.Attributes["incoterms"])
	}
	if ctx.Attributes["customKey"] != "customValue" {
		t.Errorf("customKey = %v, want customValue", ctx.Attributes["customKey"])
	}
	if _, ok := ctx.Attributes["desiredShipDate"]; !ok {
		t.Error("desiredShipDate should be set in attributes")
	}
	if _, ok := ctx.Attributes["specialRequirements"]; !ok {
		t.Error("specialRequirements should be set in attributes")
	}

	// 計算自体が成功すること
	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}
	if result.AppliedCount() != 1 {
		t.Errorf("AppliedCount = %d, want 1", result.AppliedCount())
	}
}

// ============================================================
// 16. CargoSummary自動計算の検証:
//    複数CargoItemsからSummaryが正しく集計されること
// ============================================================

func TestCalculateCharges_CargoSummaryAutoCalculation(t *testing.T) {
	tariff := newTestTariff("Cargo Summary Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 100, "USD")

	// 複数のCargoItem
	items := []CargoItem{
		{
			ID:       uuid.New(),
			Quantity: decimal.NewFromInt(3),
			WeightKG: decimal.NewFromInt(5000),
			VolumeM3: decimal.NewFromInt(10),
		},
		{
			ID:       uuid.New(),
			Quantity: decimal.NewFromInt(2),
			WeightKG: decimal.NewFromInt(3000),
			VolumeM3: decimal.NewFromInt(8),
		},
	}

	req, err := NewCalculationRequest(oceanRoute(), items, simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	// Summary自動計算の検証
	// TotalQuantity = 3 + 2 = 5
	if !req.Summary.TotalQuantity.Equal(decimal.NewFromInt(5)) {
		t.Errorf("TotalQuantity = %v, want 5", req.Summary.TotalQuantity)
	}
	// TotalWeightKG = 5000 + 3000 = 8000
	if !req.Summary.TotalWeightKG.Equal(decimal.NewFromInt(8000)) {
		t.Errorf("TotalWeightKG = %v, want 8000", req.Summary.TotalWeightKG)
	}
	// TotalVolumeM3 = 10 + 8 = 18
	if !req.Summary.TotalVolumeM3.Equal(decimal.NewFromInt(18)) {
		t.Errorf("TotalVolumeM3 = %v, want 18", req.Summary.TotalVolumeM3)
	}
	// ChargeableWeightKG = max(8000, 18*1000) = 18000
	expectedCW := decimal.NewFromInt(18000)
	if !req.Summary.ChargeableWeightKG.Equal(expectedCW) {
		t.Errorf("ChargeableWeightKG = %v, want %v (volumetric weight wins)", req.Summary.ChargeableWeightKG, expectedCW)
	}

	// OFT: 100 * 5(TotalQuantity) = 500
	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}
	oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
	if oft == nil {
		t.Fatal("OFT not found")
	}
	if !oft.Amount.Amount.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("OFT amount = %v, want 500 (100 * 5)", oft.Amount.Amount)
	}
}

// ============================================================
// 17. describeScopeForItem: ScopeDescriptionの生成検証
// ============================================================

func TestCalculateCharges_ScopeDescriptions(t *testing.T) {
	tariff := newTestTariff("Scope Description Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	tokyoLocation := LocationService{LocationID: calcTokyoPortID, ServiceType: "HANDLING"}

	addFlatLineItem(t, tariff, "OFT", "FREIGHT_BASIC", oceanTransport, 100, "USD")
	addFlatLineItem(t, tariff, "THC", "ORIGIN_LOCAL", tokyoLocation, 100, "JPY")

	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	// TransportationServiceのdescription
	oft := findAppliedByChargeCode(result.AppliedItems, "OFT")
	if oft == nil {
		t.Fatal("OFT not found")
	}
	if oft.ScopeDescription == "" {
		t.Error("OFT ScopeDescription should not be empty")
	}
	// "Transportation (OCEAN) from 10000000 to 10000000" のようなフォーマットを期待
	if oft.ScopeDescription == "unknown scope" {
		t.Error("OFT ScopeDescription should not be 'unknown scope'")
	}

	// LocationServiceのdescription
	thc := findAppliedByChargeCode(result.AppliedItems, "THC")
	if thc == nil {
		t.Fatal("THC not found")
	}
	if thc.ScopeDescription == "" {
		t.Error("THC ScopeDescription should not be empty")
	}
	if thc.ScopeDescription == "unknown scope" {
		t.Error("THC ScopeDescription should not be 'unknown scope'")
	}
}

// ============================================================
// 18. 大規模料金表テスト: 多数のLineItemを持つ場合の動作確認
// ============================================================

func TestCalculateCharges_LargeTariff(t *testing.T) {
	tariff := newTestTariff("Large Tariff Test")

	oceanTransport := TransportationService{
		OriginID: calcTokyoPortID, DestinationID: calcLosAngelesID, Mode: shared.ModeOcean,
	}
	// 不一致スコープ
	airTransport := TransportationService{
		OriginID: calcNaritaID, DestinationID: calcLaxAirportID, Mode: shared.ModeAir,
	}

	// 50件の適用可能なLineItem
	for i := 0; i < 50; i++ {
		addFlatLineItem(t, tariff, "CHARGE_"+string(rune('A'+i%26)), "FREIGHT_BASIC", oceanTransport, 100, "USD")
	}
	// 30件の不一致LineItem
	for i := 0; i < 30; i++ {
		addFlatLineItem(t, tariff, "AIR_CHARGE_"+string(rune('A'+i%26)), "FREIGHT_BASIC", airTransport, 200, "USD")
	}

	req, err := NewCalculationRequest(oceanRoute(), simpleCargoItems(), simpleConditions())
	if err != nil {
		t.Fatalf("NewCalculationRequest failed: %v", err)
	}

	result, err := tariff.CalculateCharges(*req)
	if err != nil {
		t.Fatalf("CalculateCharges failed: %v", err)
	}

	if result.AppliedCount() != 50 {
		t.Errorf("AppliedCount = %d, want 50", result.AppliedCount())
	}
	if result.SkippedCount() != 30 {
		t.Errorf("SkippedCount = %d, want 30", result.SkippedCount())
	}

	// USD合計: 100 * 2(qty) * 50(items) = 10000
	usdTotal := findTotalByCurrency(result.TotalAmounts, "USD")
	if usdTotal == nil {
		t.Fatal("USD total not found")
	}
	expected := decimal.NewFromFloat(10000)
	if !usdTotal.Amount.Amount.Equal(expected) {
		t.Errorf("USD total = %v, want %v", usdTotal.Amount.Amount, expected)
	}
}

// ============================================================
// 19. NewCalculationRequest バリデーション
// ============================================================

func TestNewCalculationRequest_EmptyItems(t *testing.T) {
	_, err := NewCalculationRequest(oceanRoute(), []CargoItem{}, simpleConditions())
	if err == nil {
		t.Fatal("expected error for empty items, got nil")
	}
}

func TestNewCalculationRequest_InvalidSummary(t *testing.T) {
	// Quantityが0のSummary
	summary := CargoSummary{
		TotalQuantity:      decimal.Zero,
		TotalWeightKG:      decimal.NewFromInt(100),
		TotalVolumeM3:      decimal.NewFromInt(1),
		ChargeableWeightKG: decimal.NewFromInt(100),
	}
	_, err := NewCalculationRequestWithSummary(oceanRoute(), summary, simpleConditions())
	if err == nil {
		t.Fatal("expected error for zero quantity summary, got nil")
	}
}

// ============================================================
// 20. CargoSummary.Validate テスト
// ============================================================

func TestCargoSummary_Validate(t *testing.T) {
	tests := []struct {
		name    string
		summary CargoSummary
		wantErr bool
	}{
		{
			name: "valid",
			summary: CargoSummary{
				TotalQuantity:      decimal.NewFromInt(1),
				TotalWeightKG:      decimal.NewFromInt(100),
				TotalVolumeM3:      decimal.NewFromInt(1),
				ChargeableWeightKG: decimal.NewFromInt(1000),
			},
			wantErr: false,
		},
		{
			name: "zero quantity",
			summary: CargoSummary{
				TotalQuantity: decimal.Zero,
			},
			wantErr: true,
		},
		{
			name: "negative weight",
			summary: CargoSummary{
				TotalQuantity: decimal.NewFromInt(1),
				TotalWeightKG: decimal.NewFromInt(-1),
			},
			wantErr: true,
		},
		{
			name: "negative volume",
			summary: CargoSummary{
				TotalQuantity: decimal.NewFromInt(1),
				TotalWeightKG: decimal.Zero,
				TotalVolumeM3: decimal.NewFromInt(-1),
			},
			wantErr: true,
		},
		{
			name: "negative chargeable weight",
			summary: CargoSummary{
				TotalQuantity:      decimal.NewFromInt(1),
				TotalWeightKG:      decimal.Zero,
				TotalVolumeM3:      decimal.Zero,
				ChargeableWeightKG: decimal.NewFromInt(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.summary.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================
// 21. NewCargoSummaryFromItems: 容積重量が実重量を下回る場合
//    ChargeableWeight = 実重量
// ============================================================

func TestNewCargoSummaryFromItems_ActualWeightWins(t *testing.T) {
	items := []CargoItem{
		{
			ID:       uuid.New(),
			Quantity: decimal.NewFromInt(1),
			WeightKG: decimal.NewFromInt(5000), // 実重量 5000kg
			VolumeM3: decimal.NewFromInt(2),     // 容積重量 2*1000 = 2000kg
		},
	}
	summary := NewCargoSummaryFromItems(items)

	// 実重量(5000) > 容積重量(2000) なので ChargeableWeight = 5000
	if !summary.ChargeableWeightKG.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("ChargeableWeightKG = %v, want 5000 (actual weight wins)", summary.ChargeableWeightKG)
	}
}

func TestNewCargoSummaryFromItems_VolumetricWeightWins(t *testing.T) {
	items := []CargoItem{
		{
			ID:       uuid.New(),
			Quantity: decimal.NewFromInt(1),
			WeightKG: decimal.NewFromInt(500),  // 実重量 500kg
			VolumeM3: decimal.NewFromInt(10),    // 容積重量 10*1000 = 10000kg
		},
	}
	summary := NewCargoSummaryFromItems(items)

	// 容積重量(10000) > 実重量(500) なので ChargeableWeight = 10000
	if !summary.ChargeableWeightKG.Equal(decimal.NewFromInt(10000)) {
		t.Errorf("ChargeableWeightKG = %v, want 10000 (volumetric weight wins)", summary.ChargeableWeightKG)
	}
}
