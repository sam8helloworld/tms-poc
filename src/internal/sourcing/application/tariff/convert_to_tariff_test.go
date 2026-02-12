package tariff

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/shopspring/decimal"
)

// ============================================================
// テストヘルパー
// ============================================================

func newUC() *RegisterTariffUseCase {
	return &RegisterTariffUseCase{}
}

var (
	testContractID = uuid.New()
	testFrom       = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	testTo         = time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)

	// ロケーションID
	tokyoPortID    = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	losAngelesID   = uuid.MustParse("10000000-0000-0000-0000-000000000002")
	naritaID       = uuid.MustParse("10000000-0000-0000-0000-000000000003")
	laxAirportID   = uuid.MustParse("10000000-0000-0000-0000-000000000004")
	warehouseID    = uuid.MustParse("10000000-0000-0000-0000-000000000005")
	factoryID      = uuid.MustParse("10000000-0000-0000-0000-000000000006")
	customsOffice  = uuid.MustParse("10000000-0000-0000-0000-000000000007")
	cfsWarehouseID = uuid.MustParse("10000000-0000-0000-0000-000000000008")
)

// flatItem: FLATプライシングのLineItemを構築するヘルパー
func flatItem(chargeCode, category string, scopeType pricing.ServiceScopeType, scopeAttrs map[string]string, amount float64, currency string) pricing.ParsedLineItem {
	return pricing.ParsedLineItem{
		ChargeCode:        chargeCode,
		Category:          category,
		ServiceScopeType:  scopeType,
		ServiceScopeAttrs: scopeAttrs,
		PricingType:       pricing.PricingFlat,
		PricingAttrs: map[string]any{
			"Amount":   amount,
			"Currency": currency,
		},
	}
}

// exprItem: EXPRESSIONプライシングのLineItemを構築するヘルパー
func exprItem(chargeCode, category string, scopeType pricing.ServiceScopeType, scopeAttrs map[string]string, formula, currency string) pricing.ParsedLineItem {
	return pricing.ParsedLineItem{
		ChargeCode:        chargeCode,
		Category:          category,
		ServiceScopeType:  scopeType,
		ServiceScopeAttrs: scopeAttrs,
		PricingType:       pricing.PricingExpression,
		PricingAttrs: map[string]any{
			"Formula":  formula,
			"Currency": currency,
		},
	}
}

// compositeItem: COMPOSITEプライシングのLineItemを構築するヘルパー
func compositeItem(chargeCode, category string, scopeType pricing.ServiceScopeType, scopeAttrs map[string]string, steps []map[string]any) pricing.ParsedLineItem {
	return pricing.ParsedLineItem{
		ChargeCode:        chargeCode,
		Category:          category,
		ServiceScopeType:  scopeType,
		ServiceScopeAttrs: scopeAttrs,
		PricingType:       pricing.PricingComposite,
		PricingAttrs: map[string]any{
			"Steps": steps,
		},
	}
}

// transportAttrs: TRANSPORTATION scopeの属性を構築するヘルパー
func transportAttrs(originID, destID uuid.UUID, mode string) map[string]string {
	return map[string]string{
		"OriginID":      originID.String(),
		"DestinationID": destID.String(),
		"Mode":          mode,
	}
}

// locationAttrs: LOCATION scopeの属性を構築するヘルパー
func locationAttrs(locationID uuid.UUID, serviceType string) map[string]string {
	return map[string]string{
		"LocationID":  locationID.String(),
		"ServiceType": serviceType,
	}
}

// ============================================================
// 1. 海上輸送 (Ocean Freight) テスト
// ============================================================

