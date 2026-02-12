package tracking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// SyncTrackingEventsUseCase: トラッキングイベント同期ユースケース
// セグメント毎にプロバイダーからイベントを取得し、TrackingUnitに適用する
type SyncTrackingEventsUseCase struct {
	trackingRepo     domain.TrackingUnitRepository
	providerRegistry domain.TrackingDataProviderRegistry
}

// NewSyncTrackingEventsUseCase: コンストラクタ
func NewSyncTrackingEventsUseCase(
	trackingRepo domain.TrackingUnitRepository,
	providerRegistry domain.TrackingDataProviderRegistry,
) *SyncTrackingEventsUseCase {
	return &SyncTrackingEventsUseCase{
		trackingRepo:     trackingRepo,
		providerRegistry: providerRegistry,
	}
}

// Execute: ユースケースを実行
func (uc *SyncTrackingEventsUseCase) Execute(
	ctx context.Context,
	input SyncTrackingInput,
) (*SyncTrackingOutput, error) {
	// 1. 入力バリデーション
	if input.TrackingUnitID == uuid.Nil {
		return nil, NewSyncTrackingError("INVALID_INPUT", "tracking unit ID is required")
	}

	// 2. TrackingUnit取得
	unit, err := uc.trackingRepo.FindByID(ctx, input.TrackingUnitID)
	if err != nil {
		return nil, NewSyncTrackingError("TRACKING_FETCH_ERROR", "failed to fetch tracking unit").
			WithDetail("trackingUnitID", input.TrackingUnitID).
			WithCause(err)
	}
	if unit == nil {
		return nil, NewSyncTrackingError("TRACKING_NOT_FOUND", "tracking unit not found").
			WithDetail("trackingUnitID", input.TrackingUnitID)
	}

	// 3. 各セグメントの同期処理
	segments := unit.Segments()
	syncedSegments := make([]SyncedSegmentDetail, 0, len(segments))
	totalNewEvents := 0

	for _, seg := range segments {
		detail := uc.syncSegment(ctx, unit, seg)
		totalNewEvents += detail.NewEventsCount
		syncedSegments = append(syncedSegments, detail)
	}

	// 4. 永続化
	if err := uc.trackingRepo.Save(ctx, unit); err != nil {
		return nil, NewSyncTrackingError("TRACKING_SAVE_ERROR", "failed to save tracking unit").
			WithCause(err)
	}

	// 5. 出力DTOの作成
	output := &SyncTrackingOutput{
		TrackingUnitID: input.TrackingUnitID,
		OverallStatus:  unit.CurrentStatus(),
		SyncedSegments: syncedSegments,
		TotalNewEvents: totalNewEvents,
		SyncedAt:       time.Now(),
	}

	return output, nil
}

// syncSegment: セグメント単位の同期処理
// セグメント単位のエラーは記録して続行（部分的成功を許容）
func (uc *SyncTrackingEventsUseCase) syncSegment(
	ctx context.Context,
	unit *domain.TrackingUnit,
	seg *domain.TrackingSegment,
) SyncedSegmentDetail {
	detail := SyncedSegmentDetail{
		SegmentID:    seg.ID,
		LatestStatus: seg.Status,
	}

	// ARRIVEDセグメントはスキップ
	if seg.Status == domain.StatusArrived {
		return detail
	}

	// プロバイダー解決
	provider, err := uc.providerRegistry.GetProvider(seg.PrimarySource)
	if err != nil {
		detail.Error = fmt.Sprintf("provider not found for source: %s", seg.PrimarySource)
		return detail
	}

	// イベント取得
	query := domain.TrackingQuery{
		TrackingNumber: seg.CarrierTrackingNumber,
		Mode:           seg.Mode,
	}
	events, err := provider.FetchEvents(ctx, query)
	if err != nil {
		detail.Error = fmt.Sprintf("failed to fetch events: %v", err)
		return detail
	}

	// 重複チェック用Set構築: 既存イベントのCode+Timestamp
	existingSet := make(map[string]struct{})
	for _, existing := range seg.Events {
		key := buildEventKey(existing.Code, existing.Timestamp)
		existingSet[key] = struct{}{}
	}

	// 新規イベントの適用
	newCount := 0
	for _, event := range events {
		key := buildEventKey(event.Code, event.Timestamp)
		if _, exists := existingSet[key]; exists {
			continue // 重複スキップ
		}
		if err := unit.UpdateSegmentStatus(seg.ID, event); err != nil {
			detail.Error = fmt.Sprintf("failed to apply event: %v", err)
			return detail
		}
		newCount++
	}

	detail.NewEventsCount = newCount
	detail.LatestStatus = seg.Status

	return detail
}

// buildEventKey: 重複チェック用キーを生成（Code+Timestamp）
func buildEventKey(code string, timestamp time.Time) string {
	return fmt.Sprintf("%s:%d", code, timestamp.UnixNano())
}
