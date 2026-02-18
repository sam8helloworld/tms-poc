package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	networkpersistence "github.com/sam8helloworld/tms-poc/internal/network/infrastructure/persistence"
	rateapp "github.com/sam8helloworld/tms-poc/internal/rate/application/rate"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	bidapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/bid"
	tariffapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/tariff"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	sourcingpersistence "github.com/sam8helloworld/tms-poc/internal/sourcing/infrastructure/persistence"
)

// RateSimulationScenario: マルチプロバイダー対応レートシミュレーションシナリオ
// 複数業者がセグメントごとに担当する輸送ルートのコストを見積もる業務フロー
type RateSimulationScenario struct{}

func (s *RateSimulationScenario) Name() string { return "rate-simulation" }
func (s *RateSimulationScenario) Description() string {
	return "Multi-provider rate simulation: segment-level cost estimation with multiple providers per route"
}

func (s *RateSimulationScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== Multi-Provider Rate Simulation Scenario ===")
	fmt.Println()

	// Step 1: ロケーション作成
	locations, err := s.step1CreateLocations(ctx, pool)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	// Step 2: ベンダー作成
	vendors, err := s.step2CreateVendors(ctx, pool)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}

	shipperID := uuid.New()
	bidRequestID := uuid.New()

	// Step 3: DRAFT契約作成（4社分）
	contracts, err := s.step3CreateContracts(ctx, deps, vendors, shipperID, bidRequestID)
	if err != nil {
		return fmt.Errorf("step 3: %w", err)
	}

	// Step 4: Tariff登録（各ベンダーの担当セグメント分）
	err = s.step4RegisterTariffs(ctx, deps, contracts, vendors, locations)
	if err != nil {
		return fmt.Errorf("step 4: %w", err)
	}

	// Step 5: 全契約をCONTRACTED化
	err = s.step5AwardAllContracts(ctx, deps, contracts, vendors)
	if err != nil {
		return fmt.Errorf("step 5: %w", err)
	}

	// Step 6: DRAFTレート作成
	rateOutput, err := s.step6CreateRate(ctx, deps, shipperID)
	if err != nil {
		return fmt.Errorf("step 6: %w", err)
	}

	// Step 7: 全契約のTariffをレートに適用
	err = s.step7ApplyAllContractsToRate(ctx, deps, rateOutput.RateID, contracts)
	if err != nil {
		return fmt.Errorf("step 7: %w", err)
	}

	// Step 8: レートACTIVE化
	err = s.step8ActivateRate(ctx, deps, rateOutput.RateID)
	if err != nil {
		return fmt.Errorf("step 8: %w", err)
	}

	// Step 9: シミュレーション実行＋詳細結果表示
	err = s.step9Simulate(ctx, deps, rateOutput.RateID, locations, vendors)
	if err != nil {
		return fmt.Errorf("step 9: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// ====== Location indexes ======
const (
	locTokyoCY      = 0
	locTokyoPort    = 1
	locShanghaiPort = 2
	locShanghaiCFS  = 3
	locSingapore    = 4
	locBangkok      = 5
)

// ====== Vendor indexes ======
const (
	vendorJPDrayage    = 0
	vendorOceanCarrier = 1
	vendorCNHandler    = 2
	vendorGlobalFWD    = 3
)

// ====== Step 1: ロケーション作成 ======
func (s *RateSimulationScenario) step1CreateLocations(ctx context.Context, pool *pgxpool.Pool) ([]locationInfo, error) {
	fmt.Println("[Step 1] Creating 6 locations...")

	repo := networkpersistence.NewPostgresLocationRepo(pool)

	defs := []struct {
		name    string
		code    string
		country string
		locType shared.LocationType
	}{
		{"Tokyo CY", "JPTYC", "JP", shared.LocTypeYard},
		{"Tokyo Port", "JPTYO", "JP", shared.LocTypePort},
		{"Shanghai Port", "CNSHA", "CN", shared.LocTypePort},
		{"Shanghai CFS", "CNSFC", "CN", shared.LocTypeWarehouse},
		{"Singapore Port", "SGSIN", "SG", shared.LocTypePort},
		{"Bangkok Port", "THBKK", "TH", shared.LocTypePort},
	}

	locations := make([]locationInfo, 0, len(defs))
	for _, d := range defs {
		code := d.code
		loc := &route.Location{
			ID:          route.LocationID(uuid.New()),
			Name:        d.name,
			UnLocode:    &code,
			CountryCode: d.country,
			Type:        string(d.locType),
		}
		if err := repo.Save(ctx, loc); err != nil {
			return nil, fmt.Errorf("save location %s: %w", d.name, err)
		}
		locations = append(locations, locationInfo{location: loc, name: d.name})
	}

	fmt.Println()
	fmt.Println("  ┌─ [拠点マスタ] 登録済み拠点 ─────────────────────────────")
	fmt.Printf("  │ %-5s  %-18s  %-8s  %s\n", "Code", "Name", "Country", "Type")
	fmt.Println("  │ " + repeatChar('-', 52))
	for _, loc := range locations {
		code := ""
		if loc.location.UnLocode != nil {
			code = *loc.location.UnLocode
		}
		fmt.Printf("  │ %-5s  %-18s  %-8s  %s\n", code, loc.name, loc.location.CountryCode, loc.location.Type)
	}
	fmt.Printf("  └─ 計 %d 件\n", len(locations))
	fmt.Println()

	return locations, nil
}

// ====== Step 2: ベンダー作成 ======
func (s *RateSimulationScenario) step2CreateVendors(ctx context.Context, pool *pgxpool.Pool) ([]vendorInfo, error) {
	fmt.Println("[Step 2] Creating 4 vendors...")

	repo := sourcingpersistence.NewPostgresVendorRepo(pool)

	defs := []struct {
		name       string
		vendorType contract.ProviderType
	}{
		{"JP Drayage Co", contract.ProviderTypeForwarder},
		{"Ocean Carrier Alpha", contract.ProviderTypeCarrier},
		{"CN Handler Co", contract.ProviderTypeForwarder},
		{"Global FWD", contract.ProviderTypeForwarder},
	}

	vendors := make([]vendorInfo, 0, len(defs))
	for _, d := range defs {
		v, err := contract.NewVendor(d.name, d.vendorType)
		if err != nil {
			return nil, err
		}
		if err := repo.Save(ctx, v); err != nil {
			return nil, fmt.Errorf("save vendor %s: %w", d.name, err)
		}
		vendors = append(vendors, vendorInfo{vendor: v, name: d.name})
	}

	fmt.Println()
	fmt.Println("  ┌─ [業者マスタ] 登録済み業者 ─────────────────────────────────────────")
	fmt.Printf("  │ %-20s  %-20s  %s\n", "Name", "Type", "Role")
	fmt.Println("  │ " + repeatChar('-', 65))
	roles := []string{"輸出側国内輸送", "国際海上輸送", "輸入側ハンドリング", "フルサービスFWD"}
	for i, v := range vendors {
		fmt.Printf("  │ %-20s  %-20s  %s\n", v.name, string(v.vendor.Type), roles[i])
	}
	fmt.Printf("  └─ 計 %d 件\n", len(vendors))
	fmt.Println()

	return vendors, nil
}

// ====== Step 3: DRAFT契約作成 ======
func (s *RateSimulationScenario) step3CreateContracts(
	ctx context.Context,
	deps *ScenarioDeps,
	vendors []vendorInfo,
	shipperID uuid.UUID,
	bidRequestID uuid.UUID,
) ([]*bidapp.CreateBidContractOutput, error) {
	fmt.Println("[Step 3] Creating DRAFT contracts (1 per vendor)...")

	contracts := make([]*bidapp.CreateBidContractOutput, 0, len(vendors))
	for _, v := range vendors {
		input := bidapp.CreateBidContractInput{
			BidRequestID: bidRequestID,
			ProviderID:   v.vendor.ID,
			ShipperID:    shipperID,
			ValidFrom:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			ValidTo:      time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		}

		output, err := deps.CreateBidContractUC.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("create contract for %s: %w", v.name, err)
		}
		contracts = append(contracts, output)
		fmt.Printf("  -> %-20s contract %s (DRAFT)\n", v.name+":", output.ContractID.String()[:8])
	}
	fmt.Println()

	return contracts, nil
}

// ====== Step 4: Tariff登録 ======
func (s *RateSimulationScenario) step4RegisterTariffs(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
	locations []locationInfo,
) error {
	fmt.Println("[Step 4] Registering tariffs per vendor...")

	tokyoCY := uuid.UUID(locations[locTokyoCY].location.ID)
	tokyoPort := uuid.UUID(locations[locTokyoPort].location.ID)
	shanghaiPort := uuid.UUID(locations[locShanghaiPort].location.ID)
	shanghaiCFS := uuid.UUID(locations[locShanghaiCFS].location.ID)
	singaporePort := uuid.UUID(locations[locSingapore].location.ID)
	bangkokPort := uuid.UUID(locations[locBangkok].location.ID)

	type tariffDef struct {
		vendorIdx int
		name      string
		items     []tariffapp.LineItemInput
	}

	mkScope := func(originID, destID uuid.UUID, mode shared.TransportMode) (pricing.ServiceScopeType, map[string]string) {
		return pricing.ScopeTransportation, map[string]string{
			"OriginID":      originID.String(),
			"DestinationID": destID.String(),
			"Mode":          string(mode),
		}
	}

	flatItem := func(chargeCode, category string, originID, destID uuid.UUID, mode shared.TransportMode, amount float64) tariffapp.LineItemInput {
		scopeType, scopeAttrs := mkScope(originID, destID, mode)
		return tariffapp.LineItemInput{
			ChargeCode:  chargeCode,
			Category:    category,
			ScopeType:   scopeType,
			ScopeAttrs:  scopeAttrs,
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   decimal.NewFromFloat(amount),
				"Currency": "USD",
			},
		}
	}

	exprItem := func(chargeCode, category string, originID, destID uuid.UUID, mode shared.TransportMode, formula string) tariffapp.LineItemInput {
		scopeType, scopeAttrs := mkScope(originID, destID, mode)
		return tariffapp.LineItemInput{
			ChargeCode:  chargeCode,
			Category:    category,
			ScopeType:   scopeType,
			ScopeAttrs:  scopeAttrs,
			PricingType: pricing.PricingExpression,
			PricingAttrs: map[string]any{
				"Formula":  formula,
				"Currency": "USD",
			},
		}
	}

	compositeItem := func(chargeCode, category string, originID, destID uuid.UUID, mode shared.TransportMode, flatAmount float64, exprFormula string) tariffapp.LineItemInput {
		scopeType, scopeAttrs := mkScope(originID, destID, mode)
		return tariffapp.LineItemInput{
			ChargeCode:  chargeCode,
			Category:    category,
			ScopeType:   scopeType,
			ScopeAttrs:  scopeAttrs,
			PricingType: pricing.PricingComposite,
			PricingAttrs: map[string]any{
				"Steps": []map[string]any{
					{
						"Type":     "FLAT",
						"Amount":   decimal.NewFromFloat(flatAmount),
						"Currency": "USD",
					},
					{
						"Type":     "EXPRESSION",
						"Formula":  exprFormula,
						"Currency": "USD",
					},
				},
			},
		}
	}

	defs := []tariffDef{
		// JP Drayage Co: Tokyo CY -> Tokyo Port (TRUCK) Drayage OFT
		{
			vendorIdx: vendorJPDrayage,
			name:      "JP Drayage Co 2026H1 Drayage Rate",
			items: []tariffapp.LineItemInput{
				flatItem("OFT", "FREIGHT", tokyoCY, tokyoPort, shared.ModeTruck, 450.00),
			},
		},
		// Ocean Carrier Alpha: 2 routes
		//   Tokyo Port -> Shanghai Port (OCEAN): OFT + BAF
		//   Shanghai Port -> Singapore Port (OCEAN): OFT + BAF
		{
			vendorIdx: vendorOceanCarrier,
			name:      "Ocean Carrier Alpha 2026H1 Ocean Rate",
			items: []tariffapp.LineItemInput{
				flatItem("OFT", "FREIGHT", tokyoPort, shanghaiPort, shared.ModeOcean, 1200.00),
				flatItem("BAF", "SURCHARGE", tokyoPort, shanghaiPort, shared.ModeOcean, 350.00),
				flatItem("OFT", "FREIGHT", shanghaiPort, singaporePort, shared.ModeOcean, 800.00),
				flatItem("BAF", "SURCHARGE", shanghaiPort, singaporePort, shared.ModeOcean, 200.00),
			},
		},
		// CN Handler Co: Shanghai Port -> Shanghai CFS (TRUCK): OFT + THC(Expression) + CFS(Composite)
		{
			vendorIdx: vendorCNHandler,
			name:      "CN Handler Co 2026H1 Handling Rate",
			items: []tariffapp.LineItemInput{
				flatItem("OFT", "FREIGHT", shanghaiPort, shanghaiCFS, shared.ModeTruck, 380.00),
				exprItem("THC", "HANDLING", shanghaiPort, shanghaiCFS, shared.ModeTruck, "chargeable_weight * 0.5"),
				compositeItem("CFS", "FREIGHT", shanghaiPort, shanghaiCFS, shared.ModeTruck, 150.00, "weight * 0.02"),
			},
		},
		// Global FWD: Tokyo Port -> Bangkok Port (OCEAN): OFT + BAF + THC + CFS (一括)
		//             Shanghai Port -> Singapore Port (OCEAN): OFT + BAF (backup, やや高め)
		{
			vendorIdx: vendorGlobalFWD,
			name:      "Global FWD 2026H1 Full Service Rate",
			items: []tariffapp.LineItemInput{
				flatItem("OFT", "FREIGHT", tokyoPort, bangkokPort, shared.ModeOcean, 1800.00),
				flatItem("BAF", "SURCHARGE", tokyoPort, bangkokPort, shared.ModeOcean, 450.00),
				exprItem("THC", "HANDLING", tokyoPort, bangkokPort, shared.ModeOcean, "chargeable_weight * 0.6"),
				compositeItem("CFS", "FREIGHT", tokyoPort, bangkokPort, shared.ModeOcean, 180.00, "weight * 0.025"),
				flatItem("OFT", "FREIGHT", shanghaiPort, singaporePort, shared.ModeOcean, 850.00),
				flatItem("BAF", "SURCHARGE", shanghaiPort, singaporePort, shared.ModeOcean, 220.00),
			},
		},
	}

	for _, td := range defs {
		input := tariffapp.RegisterTariffDirectInput{
			ContractID:    contracts[td.vendorIdx].ContractID,
			TariffName:    td.name,
			EffectiveFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EffectiveTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
			LineItems:     td.items,
		}

		output, err := deps.RegisterTariffDirectUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("register tariff for %s: %w", vendors[td.vendorIdx].name, err)
		}
		fmt.Printf("  -> %-20s %d line items registered (tariff: %s)\n",
			vendors[td.vendorIdx].name+":", len(td.items), output.TariffID.String()[:8])
	}
	fmt.Println()

	return nil
}