func TestConvertToTariff_OceanFreight_FCL(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Japan-US Ocean FCL Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// OFT: 海上運賃本体 (per container)
			flatItem("OFT", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 2500.00, "USD"),
			// BAF: 燃料割増料
			flatItem("BAF", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 450.00, "USD"),
			// LSS: 低硫黄燃料割増料
			flatItem("LSS", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 120.00, "USD"),
			// CAF: 通貨変動割増料
			flatItem("CAF", "SURCHARGE_CCY", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 80.00, "USD"),
			// CIC: コンテナ需給調整料
			flatItem("CIC", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 200.00, "USD"),
			// PSS: 繁忙期割増料
			flatItem("PSS", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 300.00, "USD"),
			// WRS: 戦争危険割増料
			flatItem("WRS", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 50.00, "USD"),
			// Origin THC: ターミナル使用料（出発地）
			flatItem("THC", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 35000, "JPY"),
			// Dest THC: ターミナル使用料（到着地）
			flatItem("THC_DEST", "DEST_LOCAL", pricing.ScopeLocation,
				locationAttrs(losAngelesID, "HANDLING"), 350.00, "USD"),
			// Doc Fee: 書類作成料
			flatItem("DOC_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 5000, "JPY"),
			// Seal Fee: コンテナシール代
			flatItem("SEAL_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 1500, "JPY"),
			// Container Cleaning Fee: コンテナ清掃料
			flatItem("CONTAINER_CLEANING", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 8000, "JPY"),
			// Telex Release Fee: サレンダーB/L発行料
			flatItem("TELEX_RELEASE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 3000, "JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	// 基本プロパティの検証
	if tariff.Name != "2026 Japan-US Ocean FCL Rate" {
		t.Errorf("Name = %q, want %q", tariff.Name, "2026 Japan-US Ocean FCL Rate")
	}
	if tariff.ContractID != testContractID {
		t.Errorf("ContractID = %v, want %v", tariff.ContractID, testContractID)
	}
	if tariff.Version != 1 {
		t.Errorf("Version = %d, want 1", tariff.Version)
	}
	if !tariff.EffectiveDate.From.Equal(testFrom) {
		t.Errorf("EffectiveDate.From = %v, want %v", tariff.EffectiveDate.From, testFrom)
	}
	if !tariff.EffectiveDate.To.Equal(testTo) {
		t.Errorf("EffectiveDate.To = %v, want %v", tariff.EffectiveDate.To, testTo)
	}

	// LineItem数の検証
	if len(tariff.LineItems) != 13 {
		t.Fatalf("LineItems count = %d, want 13", len(tariff.LineItems))
	}

	// OFT LineItemの詳細検証
	oft := tariff.LineItems[0]
	if oft.ChargeCode != "OFT" {
		t.Errorf("OFT ChargeCode = %q, want %q", oft.ChargeCode, "OFT")
	}
	if oft.Category != "FREIGHT_BASIC" {
		t.Errorf("OFT Category = %q, want %q", oft.Category, "FREIGHT_BASIC")
	}
	ts, ok := oft.Scope.(pricing.TransportationService)
	if !ok {
		t.Fatalf("OFT Scope type = %T, want TransportationService", oft.Scope)
	}
	if ts.OriginID != route.LocationID(tokyoPortID) {
		t.Errorf("OFT OriginID = %v, want %v", ts.OriginID, route.LocationID(tokyoPortID))
	}
	if ts.DestinationID != route.LocationID(losAngelesID) {
		t.Errorf("OFT DestinationID = %v, want %v", ts.DestinationID, route.LocationID(losAngelesID))
	}
	if ts.Mode != shared.ModeOcean {
		t.Errorf("OFT Mode = %q, want %q", ts.Mode, shared.ModeOcean)
	}
	flat, ok := oft.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("OFT Logic type = %T, want *FlatStrategy", oft.Logic)
	}
	if !flat.Amount.Amount.Equal(decimal.NewFromFloat(2500.00)) {
		t.Errorf("OFT Amount = %v, want 2500.00", flat.Amount.Amount)
	}
	if flat.Amount.Currency != "USD" {
		t.Errorf("OFT Currency = %q, want %q", flat.Amount.Currency, "USD")
	}

	// Origin THCの検証（LOCATION scope）
	thc := tariff.LineItems[7]
	if thc.ChargeCode != "THC" {
		t.Errorf("THC ChargeCode = %q, want %q", thc.ChargeCode, "THC")
	}
	ls, ok := thc.Scope.(pricing.LocationService)
	if !ok {
		t.Fatalf("THC Scope type = %T, want LocationService", thc.Scope)
	}
	if ls.LocationID != route.LocationID(tokyoPortID) {
		t.Errorf("THC LocationID = %v, want %v", ls.LocationID, route.LocationID(tokyoPortID))
	}
	if ls.ServiceType != "HANDLING" {
		t.Errorf("THC ServiceType = %q, want %q", ls.ServiceType, "HANDLING")
	}
	thcFlat, ok := thc.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("THC Logic type = %T, want *FlatStrategy", thc.Logic)
	}
	if !thcFlat.Amount.Amount.Equal(decimal.NewFromFloat(35000)) {
		t.Errorf("THC Amount = %v, want 35000", thcFlat.Amount.Amount)
	}
	if thcFlat.Amount.Currency != "JPY" {
		t.Errorf("THC Currency = %q, want %q", thcFlat.Amount.Currency, "JPY")
	}
}

// ============================================================
// 2. 航空輸送 (Air Freight) テスト
// ============================================================

func TestConvertToTariff_AirFreight(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 NRT-LAX Air Freight Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// Air Freight Rate: 航空運賃（重量帯別はExpression式で表現）
			exprItem("AIR_FREIGHT", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(naritaID, laxAirportID, "AIR"),
				"weight <= 45 ? weight * 8.50 : weight <= 100 ? weight * 7.20 : weight <= 300 ? weight * 6.00 : weight <= 500 ? weight * 5.50 : weight * 5.00",
				"USD"),
			// FSC: 燃料割増料 (per kg)
			exprItem("FSC", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(naritaID, laxAirportID, "AIR"),
				"chargeable_weight * 1.20",
				"USD"),
			// SSC: 保安料 (per kg)
			exprItem("SSC", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(naritaID, laxAirportID, "AIR"),
				"chargeable_weight * 0.10",
				"USD"),
			// AWB Fee: 航空運送状発行料（固定）
			flatItem("AWB_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(naritaID, "HANDLING"), 3000, "JPY"),
			// Terminal Charge: 空港上屋利用料
			flatItem("TERMINAL_CHARGE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(naritaID, "HANDLING"), 25.00, "USD"),
			// X-Ray Fee: 爆発物検査料
			flatItem("XRAY_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(naritaID, "HANDLING"), 5000, "JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if tariff.Name != "2026 NRT-LAX Air Freight Rate" {
		t.Errorf("Name = %q, want %q", tariff.Name, "2026 NRT-LAX Air Freight Rate")
	}
	if len(tariff.LineItems) != 6 {
		t.Fatalf("LineItems count = %d, want 6", len(tariff.LineItems))
	}

	// Air Freight (EXPRESSION)の検証
	airFreight := tariff.LineItems[0]
	if airFreight.ChargeCode != "AIR_FREIGHT" {
		t.Errorf("AIR_FREIGHT ChargeCode = %q", airFreight.ChargeCode)
	}
	ts, ok := airFreight.Scope.(pricing.TransportationService)
	if !ok {
		t.Fatalf("AIR_FREIGHT Scope type = %T, want TransportationService", airFreight.Scope)
	}
	if ts.Mode != shared.ModeAir {
		t.Errorf("AIR_FREIGHT Mode = %q, want %q", ts.Mode, shared.ModeAir)
	}
	cel, ok := airFreight.Logic.(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("AIR_FREIGHT Logic type = %T, want *ExpressionStrategy", airFreight.Logic)
	}
	if cel.Currency != "USD" {
		t.Errorf("AIR_FREIGHT Currency = %q, want %q", cel.Currency, "USD")
	}
	if cel.Formula == "" {
		t.Error("AIR_FREIGHT Formula should not be empty")
	}

	// FSC (EXPRESSION)の検証
	fsc := tariff.LineItems[1]
	fscCel, ok := fsc.Logic.(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("FSC Logic type = %T, want *ExpressionStrategy", fsc.Logic)
	}
	if fscCel.Formula != "chargeable_weight * 1.20" {
		t.Errorf("FSC Formula = %q, want %q", fscCel.Formula, "chargeable_weight * 1.20")
	}

	// AWB Fee (FLAT, LOCATION scope)の検証
	awb := tariff.LineItems[3]
	ls, ok := awb.Scope.(pricing.LocationService)
	if !ok {
		t.Fatalf("AWB Scope type = %T, want LocationService", awb.Scope)
	}
	if ls.LocationID != route.LocationID(naritaID) {
		t.Errorf("AWB LocationID = %v, want %v", ls.LocationID, route.LocationID(naritaID))
	}
	awbFlat, ok := awb.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("AWB Logic type = %T, want *FlatStrategy", awb.Logic)
	}
	if !awbFlat.Amount.Amount.Equal(decimal.NewFromFloat(3000)) {
		t.Errorf("AWB Amount = %v, want 3000", awbFlat.Amount.Amount)
	}
}

// ============================================================
// 3. 陸上輸送 (Trucking / Drayage) テスト
// ============================================================

func TestConvertToTariff_Drayage(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Tokyo Port Drayage Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// Drayage Fee: ラウンドトリップ運賃
			flatItem("DRAYAGE", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, warehouseID, "TRUCK"), 65000, "JPY"),
			// 3-Axle Surcharge: 3軸シャーシ使用料
			flatItem("3AXLE_SURCHARGE", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, warehouseID, "TRUCK"), 15000, "JPY"),
			// MG Fee: 発電機使用料 (Reeferコンテナ用)
			flatItem("MG_FEE", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, warehouseID, "TRUCK"), 20000, "JPY"),
			// Toll Fee: 高速道路料金
			flatItem("TOLL_FEE", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, warehouseID, "TRUCK"), 5500, "JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if tariff.Name != "2026 Tokyo Port Drayage Rate" {
		t.Errorf("Name = %q", tariff.Name)
	}
	if len(tariff.LineItems) != 4 {
		t.Fatalf("LineItems count = %d, want 4", len(tariff.LineItems))
	}

	// Drayage Fee: TRUCK mode検証
	drayage := tariff.LineItems[0]
	ts, ok := drayage.Scope.(pricing.TransportationService)
	if !ok {
		t.Fatalf("DRAYAGE Scope type = %T, want TransportationService", drayage.Scope)
	}
	if ts.Mode != shared.ModeTruck {
		t.Errorf("DRAYAGE Mode = %q, want %q", ts.Mode, shared.ModeTruck)
	}
	if ts.OriginID != route.LocationID(tokyoPortID) {
		t.Errorf("DRAYAGE OriginID = %v, want %v", ts.OriginID, route.LocationID(tokyoPortID))
	}
	if ts.DestinationID != route.LocationID(warehouseID) {
		t.Errorf("DRAYAGE DestinationID = %v, want %v", ts.DestinationID, route.LocationID(warehouseID))
	}
	drayageFlat, ok := drayage.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("DRAYAGE Logic type = %T, want *FlatStrategy", drayage.Logic)
	}
	if !drayageFlat.Amount.Amount.Equal(decimal.NewFromFloat(65000)) {
		t.Errorf("DRAYAGE Amount = %v, want 65000", drayageFlat.Amount.Amount)
	}
	if drayageFlat.Amount.Currency != "JPY" {
		t.Errorf("DRAYAGE Currency = %q, want %q", drayageFlat.Amount.Currency, "JPY")
	}
}

