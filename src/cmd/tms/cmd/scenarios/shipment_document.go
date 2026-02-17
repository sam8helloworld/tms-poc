package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	docapp "github.com/sam8helloworld/tms-poc/internal/document/application/document"
	docdomain "github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	networkpersistence "github.com/sam8helloworld/tms-poc/internal/network/infrastructure/persistence"
	shipmentapp "github.com/sam8helloworld/tms-poc/internal/shipment/application/shipment"
	shipmentdomain "github.com/sam8helloworld/tms-poc/internal/shipment/domain/shipment"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	sourcingpersistence "github.com/sam8helloworld/tms-poc/internal/sourcing/infrastructure/persistence"
)

// ShipmentDocumentScenario: 輸送案件の書類E2Eシナリオ
// 輸送案件を作成し、海上輸送で発生する書類を登録・確認してマイルストーンを蓄積する
type ShipmentDocumentScenario struct{}

func (s *ShipmentDocumentScenario) Name() string { return "shipment-document" }
func (s *ShipmentDocumentScenario) Description() string {
	return "Shipment document E2E: create shipment → register documents → complete"
}

// docStep: 書類登録→確認→マイルストーン記録の1ステップの定義
type docStep struct {
	stepNo        int
	label         string
	docType       shared.DocType
	origin        docdomain.DocumentOrigin
	fileName      string
	milestoneType shipmentdomain.MilestoneType // "" = マイルストーンなし
	payload       shipmentdomain.MilestonePayload
}

func (s *ShipmentDocumentScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== Shipment Document E2E Scenario ===")
	fmt.Println()

	// === Step 0: Setup ===
	origin, dest, vendor, err := s.setup(ctx, pool)
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	shipperID := uuid.New()
	consigneeID := uuid.New()
	uploaderID := uuid.New()

	// === Step 1: 輸送案件作成 ===
	shipOutput, err := s.step1CreateShipment(ctx, deps, shipperID, consigneeID, origin, dest)
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}
	shipmentID := shipOutput.ShipmentID

	// 書類ステップの定義
	now := time.Now()
	steps := s.defineDocSteps(now, vendor)

	// === Step 2-11: 書類登録・確認・マイルストーン記録 ===
	for _, step := range steps {
		if err := s.executeDocStep(ctx, deps, step, shipmentID, uploaderID); err != nil {
			return fmt.Errorf("step %d (%s): %w", step.stepNo, step.label, err)
		}
	}

	// === 最終結果表示 ===
	s.printFinalSummary(shipmentID)

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// ====== Setup ======

func (s *ShipmentDocumentScenario) setup(ctx context.Context, pool *pgxpool.Pool) (route.LocationID, route.LocationID, *contract.Vendor, error) {
	fmt.Print("[Setup] Creating locations and vendor... ")

	locationRepo := networkpersistence.NewPostgresLocationRepo(pool)
	vendorRepo := sourcingpersistence.NewPostgresVendorRepo(pool)

	// Locations
	tokyoCode := "JPTYO"
	tokyo := &route.Location{
		ID:          route.LocationID(uuid.New()),
		Name:        "Tokyo",
		UnLocode:    &tokyoCode,
		CountryCode: "JP",
		Type:        "PORT",
	}
	if err := locationRepo.Save(ctx, tokyo); err != nil {
		return route.LocationID(uuid.Nil), route.LocationID(uuid.Nil), nil, err
	}

	shanghaiCode := "CNSHA"
	shanghai := &route.Location{
		ID:          route.LocationID(uuid.New()),
		Name:        "Shanghai",
		UnLocode:    &shanghaiCode,
		CountryCode: "CN",
		Type:        "PORT",
	}
	if err := locationRepo.Save(ctx, shanghai); err != nil {
		return route.LocationID(uuid.Nil), route.LocationID(uuid.Nil), nil, err
	}

	// Vendor
	vendor, err := contract.NewVendor("Ocean Express Co.", contract.ProviderTypeCarrier)
	if err != nil {
		return route.LocationID(uuid.Nil), route.LocationID(uuid.Nil), nil, err
	}
	if err := vendorRepo.Save(ctx, vendor); err != nil {
		return route.LocationID(uuid.Nil), route.LocationID(uuid.Nil), nil, err
	}

	fmt.Println("done")
	fmt.Println()
	fmt.Println("  ┌─ [Setup] マスターデータ ─────────────────────────────")
	fmt.Printf("  │ Origin:      Tokyo (JPTYO)  ID: %s\n", uuid.UUID(tokyo.ID).String()[:8])
	fmt.Printf("  │ Destination: Shanghai (CNSHA) ID: %s\n", uuid.UUID(shanghai.ID).String()[:8])
	fmt.Printf("  │ Vendor:      %s  ID: %s\n", vendor.Name, vendor.ID.String()[:8])
	fmt.Println("  └──────────────────────────────────────────────────────")
	fmt.Println()

	return tokyo.ID, shanghai.ID, vendor, nil
}