// ====== Step 5: 全契約をCONTRACTED化 ======
func (s *RateSimulationScenario) step5AwardAllContracts(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
) error {
	fmt.Println("[Step 5] Awarding all contracts (DRAFT -> CONTRACTED)...")

	for i, c := range contracts {
		input := bidapp.AwardBidContractInput{
			ContractID: c.ContractID,
		}
		_, err := deps.AwardBidContractUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("award contract for %s: %w", vendors[i].name, err)
		}
		fmt.Printf("  -> %-20s CONTRACTED\n", vendors[i].name+":")
	}
	fmt.Println()

	return nil
}

// ====== Step 6: DRAFTレート作成 ======
func (s *RateSimulationScenario) step6CreateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	shipperID uuid.UUID,
) (*rateapp.CreateRateOutput, error) {
	fmt.Print("[Step 6] Creating rate \"2026 H1 Multi-Provider Rate\"... ")

	input := rateapp.CreateRateInput{
		ShipperID: shipperID,
		Name:      "2026 H1 Multi-Provider Rate",
		ValidFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ValidTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	}

	output, err := deps.CreateRateUC.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	fmt.Println("done")
	fmt.Println()

	return output, nil
}

// ====== Step 7: 全契約のTariffをレートに適用 ======
func (s *RateSimulationScenario) step7ApplyAllContractsToRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	contracts []*bidapp.CreateBidContractOutput,
) error {
	fmt.Println("[Step 7] Applying all contract tariffs to rate...")

	totalEntries := 0
	for _, c := range contracts {
		input := rateapp.ApplyContractToRateInput{
			RateID:     rateID,
			ContractID: c.ContractID,
		}

		output, err := deps.ApplyContractToRateUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("apply contract %s: %w", c.ContractID.String()[:8], err)
		}
		totalEntries = output.TotalEntryCount
		fmt.Printf("  -> ContractID %s: %d entries applied (total: %d)\n",
			c.ContractID.String()[:8], len(output.AddedEntries), output.TotalEntryCount)
	}

	fmt.Printf("  Total rate entries: %d\n", totalEntries)
	fmt.Println()

	return nil
}