func TestConvertToTariff_Haulage(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 LCL Haulage Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// Charter Fee: チャーター便運賃（4tトラック固定料金）
			flatItem("CHARTER_4T", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(warehouseID, tokyoPortID, "TRUCK"), 45000, "JPY"),
			// LTL Rate: 混載便運賃（kg単価のExpression式）
			exprItem("LTL_RATE", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(warehouseID, tokyoPortID, "TRUCK"),
				"max(weight_kg, volume_m3 * 280) * 35",
				"JPY"),
			// Waiting Charge: 待機料（30分超過ごとの追加）
			exprItem("WAITING_CHARGE", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(warehouseID, tokyoPortID, "TRUCK"),
				"max(0, waiting_minutes - 30) / 30 * 3000",
				"JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if len(tariff.LineItems) != 3 {
		t.Fatalf("LineItems count = %d, want 3", len(tariff.LineItems))
	}

	// Charter Fee (FLAT)
	charter := tariff.LineItems[0]
	charterFlat, ok := charter.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("CHARTER Logic type = %T, want *FlatStrategy", charter.Logic)
	}
	if !charterFlat.Amount.Amount.Equal(decimal.NewFromFloat(45000)) {
		t.Errorf("CHARTER Amount = %v, want 45000", charterFlat.Amount.Amount)
	}

	// LTL Rate (EXPRESSION)
	ltl := tariff.LineItems[1]
	ltlCel, ok := ltl.Logic.(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("LTL Logic type = %T, want *ExpressionStrategy", ltl.Logic)
	}
	if ltlCel.Formula != "max(weight_kg, volume_m3 * 280) * 35" {
		t.Errorf("LTL Formula = %q", ltlCel.Formula)
	}
	if ltlCel.Currency != "JPY" {
		t.Errorf("LTL Currency = %q, want %q", ltlCel.Currency, "JPY")
	}
}