// ====== Step 1: 輸送案件作成 ======

func (s *ShipmentDocumentScenario) step1CreateShipment(
	ctx context.Context,
	deps *ScenarioDeps,
	shipperID, consigneeID uuid.UUID,
	originID, destID route.LocationID,
) (*shipmentapp.CreateShipmentOutput, error) {
	fmt.Println("[Step 1] 輸送案件作成")

	input := shipmentapp.CreateShipmentInput{
		ShipmentNo:       "SHP-2026-0001",
		ShipperID:        shipperID,
		ConsigneeID:      consigneeID,
		OriginLocationID: originID,
		DestLocationID:   destID,
		RateID:           uuid.New(), // ダミー（本シナリオではRate未使用）
	}

	output, err := deps.CreateShipmentUC.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	fmt.Printf("  → Shipment %s 作成完了\n", output.ShipmentNo)
	fmt.Println()
	fmt.Println("  ┌─ [輸送案件] ──────────────────────────────────────────")
	fmt.Printf("  │ ID:     %s\n", output.ShipmentID.String()[:8])
	fmt.Printf("  │ No:     %s\n", output.ShipmentNo)
	fmt.Printf("  │ Status: %s\n", output.Status)
	fmt.Println("  └──────────────────────────────────────────────────────")
	fmt.Println()

	return output, nil
}

// ====== 書類ステップ定義 ======

