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

// RateBafUpdateScenario: BAF更新によるレート更新シナリオ
// 業者からBAF（Bunker Adjustment Factor）の更新通知が届いた場合に、
// 契約の料金表を改定しレートに反映するまでの業務フロー
type RateBafUpdateScenario struct{}

func (s *RateBafUpdateScenario) Name() string { return "rate-baf-update" }
func (s *RateBafUpdateScenario) Description() string {
	return "BAF update flow: amend contract tariff → update rate entry"
}

func (s *RateBafUpdateScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== Rate BAF Update Scenario ===")
	fmt.Println()

	// === Setup（マスターデータ） ===
	locations, err := s.setupLocations(ctx, pool)
	if err != nil {
		return fmt.Errorf("setup locations: %w", err)
	}

	vendor, err := s.setupVendor(ctx, pool)
	if err != nil {
		return fmt.Errorf("setup vendor: %w", err)
	}

	shipperID := uuid.New()
	bidRequestID := uuid.New()

	// === Business Flow ===

	// Step 1: DRAFT契約作成
	contractOutput, err := s.step1CreateContract(ctx, deps, vendor, shipperID, bidRequestID)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	// Step 2: Tariff登録（OFT + BAF の2種類 × 2ルート = 4 LineItems）
	tariffOutput, lineItemMap, err := s.step2RegisterTariff(ctx, deps, contractOutput.ContractID, vendor, locations)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}

	// Step 3: 契約Award
	err = s.step3AwardContract(ctx, deps, contractOutput.ContractID, shipperID)
	if err != nil {
		return fmt.Errorf("step 3: %w", err)
	}

	// Step 4: DRAFTレート作成
	rateOutput, err := s.step4CreateRate(ctx, deps, shipperID)
	if err != nil {
		return fmt.Errorf("step 4: %w", err)
	}

	// Step 5: 契約のTariffをレートに反映
	applyOutput, err := s.step5ApplyContractToRate(ctx, deps, rateOutput.RateID, contractOutput.ContractID)
	if err != nil {
		return fmt.Errorf("step 5: %w", err)
	}

	// Step 6: BAF更新通知 → 改定版Tariff（v2）作成
	amendOutput, err := s.step6AmendTariffForBAF(ctx, deps, contractOutput.ContractID, tariffOutput.TariffID, locations)
	if err != nil {
		return fmt.Errorf("step 6: %w", err)
	}

	// Step 7: レートのBAFエントリをv2に差し替え
	err = s.step7UpdateRateEntries(ctx, deps, rateOutput.RateID, contractOutput.ContractID, applyOutput, lineItemMap, amendOutput)
	if err != nil {
		return fmt.Errorf("step 7: %w", err)
	}

	// Step 8: レートACTIVE化
	err = s.step8ActivateRate(ctx, deps, rateOutput.RateID)
	if err != nil {
		return fmt.Errorf("step 8: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// ====== Setup ======

func (s *RateBafUpdateScenario) setupLocations(ctx context.Context, pool *pgxpool.Pool) ([]locationInfo, error) {
	fmt.Print("[Setup] Creating 2 locations... ")

	repo := networkpersistence.NewPostgresLocationRepo(pool)

	defs := []struct {
		name    string
		code    string
		country string
	}{
		{"Tokyo", "JPTYO", "JP"},
		{"Shanghai", "CNSHA", "CN"},
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

func (s *RateBafUpdateScenario) setupVendor(ctx context.Context, pool *pgxpool.Pool) (*vendorInfo, error) {
	fmt.Print("[Setup] Creating 1 vendor... ")

	repo := sourcingpersistence.NewPostgresVendorRepo(pool)

	v, err := contract.NewVendor("FWD Alpha", contract.ProviderTypeForwarder)
	if err != nil {
		return nil, err
	}
	if err := repo.Save(ctx, v); err != nil {
		return nil, fmt.Errorf("save vendor: %w", err)
	}

	fmt.Println("done")

	fmt.Println()
	fmt.Println("  ┌─ [業者マスタ] 登録済み業者 ─────────────────────────────")
	fmt.Printf("  │ %-36s  %-14s  %s\n", "ID", "Name", "Type")
	fmt.Println("  │ " + repeatChar('-', 60))
	fmt.Printf("  │ %-36s  %-14s  %s\n", v.ID, "FWD Alpha", string(v.Type))
	fmt.Println("  └─ 計 1 件")
	fmt.Println()

	return &vendorInfo{vendor: v, name: "FWD Alpha"}, nil
}

// ====== Business Flow Steps ======

// lineItemRef: LineItemとその種別を追跡するための構造体
type lineItemRef struct {
	lineItemID uuid.UUID
	chargeCode string // "OFT" or "BAF"
	routeName  string
	price      decimal.Decimal
}

func (s *RateBafUpdateScenario) step1CreateContract(
	ctx context.Context,
	deps *ScenarioDeps,
	vendor *vendorInfo,
	shipperID uuid.UUID,
	bidRequestID uuid.UUID,
) (*bidapp.CreateBidContractOutput, error) {
	fmt.Println("[Step 1] Creating bid contract...")

	input := bidapp.CreateBidContractInput{
		BidRequestID: bidRequestID,
		ProviderID:   vendor.vendor.ID,
		ShipperID:    shipperID,
		ValidFrom:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		ValidTo:      time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	}

	output, err := deps.CreateBidContractUC.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create contract: %w", err)
	}

	fmt.Printf("  → FWD Alpha: contract %s 作成（DRAFT）\n", output.ContractID.String()[:8])
	fmt.Println()

	return output, nil
}

func (s *RateBafUpdateScenario) step2RegisterTariff(
	ctx context.Context,
	deps *ScenarioDeps,
	contractID uuid.UUID,
	vendor *vendorInfo,
	locations []locationInfo,
) (*tariffapp.RegisterTariffDirectOutput, map[uuid.UUID]lineItemRef, error) {
	fmt.Println("[Step 2] Registering tariff (OFT + BAF × 2 routes = 4 line items)...")

	tokyoID := uuid.UUID(locations[0].location.ID)
	shanghaiID := uuid.UUID(locations[1].location.ID)

	// OFT: 海上運賃, BAF: 燃油サーチャージ（各ルート）
	oftPrice1 := decimal.NewFromInt(1200) // Tokyo → Shanghai OFT: $1,200
	bafPrice1 := decimal.NewFromInt(350)  // Tokyo → Shanghai BAF: $350
	oftPrice2 := decimal.NewFromInt(1100) // Shanghai → Tokyo OFT: $1,100
	bafPrice2 := decimal.NewFromInt(320)  // Shanghai → Tokyo BAF: $320

	lineItems := []tariffapp.LineItemInput{
		{
			ChargeCode: "OFT",
			Category:   "FREIGHT",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      tokyoID.String(),
				"DestinationID": shanghaiID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   oftPrice1,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "BAF",
			Category:   "SURCHARGE",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      tokyoID.String(),
				"DestinationID": shanghaiID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   bafPrice1,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "OFT",
			Category:   "FREIGHT",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      shanghaiID.String(),
				"DestinationID": tokyoID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   oftPrice2,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "BAF",
			Category:   "SURCHARGE",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      shanghaiID.String(),
				"DestinationID": tokyoID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   bafPrice2,
				"Currency": "USD",
			},
		},
	}

	input := tariffapp.RegisterTariffDirectInput{
		ContractID:    contractID,
		TariffName:    "FWD Alpha 2026H1 Ocean Rate Sheet",
		EffectiveFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EffectiveTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		LineItems:     lineItems,
	}

	output, err := deps.RegisterTariffDirectUC.Execute(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("register tariff: %w", err)
	}

	// LineItemIDとChargeCodeの対応を保持
	lineItemMap := map[uuid.UUID]lineItemRef{
		output.LineItemIDs[0]: {lineItemID: output.LineItemIDs[0], chargeCode: "OFT", routeName: "Tokyo → Shanghai", price: oftPrice1},
		output.LineItemIDs[1]: {lineItemID: output.LineItemIDs[1], chargeCode: "BAF", routeName: "Tokyo → Shanghai", price: bafPrice1},
		output.LineItemIDs[2]: {lineItemID: output.LineItemIDs[2], chargeCode: "OFT", routeName: "Shanghai → Tokyo", price: oftPrice2},
		output.LineItemIDs[3]: {lineItemID: output.LineItemIDs[3], chargeCode: "BAF", routeName: "Shanghai → Tokyo", price: bafPrice2},
	}

	fmt.Printf("  → 1 tariff registered (%d line items)\n", output.LineItemCount)

	// 画面イメージ: 料金表の詳細
	fmt.Println()
	fmt.Println("  ┌─ [料金表画面] FWD Alpha の料金表 ─────────────────────────────────────────")
	fmt.Printf("  │ Tariff : %s (v1)\n", output.TariffName)
	fmt.Printf("  │ Period : %s 〜 %s\n",
		output.EffectiveFrom.Format("2006-01-02"),
		output.EffectiveTo.Format("2006-01-02"),
	)
	fmt.Println("  │")
	fmt.Printf("  │ %-6s  %-10s  %-25s  %s\n", "Code", "Category", "Route", "UnitPrice")
	fmt.Println("  │ " + repeatChar('-', 60))
	fmt.Printf("  │ %-6s  %-10s  %-25s  $%s USD\n", "OFT", "FREIGHT", "Tokyo → Shanghai (OCEAN)", oftPrice1.StringFixed(0))
	fmt.Printf("  │ %-6s  %-10s  %-25s  $%s USD\n", "BAF", "SURCHARGE", "Tokyo → Shanghai (OCEAN)", bafPrice1.StringFixed(0))
	fmt.Printf("  │ %-6s  %-10s  %-25s  $%s USD\n", "OFT", "FREIGHT", "Shanghai → Tokyo (OCEAN)", oftPrice2.StringFixed(0))
	fmt.Printf("  │ %-6s  %-10s  %-25s  $%s USD\n", "BAF", "SURCHARGE", "Shanghai → Tokyo (OCEAN)", bafPrice2.StringFixed(0))
	fmt.Printf("  └─ 計 %d 件\n", output.LineItemCount)
	fmt.Println()

	return output, lineItemMap, nil
}

func (s *RateBafUpdateScenario) step3AwardContract(
	ctx context.Context,
	deps *ScenarioDeps,
	contractID uuid.UUID,
	shipperID uuid.UUID,
) error {
	fmt.Println("[Step 3] Awarding contract...")

	input := bidapp.AwardBidContractInput{
		ContractID: contractID,
	}
	_, err := deps.AwardBidContractUC.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("award contract: %w", err)
	}

	fmt.Printf("  → FWD Alpha: DRAFT → CONTRACTED\n")

	// 画面イメージ: 契約ステータス確認
	contractData, err := deps.SourcingQuery.GetContract(ctx, contractID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [契約管理画面] 契約詳細 ────────────────────────────────")
		fmt.Printf("  │  ID     : %s\n", contractData.ID.String()[:8]+"...")
		fmt.Printf("  │  Status : %s  ← Award完了\n", string(contractData.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			contractData.ValidFrom.Time.Format("2006-01-02"),
			contractData.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Println("  └──────────────────────────────────────────────────────")
		fmt.Println()
	}

	return nil
}

func (s *RateBafUpdateScenario) step4CreateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	shipperID uuid.UUID,
) (*rateapp.CreateRateOutput, error) {
	fmt.Print("[Step 4] Creating rate \"2026 H1 Rate\"... ")

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

	rate, err := deps.RateQuery.GetRate(ctx, output.RateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レート管理画面] レート詳細 ─────────────────────────────")
		fmt.Printf("  │  ID     : %s\n", rate.ID.String()[:8]+"...")
		fmt.Printf("  │  Name   : %s\n", rate.Name)
		fmt.Printf("  │  Status : %s  ← エントリ追加中\n", string(rate.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			rate.ValidFrom.Time.Format("2006-01-02"),
			rate.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Printf("  │  Entries: 0 件（まだ空）\n")
		fmt.Println("  └──────────────────────────────────────────────────────")
		fmt.Println()
	}

	return output, nil
}

func (s *RateBafUpdateScenario) step5ApplyContractToRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	contractID uuid.UUID,
) (*rateapp.ApplyContractToRateOutput, error) {
	fmt.Println("[Step 5] Applying contract to rate...")

	input := rateapp.ApplyContractToRateInput{
		RateID:     rateID,
		ContractID: contractID,
	}

	output, err := deps.ApplyContractToRateUC.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("apply contract: %w", err)
	}

	fmt.Printf("  → %d エントリを適用\n", len(output.AddedEntries))

	// 画面イメージ: レートカード
	fmt.Println()
	fmt.Println("  ┌─ [レートカード] レート一覧（BAF更新前） ──────────────────────────────────────────")
	fmt.Printf("  │ %-25s│ %-12s│ %-8s│ %s\n", "Route", "Provider", "Charge", "UnitPrice")
	fmt.Printf("  │%s┼%s┼%s┼%s\n", repeatChar('-', 26), repeatChar('-', 13), repeatChar('-', 9), repeatChar('-', 16))
	for _, entry := range output.AddedEntries {
		routeDesc := "—"
		if entry.OriginID != nil && entry.DestinationID != nil {
			routeDesc = entry.OriginID.String()[:8] + "→" + entry.DestinationID.String()[:8]
		}
		fmt.Printf("  │ %-25s│ %-12s│ %-8s│ $%s %s\n",
			routeDesc,
			"FWD Alpha",
			entry.ChargeCode,
			entry.UnitPrice.Amount.StringFixed(2),
			entry.UnitPrice.Currency,
		)
	}
	fmt.Printf("  └─ 計 %d エントリ\n", len(output.AddedEntries))
	fmt.Println()

	return output, nil
}

func (s *RateBafUpdateScenario) step6AmendTariffForBAF(
	ctx context.Context,
	deps *ScenarioDeps,
	contractID uuid.UUID,
	baseTariffID uuid.UUID,
	locations []locationInfo,
) (*tariffapp.AmendContractTariffDirectOutput, error) {
	fmt.Println("[Step 6] BAF update notification received → Creating tariff v2...")
	fmt.Println("  ※ 業者から燃油サーチャージ（BAF）の改定通知が届きました")
	fmt.Println()

	tokyoID := uuid.UUID(locations[0].location.ID)
	shanghaiID := uuid.UUID(locations[1].location.ID)

	// BAFが値上がりするシナリオ（OFTは据え置き）
	oftPrice1 := decimal.NewFromInt(1200)   // 据え置き
	newBafPrice1 := decimal.NewFromInt(420) // $350 → $420（+20%）
	oftPrice2 := decimal.NewFromInt(1100)   // 据え置き
	newBafPrice2 := decimal.NewFromInt(380) // $320 → $380（+18.75%）

	lineItems := []tariffapp.LineItemInput{
		{
			ChargeCode: "OFT",
			Category:   "FREIGHT",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      tokyoID.String(),
				"DestinationID": shanghaiID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   oftPrice1,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "BAF",
			Category:   "SURCHARGE",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      tokyoID.String(),
				"DestinationID": shanghaiID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   newBafPrice1,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "OFT",
			Category:   "FREIGHT",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      shanghaiID.String(),
				"DestinationID": tokyoID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   oftPrice2,
				"Currency": "USD",
			},
		},
		{
			ChargeCode: "BAF",
			Category:   "SURCHARGE",
			ScopeType:  pricing.ScopeTransportation,
			ScopeAttrs: map[string]string{
				"OriginID":      shanghaiID.String(),
				"DestinationID": tokyoID.String(),
				"Mode":          string(shared.ModeOcean),
			},
			PricingType: pricing.PricingFlat,
			PricingAttrs: map[string]any{
				"Amount":   newBafPrice2,
				"Currency": "USD",
			},
		},
	}

	input := tariffapp.AmendContractTariffDirectInput{
		ContractID:    contractID,
		BaseTariffID:  baseTariffID,
		EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EffectiveTo:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		LineItems:     lineItems,
	}

	output, err := deps.AmendTariffDirectUC.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("amend tariff: %w", err)
	}

	fmt.Printf("  → Tariff v%d 作成完了（ID: %s）\n", output.TariffVersion, output.TariffID.String()[:8])
	fmt.Printf("  → BaseTariff: %s\n", output.BaseTariffID.String()[:8])

	// 画面イメージ: 改定前後のBAF比較
	fmt.Println()
	fmt.Println("  ┌─ [料金表画面] BAF改定前後の比較 ──────────────────────────────────────────")
	fmt.Printf("  │ %-25s│ %-8s│ %-12s│ %-12s│ %s\n", "Route", "Code", "v1 Price", "v2 Price", "Change")
	fmt.Printf("  │%s┼%s┼%s┼%s┼%s\n", repeatChar('-', 26), repeatChar('-', 9), repeatChar('-', 13), repeatChar('-', 13), repeatChar('-', 10))
	fmt.Printf("  │ %-25s│ %-8s│ $%-11s│ $%-11s│ %s\n", "Tokyo → Shanghai", "OFT", "1200", "1200", "—")
	fmt.Printf("  │ %-25s│ %-8s│ $%-11s│ $%-11s│ %s\n", "Tokyo → Shanghai", "BAF", "350", "420", "+$70 (↑20%)")
	fmt.Printf("  │ %-25s│ %-8s│ $%-11s│ $%-11s│ %s\n", "Shanghai → Tokyo", "OFT", "1100", "1100", "—")
	fmt.Printf("  │ %-25s│ %-8s│ $%-11s│ $%-11s│ %s\n", "Shanghai → Tokyo", "BAF", "320", "380", "+$60 (↑19%)")
	fmt.Printf("  └─ TariffID: %s → %s (v1 → v%d)\n", baseTariffID.String()[:8], output.TariffID.String()[:8], output.TariffVersion)
	fmt.Println()

	return output, nil
}