// ============================================================
// 4. 通関・取扱 (Customs & Handling) テスト
// ============================================================

func TestConvertToTariff_CustomsAndHandling(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Japan Customs Broker Fee",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// Customs Clearance Fee: 通関料（輸出申告）
			flatItem("CUSTOMS_EXPORT", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 11800, "JPY"),
			// Customs Clearance Fee: 通関料（輸入申告）
			flatItem("CUSTOMS_IMPORT", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 11800, "JPY"),
			// Handling Charge: 取扱手数料
			flatItem("HANDLING_CHARGE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 15000, "JPY"),
			// Customs Inspection Fee: 税関検査立会料
			flatItem("CUSTOMS_INSPECTION", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 25000, "JPY"),
			// Food Quarantine Fee: 食品検疫申請料
			flatItem("FOOD_QUARANTINE", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 8000, "JPY"),
			// Other Law Application Fee: 他法令申請料
			flatItem("OTHER_LAW_APP", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(customsOffice, "HANDLING"), 10000, "JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if tariff.Name != "2026 Japan Customs Broker Fee" {
		t.Errorf("Name = %q", tariff.Name)
	}
	if len(tariff.LineItems) != 6 {
		t.Fatalf("LineItems count = %d, want 6", len(tariff.LineItems))
	}

	// すべてLOCATION scopeであることの検証
	for i, item := range tariff.LineItems {
		ls, ok := item.Scope.(pricing.LocationService)
		if !ok {
			t.Errorf("LineItem[%d] Scope type = %T, want LocationService", i, item.Scope)
			continue
		}
		if ls.LocationID != route.LocationID(customsOffice) {
			t.Errorf("LineItem[%d] LocationID = %v, want %v", i, ls.LocationID, route.LocationID(customsOffice))
		}
	}

	// 通関料金額の検証
	customsExport := tariff.LineItems[0]
	flat, ok := customsExport.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("CUSTOMS_EXPORT Logic type = %T, want *FlatStrategy", customsExport.Logic)
	}
	if !flat.Amount.Amount.Equal(decimal.NewFromFloat(11800)) {
		t.Errorf("CUSTOMS_EXPORT Amount = %v, want 11800", flat.Amount.Amount)
	}
}

