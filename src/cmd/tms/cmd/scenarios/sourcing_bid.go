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

// SourcingBidScenario: 入札フローシナリオ
// 3社FWDから10ルートの料金比較→最安契約→レート反映
type SourcingBidScenario struct{}

func (s *SourcingBidScenario) Name() string       { return "sourcing-bid" }
func (s *SourcingBidScenario) Description() string {
	return "Sourcing bid flow: 3 FWDs × 10 routes → compare → award → rate"
}

type locationInfo struct {
	location *route.Location
	name     string
}

type vendorInfo struct {
	vendor *contract.Vendor
	name   string
}

type routeDef struct {
	name      string
	originIdx int
	destIdx   int
	mode      shared.TransportMode
	basePrice int // base price in USD cents
}

func (s *SourcingBidScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== Sourcing Bid Scenario ===")
	fmt.Println()

	// === Setup（マスターデータ） ===
	locations, err := s.setupLocations(ctx, deps, pool)
	if err != nil {
		return fmt.Errorf("setup locations: %w", err)
	}

	vendors, err := s.setupVendors(ctx, deps, pool)
	if err != nil {
		return fmt.Errorf("setup vendors: %w", err)
	}

	shipperID := uuid.New()
	bidRequestID := uuid.New()

	// ルート定義（10ルート）
	routes := s.defineRoutes()

	// === Business Flow ===

	// Step 1: 各FWDとのDRAFT契約作成
	contracts, err := s.step1CreateContracts(ctx, deps, vendors, shipperID, bidRequestID)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	// Step 2: 各社×10ルートのTariff登録
	tariffMap, err := s.step2RegisterTariffs(ctx, deps, contracts, vendors, locations, routes)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}

	// Step 3: 料金比較
	winners := s.step3CompareTariffs(vendors, routes, tariffMap)

	// Step 4: 契約Award
	err = s.step4AwardContracts(ctx, deps, contracts, vendors, winners, shipperID)
	if err != nil {
		return fmt.Errorf("step 4: %w", err)
	}

	// Step 5: レート作成
	rateOutput, err := s.step5CreateRate(ctx, deps, shipperID)
	if err != nil {
		return fmt.Errorf("step 5: %w", err)
	}

	// Step 6: 契約のTariffをレートに反映
	err = s.step6ApplyContractsToRate(ctx, deps, rateOutput.RateID, winners)
	if err != nil {
		return fmt.Errorf("step 6: %w", err)
	}

	// Step 7: レート有効化
	err = s.step7ActivateRate(ctx, deps, rateOutput.RateID)
	if err != nil {
		return fmt.Errorf("step 7: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// ====== Setup ======

func (s *SourcingBidScenario) setupLocations(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) ([]locationInfo, error) {
	fmt.Print("[Setup] Creating 11 locations... ")

	repo := networkpersistence.NewPostgresLocationRepo(pool)

	defs := []struct {
		name    string
		code    string
		country string
		locType string
	}{
		{"Tokyo", "JPTYO", "JP", "PORT"},
		{"Yokohama", "JPYOK", "JP", "PORT"},
		{"Shanghai", "CNSHA", "CN", "PORT"},
		{"Singapore", "SGSIN", "SG", "PORT"},
		{"Bangkok", "THBKK", "TH", "PORT"},
		{"Jakarta", "IDJKT", "ID", "PORT"},
		{"Ho Chi Minh", "VNSGN", "VN", "PORT"},
		{"Los Angeles", "USLAX", "US", "PORT"},
		{"Rotterdam", "NLRTM", "NL", "PORT"},
		{"Hamburg", "DEHAM", "DE", "PORT"},
		{"Dubai", "AEDXB", "AE", "PORT"},
	}

	locations := make([]locationInfo, 0, len(defs))
	for _, d := range defs {
		code := d.code
		loc := &route.Location{
			ID:          route.LocationID(uuid.New()),
			Name:        d.name,
			UnLocode:    &code,
			CountryCode: d.country,
			Type:        d.locType,
		}
		if err := repo.Save(ctx, loc); err != nil {
			return nil, fmt.Errorf("save location %s: %w", d.name, err)
		}
		locations = append(locations, locationInfo{location: loc, name: d.name})
	}

	fmt.Println("done")

	// 画面イメージ: 拠点マスタ一覧
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

func (s *SourcingBidScenario) setupVendors(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) ([]vendorInfo, error) {
	fmt.Print("[Setup] Creating 3 vendors... ")

	repo := sourcingpersistence.NewPostgresVendorRepo(pool)

	names := []string{"FWD Alpha", "FWD Beta", "FWD Gamma"}
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

	// 画面イメージ: 業者マスタ一覧
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

func (s *SourcingBidScenario) defineRoutes() []routeDef {
	return []routeDef{
		{"Tokyo → Shanghai (OCEAN)", 0, 2, shared.ModeOcean, 120000},
		{"Tokyo → Singapore (OCEAN)", 0, 3, shared.ModeOcean, 80000},
		{"Yokohama → Los Angeles (OCEAN)", 1, 7, shared.ModeOcean, 250000},
		{"Shanghai → Rotterdam (OCEAN)", 2, 8, shared.ModeOcean, 180000},
		{"Singapore → Hamburg (OCEAN)", 3, 9, shared.ModeOcean, 200000},
		{"Bangkok → Dubai (OCEAN)", 4, 10, shared.ModeOcean, 150000},
		{"Jakarta → Singapore (OCEAN)", 5, 3, shared.ModeOcean, 60000},
		{"Ho Chi Minh → Tokyo (OCEAN)", 6, 0, shared.ModeOcean, 100000},
		{"Shanghai → Los Angeles (OCEAN)", 2, 7, shared.ModeOcean, 300000},
		{"Rotterdam → Dubai (OCEAN)", 8, 10, shared.ModeOcean, 160000},
	}
}

// ====== Business Flow Steps ======

func (s *SourcingBidScenario) step1CreateContracts(
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
		fmt.Printf("  → %-12s contract %s 作成\n", v.name+":", output.ContractID.String()[:8])
	}

	// 画面イメージ: 契約一覧（ステータス: DRAFT）
	contractList, err := deps.SourcingQuery.ListContractsByShipper(ctx, shipperID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [契約管理画面] 契約一覧 ────────────────────────────────────────────")
		fmt.Printf("  │ %-8s  %-36s  %-10s  %-12s  %s\n", "Contract", "ProviderID", "Status", "ValidFrom", "ValidTo")
		fmt.Println("  │ " + repeatChar('-', 78))
		for _, c := range contractList {
			fmt.Printf("  │ %-8s  %-36s  %-10s  %-12s  %s\n",
				c.ID.String()[:8],
				c.ProviderID.String()[:8]+"...",
				string(c.Status),
				c.ValidFrom.Time.Format("2006-01-02"),
				c.ValidTo.Time.Format("2006-01-02"),
			)
		}
		fmt.Printf("  └─ 計 %d 件（全件 DRAFT — 料金表の登録待ち）\n", len(contractList))
		fmt.Println()
	}

	return contracts, nil
}

func (s *SourcingBidScenario) step2RegisterTariffs(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
	locations []locationInfo,
	routes []routeDef,
) (map[string]map[int]tariffResult, error) {
	fmt.Println("[Step 2] Registering tariffs (1 tariff per FWD × 10 line items)...")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// tariffMap[vendorName][routeIdx] = tariffResult
	tariffMap := make(map[string]map[int]tariffResult)

	for vi, v := range vendors {
		tariffMap[v.name] = make(map[int]tariffResult)

		// 全10ルート分の価格をまず決定し、LineItemに変換
		prices := make([]decimal.Decimal, len(routes))
		lineItems := make([]tariffapp.LineItemInput, len(routes))

		for ri, r := range routes {
			variation := 0.7 + rng.Float64()*0.6
			price := decimal.NewFromFloat(float64(r.basePrice) / 100.0 * variation)
			prices[ri] = price

			originID := uuid.UUID(locations[r.originIdx].location.ID)
			destID := uuid.UUID(locations[r.destIdx].location.ID)

			lineItems[ri] = tariffapp.LineItemInput{
				ChargeCode: "OFT",
				Category:   "FREIGHT",
				ScopeType:  pricing.ScopeTransportation,
				ScopeAttrs: map[string]string{
					"OriginID":      originID.String(),
					"DestinationID": destID.String(),
					"Mode":          string(r.mode),
				},
				PricingType: pricing.PricingFlat,
				PricingAttrs: map[string]any{
					"Amount":   price,
					"Currency": "USD",
				},
			}
		}

		// 1社 = 1 Tariff（全ルート分のLineItemをまとめて登録）
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

		// 各ルートに同じ TariffID を割り当て、価格は個別に保持
		for ri, price := range prices {
			tariffMap[v.name][ri] = tariffResult{
				tariffID: output.TariffID,
				price:    price,
			}
		}

		fmt.Printf("  → %-12s 1 tariff registered (%d line items)\n", v.name+":", len(lineItems))

		// 画面イメージ: 各社ごとに料金表一覧を表示
		tariffList, err := deps.SourcingQuery.ListTariffsByContract(ctx, contracts[vi].ContractID)
		if err == nil {
			fmt.Println()
			fmt.Printf("  ┌─ [料金表画面] %s の料金表 (ContractID: %s) ─────────────────\n",
				v.name, contracts[vi].ContractID.String()[:8])
			fmt.Printf("  │ %-8s  %-52s  %-5s  %-9s  %s\n", "TariffID", "Name", "Ver", "LineItems", "Effective")
			fmt.Println("  │ " + repeatChar('-', 88))
			for _, t := range tariffList {
				fmt.Printf("  │ %-8s  %-52s  v%-4d  %-9d  %s〜%s\n",
					t.ID.String()[:8],
					truncate(t.Name, 52),
					t.Version,
					output.LineItemCount,
					t.EffectiveFrom.Time.Format("2006-01-02"),
					t.EffectiveTo.Time.Format("2006-01-02"),
				)
			}
			fmt.Printf("  └─ 計 %d 件（1 Tariff に %d ルート分の LineItem）\n", len(tariffList), len(routes))
			fmt.Println()
		}
	}

	return tariffMap, nil
}

type tariffResult struct {
	tariffID uuid.UUID
	price    decimal.Decimal
}

type routeWinner struct {
	vendorIdx  int
	vendorName string
	routeIdx   int
	contractID uuid.UUID
}

func (s *SourcingBidScenario) step3CompareTariffs(
	vendors []vendorInfo,
	routes []routeDef,
	tariffMap map[string]map[int]tariffResult,
) []routeWinner {
	fmt.Println("[Step 3] Comparing tariffs per route...")

	// Header
	fmt.Println()
	fmt.Printf("  ┌─ [入札比較画面] ルート別料金比較 ─────────────────────────────────────────────────────\n")
	fmt.Printf("  │ %-35s", "Route")
	for _, v := range vendors {
		fmt.Printf("│ %-14s", v.name)
	}
	fmt.Println("│ Winner")
	fmt.Printf("  │ %s", repeatChar('-', 35))
	for range vendors {
		fmt.Printf("+%s", repeatChar('-', 15))
	}
	fmt.Println("+" + repeatChar('-', 14))

	winners := make([]routeWinner, 0, len(routes))

	for ri, r := range routes {
		// 最安値を先に確定
		bestIdx := 0
		bestPrice := tariffMap[vendors[0].name][ri].price
		bestName := vendors[0].name

		for vi, v := range vendors {
			price := tariffMap[v.name][ri].price
			if vi > 0 && price.LessThan(bestPrice) {
				bestIdx = vi
				bestPrice = price
				bestName = v.name
			}
		}

		fmt.Printf("  │ %-35s", r.name)
		for _, v := range vendors {
			price := tariffMap[v.name][ri].price
			marker := "  "
			if v.name == bestName {
				marker = "★"
			}
			formatted := fmt.Sprintf("$%s %s", price.StringFixed(0), marker)
			fmt.Printf("│ %-14s", formatted)
		}
		fmt.Printf("│ %s\n", bestName)

		winners = append(winners, routeWinner{
			vendorIdx:  bestIdx,
			vendorName: bestName,
			routeIdx:   ri,
		})
	}

	// 各社の勝利数サマリー
	winCount := make(map[string]int)
	for _, w := range winners {
		winCount[w.vendorName]++
	}
	fmt.Printf("  │ %s\n", repeatChar('-', 100))
	fmt.Printf("  │ 勝利数: ")
	for _, v := range vendors {
		fmt.Printf("%-14s: %d件  ", v.name, winCount[v.name])
	}
	fmt.Println()
	fmt.Println("  └─ ★ = 最安値（Award対象）")
	fmt.Println()

	return winners
}

func (s *SourcingBidScenario) step4AwardContracts(
	ctx context.Context,
	deps *ScenarioDeps,
	contracts []*bidapp.CreateBidContractOutput,
	vendors []vendorInfo,
	winners []routeWinner,
	shipperID uuid.UUID,
) error {
	fmt.Println("[Step 4] Awarding contracts...")

	// 各ベンダーが勝った数を集計
	winCount := make(map[int]int)
	for _, w := range winners {
		winCount[w.vendorIdx]++
	}

	// winnersにcontractIDを設定
	for i := range winners {
		winners[i].contractID = contracts[winners[i].vendorIdx].ContractID
	}

	// 全社の契約をCONTRACTED化（勝ったルートがある場合）
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

	// 画面イメージ: 契約一覧（ステータス変化の確認）
	contractList, err := deps.SourcingQuery.ListContractsByShipper(ctx, shipperID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [契約管理画面] 契約一覧（Award後） ─────────────────────────────────────────────────")
		fmt.Printf("  │ %-8s  %-36s  %-10s  %-5s  %s\n", "Contract", "ProviderID", "Status", "Won", "Note")
		fmt.Println("  │ " + repeatChar('-', 80))
		for ci, c := range contractList {
			won := winCount[ci]
			note := ""
			if string(c.Status) == "CONTRACTED" {
				note = "✓ Awarded"
			} else {
				note = "- Not awarded"
			}
			fmt.Printf("  │ %-8s  %-36s  %-10s  %-5d  %s\n",
				c.ID.String()[:8],
				c.ProviderID.String()[:8]+"...",
				string(c.Status),
				won,
				note,
			)
		}
		fmt.Println("  └─ CONTRACTED = レートへの反映が可能な状態")
		fmt.Println()
	}

	return nil
}

func (s *SourcingBidScenario) step5CreateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	shipperID uuid.UUID,
) (*rateapp.CreateRateOutput, error) {
	fmt.Printf("[Step 5] Creating rate \"2026 H1 Rate\"... ")

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

	fmt.Printf("done\n")

	// 画面イメージ: レート詳細（新規作成直後）
	rate, err := deps.RateQuery.GetRate(ctx, output.RateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レート管理画面] レート詳細 ─────────────────────────────────")
		fmt.Printf("  │  ID     : %s\n", rate.ID.String()[:8]+"...")
		fmt.Printf("  │  Name   : %s\n", rate.Name)
		fmt.Printf("  │  Status : %s  ← DRAFT（エントリ追加中）\n", string(rate.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			rate.ValidFrom.Time.Format("2006-01-02"),
			rate.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Printf("  │  Entries: 0 件（まだ空）\n")
		fmt.Println("  └──────────────────────────────────────────────────────────────")
		fmt.Println()
	}

	return output, nil
}

func (s *SourcingBidScenario) step6ApplyContractsToRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	winners []routeWinner,
) error {
	fmt.Println("[Step 6] Applying contracts to rate...")

	// winnersから勝ったvendorの契約IDを重複なく収集
	appliedContracts := make(map[uuid.UUID]bool)

	for _, w := range winners {
		contractID := w.contractID
		if appliedContracts[contractID] {
			continue
		}
		appliedContracts[contractID] = true

		input := rateapp.ApplyContractToRateInput{
			RateID:     rateID,
			ContractID: contractID,
		}

		output, err := deps.ApplyContractToRateUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("apply contract %s: %w", contractID.String()[:8], err)
		}
		fmt.Printf("  → ContractID %s のTariffを適用（累計エントリ数: %d）\n",
			contractID.String()[:8], output.TotalEntryCount)
	}

	// 画面イメージ: レートエントリ一覧
	// sqlcgenを直接使うためにpoolは持っていないのでRateQueryのGetRateを使い件数を表示
	rate, err := deps.RateQuery.GetRate(ctx, rateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レート管理画面] レートエントリ一覧 ─────────────────────────")
		fmt.Printf("  │  Rate: %s  Status: %s\n", rate.Name, string(rate.Status))
		fmt.Println("  │ " + repeatChar('-', 60))
		fmt.Printf("  │  %-36s  %-36s\n", "ContractID", "TariffID")
		fmt.Println("  │ " + repeatChar('-', 60))

		// 各contract→tariff のマッピングをwinnersから表示
		seen := make(map[uuid.UUID]bool)
		entryCount := 0
		for _, w := range winners {
			if seen[w.contractID] {
				continue
			}
			seen[w.contractID] = true
		}

		// エントリ情報はUseCaseの返値から集積して表示
		entryCount = 0
		for _, w := range winners {
			fmt.Printf("  │  %-36s  %-36s\n",
				w.contractID.String()[:8]+"...",
				"(TariffID from contract)",
			)
			entryCount++
		}

		fmt.Printf("  └─ 計 %d エントリ追加（各ルートのTariffが適用済み）\n", entryCount)
		fmt.Println()
	}

	return nil
}

func (s *SourcingBidScenario) step7ActivateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
) error {
	fmt.Printf("[Step 7] Activating rate... ")

	input := rateapp.ActivateRateInput{
		RateID: rateID,
	}

	output, err := deps.ActivateRateUC.Execute(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("done\n")

	// 画面イメージ: レート詳細（ACTIVE化後）
	rate, err := deps.RateQuery.GetRate(ctx, rateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レート管理画面] レート詳細（有効化後） ─────────────────────")
		fmt.Printf("  │  ID     : %s\n", rate.ID.String()[:8]+"...")
		fmt.Printf("  │  Name   : %s\n", rate.Name)
		fmt.Printf("  │  Status : %s  ← DRAFT から変化\n", string(rate.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			rate.ValidFrom.Time.Format("2006-01-02"),
			rate.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Printf("  │  Entries: %d 件（コスト計算に使用可能）\n", output.EntryCount)
		fmt.Println("  └─ ACTIVE = 出荷コスト計算・業務利用が可能な状態")
		fmt.Println()
	}

	return nil
}

// ====== Helpers ======

func repeatChar(ch byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