// ====== Step 8: レートACTIVE化 ======
func (s *RateSimulationScenario) step8ActivateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
) error {
	fmt.Print("[Step 8] Activating rate... ")

	input := rateapp.ActivateRateInput{
		RateID: rateID,
	}

	_, err := deps.ActivateRateUC.Execute(ctx, input)
	if err != nil {
		return err
	}

	fmt.Println("done (ACTIVE)")
	fmt.Println()

	return nil
}

// ====== Step 9: シミュレーション実行＋詳細結果表示 ======
func (s *RateSimulationScenario) step9Simulate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	locations []locationInfo,
	vendors []vendorInfo,
) error {
	fmt.Println("[Step 9] Simulating rate cost (cargo: 1x 20DC / 18,000kg / 30m3)...")
	fmt.Println()

	tokyoCY := route.LocationID(locations[locTokyoCY].location.ID)
	tokyoPort := route.LocationID(locations[locTokyoPort].location.ID)
	shanghaiPort := route.LocationID(locations[locShanghaiPort].location.ID)
	shanghaiCFS := route.LocationID(locations[locShanghaiCFS].location.ID)
	singaporePort := route.LocationID(locations[locSingapore].location.ID)
	bangkokPort := route.LocationID(locations[locBangkok].location.ID)

	quantity := decimal.NewFromInt(1)
	weightKG := decimal.NewFromInt(18000)
	volumeM3 := decimal.NewFromInt(30)

	type simRoute struct {
		name     string
		desc     string
		input    rateapp.SimulationRouteInput
	}

	simRoutes := []simRoute{
		// Route A: Tokyo CY -> Shanghai CFS (3 segments, multi-provider)
		{
			name: "Route A",
			desc: "Tokyo CY -> Shanghai CFS (3 segments)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      tokyoCY,
					DestinationID: shanghaiCFS,
					Segments: []route.RouteSegment{
						{OriginLocationID: tokyoCY, DestLocationID: tokyoPort, Mode: shared.ModeTruck},
						{OriginLocationID: tokyoPort, DestLocationID: shanghaiPort, Mode: shared.ModeOcean},
						{OriginLocationID: shanghaiPort, DestLocationID: shanghaiCFS, Mode: shared.ModeTruck},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
		// Route B: Tokyo Port -> Bangkok Port (1 segment, FWD bulk)
		{
			name: "Route B",
			desc: "Tokyo Port -> Bangkok Port (1 segment, FWD bulk)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      tokyoPort,
					DestinationID: bangkokPort,
					Segments: []route.RouteSegment{
						{OriginLocationID: tokyoPort, DestLocationID: bangkokPort, Mode: shared.ModeOcean},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
		// Route C: Shanghai Port -> Singapore Port (1 segment, 2 providers: main + backup)
		{
			name: "Route C",
			desc: "Shanghai Port -> Singapore Port (1 segment, 2 providers)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      shanghaiPort,
					DestinationID: singaporePort,
					Segments: []route.RouteSegment{
						{OriginLocationID: shanghaiPort, DestLocationID: singaporePort, Mode: shared.ModeOcean},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
		// Route D: Bangkok -> Tokyo (unavailable)
		{
			name: "Route D",
			desc: "Bangkok Port -> Tokyo Port (unavailable)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      bangkokPort,
					DestinationID: tokyoPort,
					Segments: []route.RouteSegment{
						{OriginLocationID: bangkokPort, DestLocationID: tokyoPort, Mode: shared.ModeOcean},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
	}

	// Build vendor name lookup
	vendorNames := make(map[uuid.UUID]string)
	for _, v := range vendors {
		vendorNames[v.vendor.ID] = v.name
	}

	// UseCase呼び出し
	conditions := make([]rateapp.SimulationRouteInput, len(simRoutes))
	for i, sr := range simRoutes {
		conditions[i] = sr.input
	}

	input := rateapp.SimulateRateCostInput{
		RateID:          rateID,
		RouteConditions: conditions,
	}

	output, err := deps.SimulateRateCostUC.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("simulate rate cost: %w", err)
	}

	// Location name lookup
	locNames := make(map[route.LocationID]string)
	for _, loc := range locations {
		locNames[loc.location.ID] = loc.name
	}

	fmt.Printf("  Rate: %s (Status: %s)\n", output.RateName, output.RateStatus)
	fmt.Println()

	// 結果表示
	for i, result := range output.RouteResults {
		sr := simRoutes[i]

		if !result.IsAvailable {
			fmt.Printf("  ┌─ [%s] %s ✗ 輸送不可 ─────\n", sr.name, sr.desc)
			fmt.Println("  │ 該当するレートエントリが見つかりません")
			fmt.Println("  │ -> このルートは現在のレートではカバーされていません")
			fmt.Println("  └──────────────────────────────────────────────────────")
			fmt.Println()
			continue
		}

		segCount := len(result.SegmentBreakdown)
		providerCount := countUniqueProviders(result.AppliedCharges)

		fmt.Printf("  ┌─ [%s] %s ✓ 輸送可能 ─────\n", sr.name, sr.desc)
		fmt.Println("  │")

		for _, seg := range result.SegmentBreakdown {
			originName := locNames[seg.OriginID]
			destName := locNames[seg.DestinationID]
			fmt.Printf("  │  [Seg %d] %s -> %s (%s)\n", seg.SegmentIndex+1, originName, destName, seg.Mode)

			if !seg.HasEntries {
				fmt.Println("  │  (no matching entries)")
				fmt.Println("  │")
				continue
			}

			for _, pg := range seg.ProviderCharges {
				provName := vendorNames[pg.ProviderID]
				if provName == "" {
					provName = pg.ProviderID.String()[:8]
				}
				fmt.Printf("  │  ┌ %s\n", provName)

				for _, charge := range pg.Charges {
					fmt.Printf("  │  │  %-5s %-12s $%s %s\n",
						charge.ChargeCode,
						charge.Category,
						charge.Amount.Amount.StringFixed(2),
						charge.Amount.Currency,
					)
				}

				for _, sub := range pg.Subtotal {
					fmt.Printf("  │  └ 小計: $%s %s\n", sub.Amount.Amount.StringFixed(2), sub.Currency)
				}
			}

			if len(seg.SkippedCharges) > 0 {
				fmt.Printf("  │  (skipped: %d charges)\n", len(seg.SkippedCharges))
			}

			fmt.Println("  │")
		}

		fmt.Println("  │  " + repeatChar('-', 40))

		// Route C special: multi-provider on same segment -> show cheapest
		if segCount == 1 && providerCount > 1 {
			// Find cheapest provider
			seg := result.SegmentBreakdown[0]
			var cheapestName string
			var cheapestTotal shared.Money
			var allProvTotal shared.Money
			first := true
			cheapestFirst := true

			for _, pg := range seg.ProviderCharges {
				for _, sub := range pg.Subtotal {
					if cheapestFirst || cheapestTotal.GreaterThan(sub.Amount) {
						cheapestTotal = sub.Amount
						cheapestName = vendorNames[pg.ProviderID]
						cheapestFirst = false
					}
					if first {
						allProvTotal = sub.Amount
						first = false
					} else {
						sum, addErr := allProvTotal.Add(sub.Amount)
						if addErr == nil {
							allProvTotal = sum
						}
					}
				}
			}

			fmt.Printf("  │  最安合計: $%s %s (%s)\n",
				cheapestTotal.Amount.StringFixed(2), cheapestTotal.Currency, cheapestName)
			fmt.Printf("  │  全Provider合計: $%s %s\n",
				allProvTotal.Amount.StringFixed(2), allProvTotal.Currency)
		} else {
			for _, total := range result.TotalAmounts {
				fmt.Printf("  │  ルート合計: $%s %s (%d providers, %d charges)\n",
					total.Amount.Amount.StringFixed(2),
					total.Currency,
					providerCount,
					len(result.AppliedCharges),
				)
			}
		}
		fmt.Println("  └──────────────────────────────────────────────────────")
		fmt.Println()
	}

	// サマリー表示
	availableCount := 0
	for _, r := range output.RouteResults {
		if r.IsAvailable {
			availableCount++
		}
	}
	fmt.Printf("  シミュレーション完了: %d/%d ルートが輸送可能\n", availableCount, len(output.RouteResults))

	return nil
}

// countUniqueProviders: 費目リスト中のユニークプロバイダー数をカウント
func countUniqueProviders(charges []rateapp.SimulatedCharge) int {
	seen := make(map[uuid.UUID]bool)
	for _, c := range charges {
		seen[c.ProviderID] = true
	}
	return len(seen)
}