// ============================================================
// 5. 倉庫 (Warehousing / CFS) テスト
// ============================================================

func TestConvertToTariff_Warehousing(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Tokyo CFS Warehouse Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// CFS Charge: 混載倉庫利用料（RT単位のExpression式）
			exprItem("CFS_CHARGE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "STORAGE"),
				"revenue_ton * 3500",
				"JPY"),
			// Storage Fee: 保管料（日数×容積のExpression式）
			exprItem("STORAGE_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "STORAGE"),
				"storage_days * volume_m3 * 150",
				"JPY"),
			// In/Out Handling Fee: 入出庫作業料（固定）
			flatItem("IN_OUT_HANDLING", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "HANDLING"), 8000, "JPY"),
			// Devanning Fee: コンテナ出し作業料
			flatItem("DEVANNING_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "HANDLING"), 35000, "JPY"),
			// Vanning Fee: コンテナ詰め作業料
			flatItem("VANNING_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "HANDLING"), 35000, "JPY"),
			// Labeling Fee: ラベリング費
			exprItem("LABELING_FEE", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(cfsWarehouseID, "HANDLING"),
				"carton_count * 50",
				"JPY"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if tariff.Name != "2026 Tokyo CFS Warehouse Rate" {
		t.Errorf("Name = %q", tariff.Name)
	}
	if len(tariff.LineItems) != 6 {
		t.Fatalf("LineItems count = %d, want 6", len(tariff.LineItems))
	}

	// CFS Charge (EXPRESSION, STORAGE scope)
	cfs := tariff.LineItems[0]
	cfsLS, ok := cfs.Scope.(pricing.LocationService)
	if !ok {
		t.Fatalf("CFS Scope type = %T, want LocationService", cfs.Scope)
	}
	if cfsLS.ServiceType != "STORAGE" {
		t.Errorf("CFS ServiceType = %q, want %q", cfsLS.ServiceType, "STORAGE")
	}
	cfsCel, ok := cfs.Logic.(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("CFS Logic type = %T, want *ExpressionStrategy", cfs.Logic)
	}
	if cfsCel.Formula != "revenue_ton * 3500" {
		t.Errorf("CFS Formula = %q, want %q", cfsCel.Formula, "revenue_ton * 3500")
	}

	// Devanning Fee (FLAT, HANDLING scope)
	devanning := tariff.LineItems[3]
	devLS, ok := devanning.Scope.(pricing.LocationService)
	if !ok {
		t.Fatalf("DEVANNING Scope type = %T, want LocationService", devanning.Scope)
	}
	if devLS.ServiceType != "HANDLING" {
		t.Errorf("DEVANNING ServiceType = %q, want %q", devLS.ServiceType, "HANDLING")
	}
	devFlat, ok := devanning.Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("DEVANNING Logic type = %T, want *FlatStrategy", devanning.Logic)
	}
	if !devFlat.Amount.Amount.Equal(decimal.NewFromFloat(35000)) {
		t.Errorf("DEVANNING Amount = %v, want 35000", devFlat.Amount.Amount)
	}
}

// ============================================================
// 6. デマレージ/ディテンション (Demurrage/Detention) テスト
// ============================================================

func TestConvertToTariff_DemurrageDetention(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Demurrage and Detention Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// Demurrage: フリータイム超過時の保管延滞料（港内）
			// フリータイム4日、超過後は日額課金
			exprItem("DEMURRAGE", "DEST_LOCAL", pricing.ScopeLocation,
				locationAttrs(losAngelesID, "STORAGE"),
				"max(0, detention_days - 4) * 150",
				"USD"),
			// Detention: フリータイム超過時のコンテナ返却延滞料（港外）
			exprItem("DETENTION", "DEST_LOCAL", pricing.ScopeLocation,
				locationAttrs(losAngelesID, "STORAGE"),
				"max(0, detention_days - 7) * 100",
				"USD"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if len(tariff.LineItems) != 2 {
		t.Fatalf("LineItems count = %d, want 2", len(tariff.LineItems))
	}

	// Demurrage (Expression式でフリータイム超過分を計算)
	demurrage := tariff.LineItems[0]
	if demurrage.ChargeCode != "DEMURRAGE" {
		t.Errorf("ChargeCode = %q, want DEMURRAGE", demurrage.ChargeCode)
	}
	demCel, ok := demurrage.Logic.(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("DEMURRAGE Logic type = %T, want *ExpressionStrategy", demurrage.Logic)
	}
	if demCel.Formula != "max(0, detention_days - 4) * 150" {
		t.Errorf("DEMURRAGE Formula = %q", demCel.Formula)
	}
}

// ============================================================
// 7. フォワーダー All-in (Door-to-Door) テスト
//    複数の輸送モードを含む複合料金表
// ============================================================

func TestConvertToTariff_ForwarderAllIn(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 FWD All-in Japan-US Door-to-Door",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// 陸送（工場→港）
			flatItem("PICKUP_DRAYAGE", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(factoryID, tokyoPortID, "TRUCK"), 55000, "JPY"),
			// 通関（輸出）
			flatItem("CUSTOMS_EXPORT", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 11800, "JPY"),
			// Origin THC
			flatItem("THC_ORIGIN", "ORIGIN_LOCAL", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"), 35000, "JPY"),
			// 海上運賃
			flatItem("OFT", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 2800.00, "USD"),
			// BAF
			flatItem("BAF", "SURCHARGE_FUEL", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"), 500.00, "USD"),
			// Dest THC
			flatItem("THC_DEST", "DEST_LOCAL", pricing.ScopeLocation,
				locationAttrs(losAngelesID, "HANDLING"), 380.00, "USD"),
			// 通関（輸入）
			flatItem("CUSTOMS_IMPORT", "DUTY_TAX", pricing.ScopeLocation,
				locationAttrs(losAngelesID, "HANDLING"), 150.00, "USD"),
			// 陸送（港→倉庫）
			flatItem("DELIVERY_DRAYAGE", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(losAngelesID, warehouseID, "TRUCK"), 850.00, "USD"),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if tariff.Name != "2026 FWD All-in Japan-US Door-to-Door" {
		t.Errorf("Name = %q", tariff.Name)
	}
	if len(tariff.LineItems) != 8 {
		t.Fatalf("LineItems count = %d, want 8", len(tariff.LineItems))
	}

	// 輸送モードの混在を検証
	transportModes := map[shared.TransportMode]int{}
	locationCount := 0
	for _, item := range tariff.LineItems {
		switch scope := item.Scope.(type) {
		case pricing.TransportationService:
			transportModes[scope.Mode]++
		case pricing.LocationService:
			locationCount++
		}
	}

	if transportModes[shared.ModeTruck] != 2 {
		t.Errorf("TRUCK items = %d, want 2", transportModes[shared.ModeTruck])
	}
	if transportModes[shared.ModeOcean] != 2 {
		t.Errorf("OCEAN items = %d, want 2", transportModes[shared.ModeOcean])
	}
	if locationCount != 4 {
		t.Errorf("LOCATION items = %d, want 4", locationCount)
	}
}

// ============================================================
// 8. COMPOSITE (複合料金) テスト
//    OFT + BAF + CAF を合成した海上運賃パッケージ
// ============================================================

func TestConvertToTariff_CompositeOceanFreight(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Composite Ocean Freight",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			// 海上運賃パッケージ: OFT($2,500) + BAF($450) + CAF($80)
			compositeItem("OCEAN_FREIGHT_PKG", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "OCEAN"),
				[]map[string]any{
					{"Type": "FLAT", "Amount": 2500.00, "Currency": "USD"},
					{"Type": "FLAT", "Amount": 450.00, "Currency": "USD"},
					{"Type": "FLAT", "Amount": 80.00, "Currency": "USD"},
				}),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	if len(tariff.LineItems) != 1 {
		t.Fatalf("LineItems count = %d, want 1", len(tariff.LineItems))
	}

	item := tariff.LineItems[0]
	if item.ChargeCode != "OCEAN_FREIGHT_PKG" {
		t.Errorf("ChargeCode = %q, want %q", item.ChargeCode, "OCEAN_FREIGHT_PKG")
	}

	comp, ok := item.Logic.(*pricing.CompositeStrategy)
	if !ok {
		t.Fatalf("Logic type = %T, want *CompositeStrategy", item.Logic)
	}
	if len(comp.Steps) != 3 {
		t.Fatalf("Steps count = %d, want 3", len(comp.Steps))
	}

	// 各StepがFlatStrategyであることを検証
	expectedAmounts := []float64{2500.00, 450.00, 80.00}
	for i, step := range comp.Steps {
		flat, ok := step.(*pricing.FlatStrategy)
		if !ok {
			t.Errorf("step[%d] type = %T, want *FlatStrategy", i, step)
			continue
		}
		if !flat.Amount.Amount.Equal(decimal.NewFromFloat(expectedAmounts[i])) {
			t.Errorf("step[%d] Amount = %v, want %v", i, flat.Amount.Amount, expectedAmounts[i])
		}
		if flat.Amount.Currency != "USD" {
			t.Errorf("step[%d] Currency = %q, want %q", i, flat.Amount.Currency, "USD")
		}
	}
}

func TestConvertToTariff_CompositeMixedStrategies(t *testing.T) {
	uc := newUC()

	// FLAT + EXPRESSION の混合Composite
	// 航空運賃: 基本運賃($500固定) + 重量帯別追加運賃(Expression式)
	data := &pricing.ParsedTariffData{
		TariffName:    "2026 Composite Air Rate",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			compositeItem("AIR_FREIGHT_PKG", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(naritaID, laxAirportID, "AIR"),
				[]map[string]any{
					{"Type": "FLAT", "Amount": 500.00, "Currency": "USD"},
					{"Type": "EXPRESSION", "Formula": "max(0, chargeable_weight - 45) * 3.50", "Currency": "USD"},
				}),
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	comp, ok := tariff.LineItems[0].Logic.(*pricing.CompositeStrategy)
	if !ok {
		t.Fatalf("Logic type = %T, want *CompositeStrategy", tariff.LineItems[0].Logic)
	}
	if len(comp.Steps) != 2 {
		t.Fatalf("Steps count = %d, want 2", len(comp.Steps))
	}

	// Step 0: FlatStrategy
	flat, ok := comp.Steps[0].(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("step[0] type = %T, want *FlatStrategy", comp.Steps[0])
	}
	if !flat.Amount.Amount.Equal(decimal.NewFromFloat(500.00)) {
		t.Errorf("step[0] Amount = %v, want 500.00", flat.Amount.Amount)
	}

	// Step 1: ExpressionStrategy
	cel, ok := comp.Steps[1].(*pricing.ExpressionStrategy)
	if !ok {
		t.Fatalf("step[1] type = %T, want *ExpressionStrategy", comp.Steps[1])
	}
	if cel.Formula != "max(0, chargeable_weight - 45) * 3.50" {
		t.Errorf("step[1] Formula = %q", cel.Formula)
	}
	if cel.Currency != "USD" {
		t.Errorf("step[1] Currency = %q, want %q", cel.Currency, "USD")
	}
}

func TestConvertToTariff_CompositeEmptySteps(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Empty Composite Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			compositeItem("TEST", "FREIGHT_BASIC", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"),
				[]map[string]any{}),
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for empty steps, got nil")
	}
}

func TestConvertToTariff_CompositeMissingStepType(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Missing Step Type Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			compositeItem("TEST", "FREIGHT_BASIC", pricing.ScopeLocation,
				locationAttrs(tokyoPortID, "HANDLING"),
				[]map[string]any{
					{"Amount": 100.0, "Currency": "USD"}, // Type missing
				}),
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing step type, got nil")
	}
}

// ============================================================
// 9. エラーケースのテスト
// ============================================================

func TestConvertToTariff_EmptyTariffName(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems:     []pricing.ParsedLineItem{},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for empty tariff name, got nil")
	}
}

func TestConvertToTariff_InvalidDateRange(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Invalid Date Range Tariff",
		EffectiveFrom: testTo,   // from > to
		EffectiveTo:   testFrom, // swapped
		LineItems:     []pricing.ParsedLineItem{},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for invalid date range, got nil")
	}
}

