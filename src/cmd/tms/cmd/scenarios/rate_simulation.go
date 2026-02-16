package scenarios

import (
	"context"
	"fmt"
	"math/rand"
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

// RateSimulationScenario: レートシミュレーションシナリオ
// ACTIVEレートに対してルート条件でコストをシミュレーションする業務フロー
type RateSimulationScenario struct{}

func (s *RateSimulationScenario) Name() string { return "rate-simulation" }
func (s *RateSimulationScenario) Description() string {
	return "Rate simulation: lookup active rate entries for routes and estimate costs"
}

func (s *RateSimulationScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== Rate Simulation Scenario ===")
	fmt.Println()

	// === Setup（マスターデータ） ===
	locations, err := s.setupLocations(ctx, pool)
	if err != nil {
		return fmt.Errorf("setup locations: %w", err)
	}

	vendors, err := s.setupVendors(ctx, pool)
	if err != nil {
		return fmt.Errorf("setup vendors: %w", err)
	}

	shipperID := uuid.New()
	bidRequestID := uuid.New()

	// ルート定義（3ルート）
	routes := s.defineRoutes()

	// === Business Flow ===

	// Step 1: 各FWDとのDRAFT契約作成
	contracts, err := s.step1CreateContracts(ctx, deps, vendors, shipperID, bidRequestID)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	// Step 2: 各社Tariff登録（OFT + BAF × 3ルート = 6 LineItems）
	tariffMap, err := s.step2RegisterTariffs(ctx, deps, contracts, vendors, locations, routes)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}

	// Step 3: 料金比較・最安業者選定
	winners := s.step3CompareTariffs(vendors, routes, tariffMap)

	// Step 4: 契約Award
	err = s.step4AwardContracts(ctx, deps, contracts, vendors, winners, shipperID)
	if err != nil {
		return fmt.Errorf("step 4: %w", err)
	}

	// Step 5: DRAFTレート作成
	rateOutput, err := s.step5CreateRate(ctx, deps, shipperID)
	if err != nil {
		return fmt.Errorf("step 5: %w", err)
	}

	// Step 6: 契約のTariffをレートに反映
	err = s.step6ApplyContractsToRate(ctx, deps, rateOutput.RateID, winners, routes)
	if err != nil {
		return fmt.Errorf("step 6: %w", err)
	}

	// Step 7: レートACTIVE化
	err = s.step7ActivateRate(ctx, deps, rateOutput.RateID)
	if err != nil {
		return fmt.Errorf("step 7: %w", err)
	}

	// Step 8: レートシミュレーション
	err = s.step8SimulateRateCost(ctx, deps, rateOutput.RateID, locations)
	if err != nil {
		return fmt.Errorf("step 8: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// ====== Setup ======

func (s *RateSimulationScenario) setupLocations(ctx context.Context, pool *pgxpool.Pool) ([]locationInfo, error) {
	fmt.Print("[Setup] Creating 3 locations... ")

	repo := networkpersistence.NewPostgresLocationRepo(pool)

	defs := []struct {
		name    string
		code    string
		country string
	}{
		{"Tokyo", "JPTYO", "JP"},
		{"Shanghai", "CNSHA", "CN"},
		{"Singapore", "SGSIN", "SG"},
	}

	locations := make([]locationInfo, 0, len(defs))
	for _, d := range defs {
		code := d.code
		loc := &route.Location{
			ID:          route.LocationID(uuid.New()),
			Name:        d.name,
			UnLocode:    &code,
			CountryCode: d.country,
			Type:        "PORT",
		}
		if err := repo.Save(ctx, loc); err != nil {
			return nil, fmt.Errorf("save location %s: %w", d.name, err)
		}
		locations = append(locations, locationInfo{location: loc, name: d.name})
	}

	fmt.Println("done")

	fmt.Println()
	fmt.Println("  ┌─ [拠点マスタ] 登録済み拠点 ─────────────────────────────")
	fmt.Printf("  │ %-5s  %-15s  %-8s  %s\n", "Code", "Name", "Country", "Type")
	fmt.Println("  │ " + repeatChar('-', 48))
	for _, loc := range locations {
		code := ""
		if loc.location.UnLocode != nil {
			code = *loc.location.UnLocode
		}
		fmt.Printf("  │ %-5s  %-15s  %-8s  %s\n", code, loc.name, loc.location.CountryCode, loc.location.Type)
	}
	fmt.Printf("  └─ 計 %d 件\n", len(locations))
	fmt.Println()

	return locations, nil
}

func (s *RateSimulationScenario) setupVendors(ctx context.Context, pool *pgxpool.Pool) ([]vendorInfo, error) {
	fmt.Print("[Setup] Creating 2 vendors... ")

	repo := sourcingpersistence.NewPostgresVendorRepo(pool)

	names := []string{"FWD Alpha", "FWD Beta"}
	vendors := make([]vendorInfo, 0, len(names))
	for _, name := range names {
		v, err := contract.NewVendor(name, contract.ProviderTypeForwarder)
		if err != nil {
			return nil, err
		}
		if err := repo.Save(ctx, v); err != nil {
			return nil, fmt.Errorf("save vendor %s: %w", name, err)
		}
		vendors = append(vendors, vendorInfo{vendor: v, name: name})
	}

	fmt.Println("done")

	fmt.Println()
	fmt.Println("  ┌─ [業者マスタ] 登録済み業者 ─────────────────────────────")
	fmt.Printf("  │ %-36s  %-14s  %s\n", "ID", "Name", "Type")
	fmt.Println("  │ " + repeatChar('-', 60))
	for _, v := range vendors {
		fmt.Printf("  │ %-36s  %-14s  %s\n", v.vendor.ID, v.name, string(v.vendor.Type))
	}
	fmt.Printf("  └─ 計 %d 件\n", len(vendors))
	fmt.Println()

	return vendors, nil
}

// simRouteDef: このシナリオ用のルート定義
type simRouteDef struct {
	name      string
	originIdx int
	destIdx   int
	mode      shared.TransportMode
	oftBase   int     // OFT基準価格（USD） - Flat Strategy
	bafBase   int     // BAF基準価格（USD） - Flat Strategy
	thcRate   float64 // THC単価（USD/kg） - Expression Strategy
	cfsBase   int     // CFS基本料（USD） - Composite Strategy (Flat部分)
	cfsRate   float64 // CFS従量単価（USD/kg） - Composite Strategy (Expression部分)
}

func (s *RateSimulationScenario) defineRoutes() []simRouteDef {
	return []simRouteDef{
		{"Tokyo → Shanghai (OCEAN)", 0, 1, shared.ModeOcean, 1200, 350, 0.5, 150, 0.02},
		{"Shanghai → Singapore (OCEAN)", 1, 2, shared.ModeOcean, 800, 200, 0.4, 120, 0.015},
		{"Tokyo → Singapore (OCEAN)", 0, 2, shared.ModeOcean, 1500, 400, 0.6, 180, 0.025},
	}
}

// ====== Business Flow Steps ======

// simTariffResult: Tariff登録結果（OFT + BAF + THC + CFSの4費目）
type simTariffResult struct {
	tariffID      uuid.UUID
	oftLineItemID uuid.UUID
	bafLineItemID uuid.UUID
	thcLineItemID uuid.UUID
	cfsLineItemID uuid.UUID
	oftPrice      decimal.Decimal
	bafPrice      decimal.Decimal
	thcPrice      decimal.Decimal // THC想定額（表示用）
	cfsPrice      decimal.Decimal // CFS想定額（表示用）
}

func (s *RateSimulationScenario) step1CreateContracts(
	ctx context.Context,
	deps *ScenarioDeps,
	vendors []vendorInfo,
	shipperID uuid.UUID,
	bidRequestID uuid.UUID,
) ([]*bidapp.CreateBidContractOutput, error) {
	fmt.Println("[Step 1] Creating bid contracts...")

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
		fmt.Printf("  → %-12s contract %s 作成（DRAFT）\n", v.name+":", output.ContractID.String()[:8])
	}
	fmt.Println()

	return contracts, nil
}

func (s *RateSimulationScenario) step2RegisterTariffs(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
	locations []locationInfo,
	routes []simRouteDef,
) (map[string]map[int]simTariffResult, error) {
	fmt.Printf("[Step 2] Registering tariffs (OFT + BAF + THC + CFS × %d routes per FWD)...\n", len(routes))

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// tariffMap[vendorName][routeIdx] = simTariffResult
	tariffMap := make(map[string]map[int]simTariffResult)

	for vi, v := range vendors {
		tariffMap[v.name] = make(map[int]simTariffResult)

		// 各ルートのOFT + BAF + THC + CFSをLineItemとして生成
		lineItems := make([]tariffapp.LineItemInput, 0, len(routes)*4)
		oftPrices := make([]decimal.Decimal, len(routes))
		bafPrices := make([]decimal.Decimal, len(routes))
		thcPrices := make([]decimal.Decimal, len(routes))
		cfsPrices := make([]decimal.Decimal, len(routes))

		for ri, r := range routes {
			originID := uuid.UUID(locations[r.originIdx].location.ID)
			destID := uuid.UUID(locations[r.destIdx].location.ID)

			// ランダム変動（±20%）
			oftVariation := 0.8 + rng.Float64()*0.4
			bafVariation := 0.8 + rng.Float64()*0.4
			thcVariation := 0.8 + rng.Float64()*0.4
			cfsBaseVariation := 0.8 + rng.Float64()*0.4
			cfsRateVariation := 0.8 + rng.Float64()*0.4

			oftPrices[ri] = decimal.NewFromFloat(float64(r.oftBase) * oftVariation)
			bafPrices[ri] = decimal.NewFromFloat(float64(r.bafBase) * bafVariation)

			// THC想定額: chargeable_weight(=max(18000, 30*1000)=30000) × thcRate × variation
			thcRateActual := r.thcRate * thcVariation
			thcPrices[ri] = decimal.NewFromFloat(30000.0 * thcRateActual) // 想定額（表示用）

			// CFS想定額: base × variation + weight(18000) × cfsRate × variation
			cfsBaseActual := float64(r.cfsBase) * cfsBaseVariation
			cfsRateActual := r.cfsRate * cfsRateVariation
			cfsPrices[ri] = decimal.NewFromFloat(cfsBaseActual + 18000.0*cfsRateActual) // 想定額（表示用）

			scopeAttrs := map[string]string{
				"OriginID":      originID.String(),
				"DestinationID": destID.String(),
				"Mode":          string(r.mode),
			}

			// OFT LineItem (Flat Strategy)
			lineItems = append(lineItems, tariffapp.LineItemInput{
				ChargeCode:  "OFT",
				Category:    "FREIGHT",
				ScopeType:   pricing.ScopeTransportation,
				ScopeAttrs:  scopeAttrs,
				PricingType: pricing.PricingFlat,
				PricingAttrs: map[string]any{
					"Amount":   oftPrices[ri],
					"Currency": "USD",
				},
			})

			// BAF LineItem (Flat Strategy)
			lineItems = append(lineItems, tariffapp.LineItemInput{
				ChargeCode:  "BAF",
				Category:    "SURCHARGE",
				ScopeType:   pricing.ScopeTransportation,
				ScopeAttrs:  scopeAttrs,
				PricingType: pricing.PricingFlat,
				PricingAttrs: map[string]any{
					"Amount":   bafPrices[ri],
					"Currency": "USD",
				},
			})

			// THC LineItem (Expression Strategy)
			// Formula: chargeable_weight * rate
			// chargeable_weight = max(weight, volume*1000) は ExpressionStrategy 内部で自動算出
			thcFormula := fmt.Sprintf("chargeable_weight * %f", thcRateActual)
			lineItems = append(lineItems, tariffapp.LineItemInput{
				ChargeCode:  "THC",
				Category:    "HANDLING",
				ScopeType:   pricing.ScopeTransportation,
				ScopeAttrs:  scopeAttrs,
				PricingType: pricing.PricingExpression,
				PricingAttrs: map[string]any{
					"Formula":  thcFormula,
					"Currency": "USD",
				},
			})

			// CFS LineItem (Composite Strategy)
			// Step1: Flat基本料 + Step2: Expression従量料の合成
			lineItems = append(lineItems, tariffapp.LineItemInput{
				ChargeCode:  "CFS",
				Category:    "FREIGHT",
				ScopeType:   pricing.ScopeTransportation,
				ScopeAttrs:  scopeAttrs,
				PricingType: pricing.PricingComposite,
				PricingAttrs: map[string]any{
					"Steps": []map[string]any{
						{
							"Type":     "FLAT",
							"Amount":   decimal.NewFromFloat(cfsBaseActual),
							"Currency": "USD",
						},
						{
							"Type":     "EXPRESSION",
							"Formula":  fmt.Sprintf("weight * %f", cfsRateActual),
							"Currency": "USD",
						},
					},
				},
			})
		}

		input := tariffapp.RegisterTariffDirectInput{
			ContractID:    contracts[vi].ContractID,
			TariffName:    fmt.Sprintf("%s 2026H1 Ocean Rate Sheet", v.name),
			EffectiveFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EffectiveTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
			LineItems:     lineItems,
		}

		output, err := deps.RegisterTariffDirectUC.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("register tariff for %s: %w", v.name, err)
		}

		// LineItemIDsはOFT, BAF, THC, CFS, OFT, BAF, THC, CFS... の順番
		for ri := range routes {
			tariffMap[v.name][ri] = simTariffResult{
				tariffID:      output.TariffID,
				oftLineItemID: output.LineItemIDs[ri*4],
				bafLineItemID: output.LineItemIDs[ri*4+1],
				thcLineItemID: output.LineItemIDs[ri*4+2],
				cfsLineItemID: output.LineItemIDs[ri*4+3],
				oftPrice:      oftPrices[ri],
				bafPrice:      bafPrices[ri],
				thcPrice:      thcPrices[ri],
				cfsPrice:      cfsPrices[ri],
			}
		}

		fmt.Printf("  → %-12s 1 tariff registered (%d line items)\n", v.name+":", len(lineItems))
	}
	fmt.Println()

	return tariffMap, nil
}

// simRouteWinner: ルートごとの勝者情報
type simRouteWinner struct {
	vendorIdx     int
	vendorName    string
	routeIdx      int
	contractID    uuid.UUID
	oftLineItemID uuid.UUID
	bafLineItemID uuid.UUID
	thcLineItemID uuid.UUID
	cfsLineItemID uuid.UUID
	oftPrice      decimal.Decimal
	bafPrice      decimal.Decimal
	thcPrice      decimal.Decimal
	cfsPrice      decimal.Decimal
	totalPrice    decimal.Decimal // OFT + BAF + THC + CFS
}

func (s *RateSimulationScenario) step3CompareTariffs(
	vendors []vendorInfo,
	routes []simRouteDef,
	tariffMap map[string]map[int]simTariffResult,
) []simRouteWinner {
	fmt.Println("[Step 3] Comparing tariffs per route (OFT + BAF + THC + CFS total)...")
	fmt.Println()

	fmt.Println("  ┌─ [入札比較画面] ルート別料金比較 ──────────────────────────────────────────────────")
	fmt.Printf("  │ %-30s", "Route")
	for _, v := range vendors {
		fmt.Printf("│ %-18s", v.name)
	}
	fmt.Println("│ Winner")
	fmt.Printf("  │ %s", repeatChar('-', 30))
	for range vendors {
		fmt.Printf("+%s", repeatChar('-', 19))
	}
	fmt.Println("+" + repeatChar('-', 14))

	winners := make([]simRouteWinner, 0, len(routes))

	for ri, r := range routes {
		bestIdx := 0
		// Total = OFT + BAF + THC + CFS
		t0 := tariffMap[vendors[0].name][ri]
		bestTotal := t0.oftPrice.Add(t0.bafPrice).Add(t0.thcPrice).Add(t0.cfsPrice)
		bestName := vendors[0].name

		for vi, v := range vendors {
			tr := tariffMap[v.name][ri]
			total := tr.oftPrice.Add(tr.bafPrice).Add(tr.thcPrice).Add(tr.cfsPrice)
			if vi > 0 && total.LessThan(bestTotal) {
				bestIdx = vi
				bestTotal = total
				bestName = v.name
			}
		}

		fmt.Printf("  │ %-30s", r.name)
		for _, v := range vendors {
			tr := tariffMap[v.name][ri]
			total := tr.oftPrice.Add(tr.bafPrice).Add(tr.thcPrice).Add(tr.cfsPrice)
			marker := "  "
			if v.name == bestName {
				marker = "★"
			}
			formatted := fmt.Sprintf("$%s %s", total.StringFixed(0), marker)
			fmt.Printf("│ %-18s", formatted)
		}
		fmt.Printf("│ %s\n", bestName)

		bestResult := tariffMap[bestName][ri]
		winners = append(winners, simRouteWinner{
			vendorIdx:     bestIdx,
			vendorName:    bestName,
			routeIdx:      ri,
			oftLineItemID: bestResult.oftLineItemID,
			bafLineItemID: bestResult.bafLineItemID,
			thcLineItemID: bestResult.thcLineItemID,
			cfsLineItemID: bestResult.cfsLineItemID,
			oftPrice:      bestResult.oftPrice,
			bafPrice:      bestResult.bafPrice,
			thcPrice:      bestResult.thcPrice,
			cfsPrice:      bestResult.cfsPrice,
			totalPrice:    bestTotal,
		})
	}

	fmt.Println("  └─ ★ = 最安値（OFT + BAF + THC + CFS合計）")
	fmt.Println()

	return winners
}

func (s *RateSimulationScenario) step4AwardContracts(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
	winners []simRouteWinner,
	shipperID uuid.UUID,
) error {
	fmt.Println("[Step 4] Awarding contracts...")

	winCount := make(map[int]int)
	for _, w := range winners {
		winCount[w.vendorIdx]++
	}

	for i := range winners {
		winners[i].contractID = contracts[winners[i].vendorIdx].ContractID
	}

	for vi, v := range vendors {
		count := winCount[vi]
		if count == 0 {
			fmt.Printf("  → %-12s 勝利ルートなし（スキップ）\n", v.name+":")
			continue
		}

		input := bidapp.AwardBidContractInput{
			ContractID: contracts[vi].ContractID,
		}
		_, err := deps.AwardBidContractUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("award contract for %s: %w", v.name, err)
		}
		fmt.Printf("  → %-12s DRAFT → CONTRACTED (%d routes won)\n", v.name+":", count)
	}
	fmt.Println()

	return nil
}