func (s *RateBafUpdateScenario) step7UpdateRateEntries(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
	contractID uuid.UUID,
	applyOutput *rateapp.ApplyContractToRateOutput,
	oldLineItemMap map[uuid.UUID]lineItemRef,
	amendOutput *tariffapp.AmendContractTariffDirectOutput,
) error {
	fmt.Println("[Step 7] Updating rate entries with new BAF tariff...")

	// BAFエントリのみを新Tariffの対応するLineItemに差し替え
	// applyOutputのAddedEntriesからBAFエントリを特定し、amendOutputのLineItemIDsと対応付ける
	updatedCount := 0
	for _, entry := range applyOutput.AddedEntries {
		ref, ok := oldLineItemMap[entry.TariffLineItemID]
		if !ok || ref.chargeCode != "BAF" {
			continue // BAF以外はスキップ
		}

		// BAFのLineItemは v2 Tariffでも同じインデックス順で作成されている
		// v1: [0]=OFTr1, [1]=BAFr1, [2]=OFTr2, [3]=BAFr2
		// v2: [0]=OFTr1, [1]=BAFr1, [2]=OFTr2, [3]=BAFr2
		var newLineItemID uuid.UUID
		switch ref.routeName {
		case "Tokyo → Shanghai":
			newLineItemID = amendOutput.LineItemIDs[1] // BAF r1
		case "Shanghai → Tokyo":
			newLineItemID = amendOutput.LineItemIDs[3] // BAF r2
		default:
			continue
		}

		input := rateapp.UpdateRateEntryTariffInput{
			RateID:              rateID,
			EntryID:             entry.EntryID,
			ContractID:          contractID,
			NewTariffID:         amendOutput.TariffID,
			NewTariffLineItemID: newLineItemID,
		}

		output, err := deps.UpdateRateEntryTariffUC.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("update entry %s: %w", entry.EntryID.String()[:8], err)
		}

		fmt.Printf("  → Entry %s: BAF %s → Tariff v2 (old=%s, new=%s)\n",
			output.EntryID.String()[:8],
			ref.routeName,
			output.OldTariffID.String()[:8],
			output.NewTariffID.String()[:8],
		)
		updatedCount++
	}

	fmt.Printf("  → %d 件のBAFエントリを更新\n", updatedCount)

	// 画面イメージ: 更新後のレートカード
	rateEntries, err := deps.RateQuery.ListRateEntries(ctx, rateID)
	if err == nil {
		fmt.Println()
		fmt.Println("  ┌─ [レートカード] レート一覧（BAF更新後） ──────────────────────────────────────────")
		fmt.Printf("  │ %-8s  %-8s  %-10s  %-12s  %s\n", "EntryID", "Tariff", "Charge", "Category", "UnitPrice")
		fmt.Println("  │ " + repeatChar('-', 60))
		for _, e := range rateEntries {
			fmt.Printf("  │ %-8s  %-8s  %-10s  %-12s  $%s %s\n",
				e.ID.String()[:8],
				e.TariffID.String()[:8],
				e.ChargeCode,
				e.Category,
				e.UnitPriceAmount.String(),
				e.UnitPriceCurrency,
			)
		}
		fmt.Printf("  └─ 計 %d エントリ（BAFエントリは新Tariffを参照）\n", len(rateEntries))
		fmt.Println()
	}

	return nil
}

func (s *RateBafUpdateScenario) step8ActivateRate(
	ctx context.Context,
	deps *ScenarioDeps,
	rateID uuid.UUID,
) error {
	fmt.Print("[Step 8] Activating rate... ")

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
		fmt.Printf("  │  Status : %s  ← BAF改定反映済み\n", string(rate.Status))
		fmt.Printf("  │  Period : %s 〜 %s\n",
			rate.ValidFrom.Time.Format("2006-01-02"),
			rate.ValidTo.Time.Format("2006-01-02"),
		)
		fmt.Printf("  │  Entries: %d 件（コスト計算に使用可能）\n", output.EntryCount)
		fmt.Println("  └─ ACTIVE = BAF改定が反映された状態で業務利用可能")
		fmt.Println()
	}

	return nil
}