func TestConvertToTariff_NilContractID(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems:     []pricing.ParsedLineItem{},
	}

	_, err := uc.convertToTariff(data, uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil contract ID, got nil")
	}
}

func TestConvertToTariff_UnsupportedScopeType(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "TEST",
				Category:          "FREIGHT_BASIC",
				ServiceScopeType:  "UNKNOWN_SCOPE",
				ServiceScopeAttrs: map[string]string{},
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": 100.0, "Currency": "USD"},
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for unsupported scope type, got nil")
	}
}

func TestConvertToTariff_UnsupportedPricingType(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "TEST",
				Category:          "FREIGHT_BASIC",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       "UNSUPPORTED",
				PricingAttrs:      map[string]any{},
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for unsupported pricing type, got nil")
	}
}

func TestConvertToTariff_InvalidTransportMode(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			flatItem("OFT", "FREIGHT_BASIC", pricing.ScopeTransportation,
				transportAttrs(tokyoPortID, losAngelesID, "SUBMARINE"), 100.0, "USD"),
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for invalid transport mode, got nil")
	}
}

func TestConvertToTariff_MissingLocationID(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: map[string]string{}, // LocationID missing
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": 100.0, "Currency": "USD"},
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing LocationID, got nil")
	}
}

func TestConvertToTariff_MissingOriginID(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:       "OFT",
				Category:         "FREIGHT_BASIC",
				ServiceScopeType: pricing.ScopeTransportation,
				ServiceScopeAttrs: map[string]string{
					"DestinationID": losAngelesID.String(),
					"Mode":          "OCEAN",
					// OriginID missing
				},
				PricingType:  pricing.PricingFlat,
				PricingAttrs: map[string]any{"Amount": 100.0, "Currency": "USD"},
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing OriginID, got nil")
	}
}