func (s *RateSimulationScenario) step5CreateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	shipperID uuid.UUID,
) (*rateapp.CreateRateOutput, error) {
	fmt.Print("[Step 5] Creating rate \"2026 H1 Rate\"... ")

	input := rateapp.CreateRateInput{
		ShipperID: shipperID,
		Name:      "2026 H1 Rate",
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

func (s *RateSimulationScenario) step6ApplyContractsToRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	winners []simRouteWinner,
	routes []simRouteDef,
) error {
	fmt.Println("[Step 6] Applying contracts to rate...")

	// winnersをcontractIDでグループ化
	type contractGroup struct {
		contractID        uuid.UUID
		tariffLineItemIDs []uuid.UUID
	}
	groupMap := make(map[uuid.UUID]*contractGroup)
	var groupOrder []uuid.UUID

	for _, w := range winners {
		if _, ok := groupMap[w.contractID]; !ok {
			groupMap[w.contractID] = &contractGroup{contractID: w.contractID}
			groupOrder = append(groupOrder, w.contractID)
		}

		// OFT, BAF, THC, CFS を追加
		groupMap[w.contractID].tariffLineItemIDs = append(
			groupMap[w.contractID].tariffLineItemIDs,
			w.oftLineItemID, w.bafLineItemID, w.thcLineItemID, w.cfsLineItemID,
		)
	}

	var allAddedEntries []rateapp.AddedEntryDetail
	for _, contractID := range groupOrder {
		group := groupMap[contractID]
		input := rateapp.ApplyContractToRateInput{
			RateID:            rateID,
			ContractID:        contractID,
			TariffLineItemIDs: group.tariffLineItemIDs,
		}

		output, err := deps.ApplyContractToRateUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("apply contract %s: %w", contractID.String()[:8], err)
		}
		allAddedEntries = append(allAddedEntries, output.AddedEntries...)
		fmt.Printf("  → ContractID %s の %d エントリを適用（累計: %d）\n",
			contractID.String()[:8], len(output.AddedEntries), output.TotalEntryCount)
	}

	// レートカード表示
	fmt.Println()
	fmt.Println("  ┌─ [レートカード] ルート別レート一覧 ────────────────────────────────────────────────────")
	fmt.Printf("  │ %-30s│ %-12s│ %-8s│ %s\n", "Route", "Provider", "Charge", "UnitPrice")
	fmt.Printf("  │%s┼%s┼%s┼%s\n", repeatChar('-', 31), repeatChar('-', 13), repeatChar('-', 9), repeatChar('-', 16))

	for _, w := range winners {
		routeName := routes[w.routeIdx].name
		for _, entry := range allAddedEntries {
			// OFT, BAF, THC, CFS のいずれかであれば表示
			desc := "Unknown"
			if entry.TariffLineItemID == w.oftLineItemID {
				desc = "OFT"
			} else if entry.TariffLineItemID == w.bafLineItemID {
				desc = "BAF"
			} else if entry.TariffLineItemID == w.thcLineItemID {
				desc = "THC"
			} else if entry.TariffLineItemID == w.cfsLineItemID {
				desc = "CFS"
			}

			if desc != "Unknown" {
				// UnitPriceは、FLATの場合はAmountそのまま、Expression/Compositeの場合は表示に工夫が必要だが、
				// ここではAddedEntryDetailのUnitPriceをそのまま表示する (Compositeなどはダミー値が入る可能性があるが、RateEntryには保存される)
				// ※ 本来はPricingStrategyごとに表示を変えるべきだが、簡単のためAmountを表示
				fmt.Printf("  │ %-30s│ %-12s│ %-8s│ $%s %s\n",
					truncate(routeName, 30),
					truncate(w.vendorName, 12),
					entry.ChargeCode,
					entry.UnitPrice.Amount.StringFixed(2),
					entry.UnitPrice.Currency,
				)
			}
		}
	}
	fmt.Printf("  └─ 計 %d エントリ（OFT+BAF+THC+CFS × %d ルート）\n", len(allAddedEntries), len(winners))

	fmt.Println()

	return nil
}