func (s *ShipmentDocumentScenario) defineDocSteps(now time.Time, _ *contract.Vendor) []docStep {
	return []docStep{
		{
			stepNo: 2, label: "ブッキング確認",
			docType: shared.DocTypeBookingConfirmation, origin: docdomain.OriginProvider,
			fileName: "booking_confirmation.pdf",
			milestoneType: shipmentdomain.MilestoneBookingConfirmed,
			payload: shipmentdomain.BookingConfirmedPayload{
				BookingNo:  "BKG-2026-001",
				VesselName: "EVER GIVEN",
				VoyageNo:   "V.2026E-045",
				ETD:        now.Add(7 * 24 * time.Hour),
				ETA:        now.Add(10 * 24 * time.Hour),
			},
		},
		{
			stepNo: 3, label: "S/I発行",
			docType: shared.DocTypeShippingInstruction, origin: docdomain.OriginShipper,
			fileName: "shipping_instruction.pdf",
			milestoneType: shipmentdomain.MilestoneShippingInstruction,
			payload: shipmentdomain.GenericPayload{
				MType: shipmentdomain.MilestoneShippingInstruction,
				Data:  map[string]interface{}{"status": "issued"},
			},
		},
		{
			stepNo: 4, label: "Commercial Invoice登録",
			docType: shared.DocTypeCommercialInvoice, origin: docdomain.OriginShipper,
			fileName: "commercial_invoice.pdf",
			// マイルストーンなし
		},
		{
			stepNo: 5, label: "Packing List登録",
			docType: shared.DocTypePackingList, origin: docdomain.OriginShipper,
			fileName: "packing_list.pdf",
			// マイルストーンなし
		},
		{
			stepNo: 6, label: "B/L受領・船積確認",
			docType: shared.DocTypeBillOfLading, origin: docdomain.OriginProvider,
			fileName: "bill_of_lading.pdf",
			milestoneType: shipmentdomain.MilestoneShipped,
			payload: shipmentdomain.ShippedPayload{
				TransportDocNo: "OOCL-BL-2026-001",
				OnBoardDate:    now.Add(7 * 24 * time.Hour),
				VesselName:     "EVER GIVEN",
				VoyageNo:       "V.2026E-045",
			},
		},
		{
			stepNo: 7, label: "輸出通関",
			docType: shared.DocTypeCustomsDeclaration, origin: docdomain.OriginProvider,
			fileName: "export_customs_declaration.pdf",
			milestoneType: shipmentdomain.MilestoneCustomsExportCleared,
			payload: shipmentdomain.CustomsClearedPayload{
				DeclarationNo: "EXP-2026-12345",
				ClearanceDate: now.Add(7 * 24 * time.Hour),
				Direction:     shared.DirectionExport,
			},
		},
		{
			stepNo: 8, label: "到着通知",
			docType: shared.DocTypeArrivalNotice, origin: docdomain.OriginProvider,
			fileName: "arrival_notice.pdf",
			milestoneType: shipmentdomain.MilestoneArrived,
			payload: shipmentdomain.ArrivedPayload{
				ArrivalDate:   now.Add(10 * 24 * time.Hour),
				DischargePort: "Shanghai (CNSHA)",
			},
		},
		{
			stepNo: 9, label: "輸入通関",
			docType: shared.DocTypeCustomsDeclaration, origin: docdomain.OriginProvider,
			fileName: "import_customs_declaration.pdf",
			milestoneType: shipmentdomain.MilestoneCustomsImportCleared,
			payload: shipmentdomain.CustomsClearedPayload{
				DeclarationNo: "IMP-2026-67890",
				ClearanceDate: now.Add(11 * 24 * time.Hour),
				Direction:     shared.DirectionImport,
			},
		},
		{
			stepNo: 10, label: "配送指図・納品",
			docType: shared.DocTypeDeliveryOrder, origin: docdomain.OriginProvider,
			fileName: "delivery_order.pdf",
			milestoneType: shipmentdomain.MilestoneDelivered,
			payload: shipmentdomain.DeliveredPayload{
				DeliveryDate:     now.Add(12 * 24 * time.Hour),
				DeliveryLocation: "Shanghai Warehouse",
				ReceiverName:     "Shanghai Trading Co.",
			},
		},
		{
			stepNo: 11, label: "Freight Invoice受領",
			docType: shared.DocTypeFreightInvoice, origin: docdomain.OriginProvider,
			fileName: "freight_invoice.pdf",
			milestoneType: shipmentdomain.MilestoneInvoiceReceived,
			payload: shipmentdomain.InvoiceReceivedPayload{
				InvoiceNo:   "FRT-INV-2026-001",
				InvoiceDate: now.Add(14 * 24 * time.Hour),
				TotalAmount: shared.Money{Amount: decimal.NewFromInt(450000), Currency: "USD"}, // $4,500.00
			},
		},
	}
}

// ====== 書類ステップ実行 ======