func TestConvertToTariff_MissingAmount(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Currency": "USD"}, // Amount missing
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing Amount, got nil")
	}
}

func TestConvertToTariff_MissingCurrency(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": 100.0}, // Currency missing
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing Currency, got nil")
	}
}

func TestConvertToTariff_MissingFormula(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Test Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "AIR_FREIGHT",
				Category:          "FREIGHT_BASIC",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(naritaID, "HANDLING"),
				PricingType:       pricing.PricingExpression,
				PricingAttrs:      map[string]any{"Currency": "USD"}, // Formula missing
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for missing Formula, got nil")
	}
}

// ============================================================
// 10. Amount型変換のテスト
// ============================================================

func TestConvertToTariff_AmountAsInt(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Amount Int Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": 35000, "Currency": "JPY"},
			},
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	flat, ok := tariff.LineItems[0].Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("Logic type = %T, want *FlatStrategy", tariff.LineItems[0].Logic)
	}
	if !flat.Amount.Amount.Equal(decimal.NewFromInt(35000)) {
		t.Errorf("Amount = %v, want 35000", flat.Amount.Amount)
	}
}

func TestConvertToTariff_AmountAsString(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Amount String Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": "2500.50", "Currency": "USD"},
			},
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	flat, ok := tariff.LineItems[0].Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("Logic type = %T, want *FlatStrategy", tariff.LineItems[0].Logic)
	}
	expected, _ := decimal.NewFromString("2500.50")
	if !flat.Amount.Amount.Equal(expected) {
		t.Errorf("Amount = %v, want 2500.50", flat.Amount.Amount)
	}
}