func (s *RateSimulationScenario) step7ActivateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
) error {
	fmt.Print("[Step 7] Activating rate... ")

	input := rateapp.ActivateRateInput{
		RateID: rateID,
	}

	output, err := deps.ActivateRateUC.Execute(ctx, input)
	if err != nil {
		return err
	}

	fmt.Println("done")

	rate, err := deps.RateQuery.GetRate(ctx, rateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レート管理画面] レート詳細（有効化後） ─────────────────────")
		fmt.Printf("  │  ID     : %s\n", rate.ID.String()[:8]+"...")
		fmt.Printf("  │  Name   : %s\n", rate.Name)
		fmt.Printf("  │  Status : %s  ← シミュレーション利用可能\n", string(rate.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			rate.ValidFrom.Time.Format("2006-01-02"),
			rate.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Printf("  │  Entries: %d 件\n", output.EntryCount)
		fmt.Println("  └─ ACTIVE = コストシミュレーションに利用可能な状態")
		fmt.Println()
	}

	return nil
}

func (s *RateSimulationScenario) step8SimulateRateCost(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	locations []locationInfo,
) error {
	fmt.Println("[Step 8] Simulating rate cost for transport routes...")
	fmt.Println("  ※ 輸送前のコスト見積もりを実施します（貨物条件: 1本 20DC / 18,000kg / 30m³）")
	fmt.Println()

	tokyoID := route.LocationID(locations[0].location.ID)
	shanghaiID := route.LocationID(locations[1].location.ID)
	singaporeID := route.LocationID(locations[2].location.ID)

	// 貨物条件（全ルート共通: 20DCコンテナ1本、18,000kg、30m³）
	quantity := decimal.NewFromInt(1)
	weightKG := decimal.NewFromInt(18000)
	volumeM3 := decimal.NewFromInt(30)

	// シミュレーション対象ルート（2つは登録済み、1つは未登録）
	type simRoute struct {
		name  string
		input rateapp.SimulationRouteInput
	}

	simRoutes := []simRoute{
		{
			name: "Tokyo → Shanghai (OCEAN)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      tokyoID,
					DestinationID: shanghaiID,
					Segments: []route.RouteSegment{
						{
							OriginLocationID: tokyoID,
							DestLocationID:   shanghaiID,
							Mode:             shared.ModeOcean,
						},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
		{
			name: "Tokyo → Singapore (OCEAN)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      tokyoID,
					DestinationID: singaporeID,
					Segments: []route.RouteSegment{
						{
							OriginLocationID: tokyoID,
							DestLocationID:   singaporeID,
							Mode:             shared.ModeOcean,
						},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
		{
			name: "Singapore → Tokyo (OCEAN)",
			input: rateapp.SimulationRouteInput{
				Route: route.PhysicalRoute{
					OriginID:      singaporeID,
					DestinationID: tokyoID,
					Segments: []route.RouteSegment{
						{
							OriginLocationID: singaporeID,
							DestLocationID:   tokyoID,
							Mode:             shared.ModeOcean,
						},
					},
				},
				Quantity: quantity,
				WeightKG: weightKG,
				VolumeM3: volumeM3,
			},
		},
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

	// 結果表示
	fmt.Printf("  Rate: %s (Status: %s)\n", output.RateName, output.RateStatus)
	fmt.Println()

	for i, result := range output.RouteResults {
		routeName := simRoutes[i].name
		if result.IsAvailable {
			fmt.Printf("  ┌─ [シミュレーション結果] %s ✓ 輸送可能 ─────────────────────\n", routeName)
			fmt.Printf("  │ %-8s  %-12s  %s\n", "Charge", "Category", "Amount")
			fmt.Println("  │ " + repeatChar('-', 40))
			for _, charge := range result.AppliedCharges {
				fmt.Printf("  │ %-8s  %-12s  $%s %s\n",
					charge.ChargeCode,
					charge.Category,
					charge.Amount.Amount.StringFixed(2),
					charge.Amount.Currency,
				)
			}
			if len(result.SkippedCharges) > 0 {
				fmt.Println("  │ " + repeatChar('-', 40))
				fmt.Printf("  │ スキップ: %d 費目\n", len(result.SkippedCharges))
			}
			fmt.Println("  │ " + repeatChar('-', 40))
			for _, total := range result.TotalAmounts {
				fmt.Printf("  │ 合計見積額: $%s %s\n",
					total.Amount.Amount.StringFixed(2),
					total.Currency,
				)
			}
			fmt.Println("  └──────────────────────────────────────────────────────")
		} else {
			fmt.Printf("  ┌─ [シミュレーション結果] %s ✗ 輸送不可 ─────────────────────\n", routeName)
			fmt.Println("  │ 該当するレートエントリが見つかりません")
			fmt.Println("  │ → このルートは現在のレートではカバーされていません")
			fmt.Println("  └──────────────────────────────────────────────────────")
		}
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