func (s *ShipmentDocumentScenario) executeDocStep(
	ctx context.Context,
	deps *ScenarioDeps,
	step docStep,
	shipmentID uuid.UUID,
	uploaderID uuid.UUID,
) error {
	fmt.Printf("[Step %d] %s\n", step.stepNo, step.label)

	// 1. 書類アップロード
	uploadOutput, err := deps.UploadDocumentUC.Execute(ctx, docapp.UploadDocumentInput{
		ShipmentID: shipmentID,
		DocType:    step.docType,
		Origin:     step.origin,
		FileName:   step.fileName,
		MimeType:   "application/pdf",
		StorageURI: fmt.Sprintf("s3://tms-docs/%s/%s", shipmentID.String()[:8], step.fileName),
		FileSize:   1024,
		UploadedBy: uploaderID,
	})
	if err != nil {
		return fmt.Errorf("upload document: %w", err)
	}
	fmt.Printf("  → 書類アップロード: %s (ID: %s)\n", step.fileName, uploadOutput.DocumentID.String()[:8])

	// 2. 書類確認
	confirmOutput, err := deps.ConfirmDocumentUC.Execute(ctx, docapp.ConfirmDocumentInput{
		DocumentID: uploadOutput.DocumentID,
	})
	if err != nil {
		return fmt.Errorf("confirm document: %w", err)
	}
	fmt.Printf("  → 書類確認完了: %s → %s\n", string(step.docType), confirmOutput.Status)

	// 3. マイルストーン記録（定義されている場合のみ）
	if step.milestoneType != "" {
		milestoneOutput, err := deps.RecordMilestoneUC.Execute(ctx, shipmentapp.RecordMilestoneInput{
			ShipmentID:       shipmentID,
			MilestoneType:    step.milestoneType,
			OccurredAt:       time.Now(),
			SourceDocumentID: uploadOutput.DocumentID,
			SourceDocType:    step.docType,
			Payload:          step.payload,
		})
		if err != nil {
			return fmt.Errorf("record milestone: %w", err)
		}
		fmt.Printf("  → マイルストーン記録: %s → Status: %s\n", string(step.milestoneType), milestoneOutput.NewStatus)
	}

	fmt.Println()
	return nil
}

// ====== 最終結果表示 ======

func (s *ShipmentDocumentScenario) printFinalSummary(shipmentID uuid.UUID) {
	fmt.Println("  ┌─ [最終結果] 輸送案件 書類・マイルストーン一覧 ────────────")
	fmt.Printf("  │ Shipment ID: %s\n", shipmentID.String()[:8])
	fmt.Println("  │")
	fmt.Println("  │  Step  DocType                   Origin    Milestone                    Status")
	fmt.Println("  │  " + repeatChar('-', 90))
	fmt.Println("  │   1    -                         -         ShipmentCreated              PLANNED")
	fmt.Println("  │   2    BOOKING_CONFIRMATION      PROVIDER  BOOKING_CONFIRMED            BOOKED")
	fmt.Println("  │   3    SHIPPING_INSTRUCTION      SHIPPER   SHIPPING_INSTRUCTION_ISSUED  BOOKED")
	fmt.Println("  │   4    COMMERCIAL_INVOICE        SHIPPER   -                            BOOKED")
	fmt.Println("  │   5    PACKING_LIST              SHIPPER   -                            BOOKED")
	fmt.Println("  │   6    BILL_OF_LADING            PROVIDER  SHIPPED                      IN_TRANSIT")
	fmt.Println("  │   7    CUSTOMS_DECLARATION       PROVIDER  CUSTOMS_EXPORT_CLEARED       IN_TRANSIT")
	fmt.Println("  │   8    ARRIVAL_NOTICE            PROVIDER  ARRIVED                      IN_TRANSIT")
	fmt.Println("  │   9    CUSTOMS_DECLARATION       PROVIDER  CUSTOMS_IMPORT_CLEARED       IN_TRANSIT")
	fmt.Println("  │  10    DELIVERY_ORDER            PROVIDER  DELIVERED                    COMPLETED")
	fmt.Println("  │  11    FREIGHT_INVOICE           PROVIDER  INVOICE_RECEIVED             COMPLETED")
	fmt.Println("  └──────────────────────────────────────────────────────────────────────────────────")
}