func TestConvertToTariff_AmountAsDecimal(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Amount Decimal Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "OFT",
				Category:          "FREIGHT_BASIC",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": decimal.NewFromFloat(1234.56), "Currency": "USD"},
			},
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	flat, ok := tariff.LineItems[0].Logic.(*pricing.FlatStrategy)
	if !ok {
		t.Fatalf("Logic type = %T, want *FlatStrategy", tariff.LineItems[0].Logic)
	}
	if !flat.Amount.Amount.Equal(decimal.NewFromFloat(1234.56)) {
		t.Errorf("Amount = %v, want 1234.56", flat.Amount.Amount)
	}
}

func TestConvertToTariff_AmountInvalidType(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Amount Invalid Type Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:        "THC",
				Category:          "ORIGIN_LOCAL",
				ServiceScopeType:  pricing.ScopeLocation,
				ServiceScopeAttrs: locationAttrs(tokyoPortID, "HANDLING"),
				PricingType:       pricing.PricingFlat,
				PricingAttrs:      map[string]any{"Amount": true, "Currency": "USD"}, // bool is invalid
			},
		},
	}

	_, err := uc.convertToTariff(data, testContractID)
	if err == nil {
		t.Fatal("expected error for invalid Amount type, got nil")
	}
}

// ============================================================
// 11. LocationService デフォルトServiceType テスト
// ============================================================

func TestConvertToTariff_LocationServiceDefaultServiceType(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Default ServiceType Test",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems: []pricing.ParsedLineItem{
			{
				ChargeCode:       "THC",
				Category:         "ORIGIN_LOCAL",
				ServiceScopeType: pricing.ScopeLocation,
				ServiceScopeAttrs: map[string]string{
					"LocationID": tokyoPortID.String(),
					// ServiceType not specified → default "HANDLING"
				},
				PricingType:  pricing.PricingFlat,
				PricingAttrs: map[string]any{"Amount": 100.0, "Currency": "USD"},
			},
		},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	ls, ok := tariff.LineItems[0].Scope.(pricing.LocationService)
	if !ok {
		t.Fatalf("Scope type = %T, want LocationService", tariff.LineItems[0].Scope)
	}
	if ls.ServiceType != "HANDLING" {
		t.Errorf("ServiceType = %q, want %q (default)", ls.ServiceType, "HANDLING")
	}
}

// ============================================================
// 12. LineItemなし（空のTariff）テスト
// ============================================================

func TestConvertToTariff_NoLineItems(t *testing.T) {
	uc := newUC()

	data := &pricing.ParsedTariffData{
		TariffName:    "Empty Tariff",
		EffectiveFrom: testFrom,
		EffectiveTo:   testTo,
		LineItems:     []pricing.ParsedLineItem{},
	}

	tariff, err := uc.convertToTariff(data, testContractID)
	if err != nil {
		t.Fatalf("convertToTariff failed: %v", err)
	}

	// LineItemなしでもTariff自体は生成可能（Validate()で後からチェック）
	if tariff.Name != "Empty Tariff" {
		t.Errorf("Name = %q, want %q", tariff.Name, "Empty Tariff")
	}
	if len(tariff.LineItems) != 0 {
		t.Errorf("LineItems count = %d, want 0", len(tariff.LineItems))
	}

	// ただしValidate()はエラーになるはず
	if err := tariff.Validate(); err == nil {
		t.Error("expected Validate() to fail for tariff with no line items")
	}
}
