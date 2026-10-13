// Package monitor는 대상별로 값을 가져와 이전 상태와 비교하고,
// 바뀐 경우에만 이력을 남기고 알린다.
package monitor

import (
	"context"
	"fmt"
	"time"

	"site-monitor/internal/config"
	"site-monitor/internal/notifier"
	"site-monitor/internal/store"
)

// Extractor는 target의 현재 값을 가져온다 (*scraper.Scraper가 이를 만족한다).
type Extractor interface {
	Extract(ctx context.Context, target config.Target) (string, error)
}

// Runner는 한 번의 점검 사이클을 수행한다.
type Runner struct {
	extractor Extractor
	store     *store.Store
	notifier  notifier.Notifier
	now       func() time.Time
}

// New는 Runner를 만든다.
func New(extractor Extractor, s *store.Store, notify notifier.Notifier) *Runner {
	return &Runner{extractor: extractor, store: s, notifier: notify, now: time.Now}
}

// CheckResult는 한 대상에 대한 점검 결과다.
type CheckResult struct {
	Target   config.Target
	IsFirst  bool // 이번 점검이 이 target의 첫 기록인 경우 true (비교 대상이 없어 Changed는 항상 false)
	Changed  bool
	OldValue string
	NewValue string
	Err      error
}

// CheckOne은 target 하나를 점검한다: 값을 가져와 이전 값과 비교하고,
// 바뀌었으면 이력을 남기고 알린 뒤, 항상 최신 상태를 저장한다.
func (r *Runner) CheckOne(ctx context.Context, target config.Target) CheckResult {
	result := CheckResult{Target: target}

	value, err := r.extractor.Extract(ctx, target)
	if err != nil {
		result.Err = fmt.Errorf("값 추출 실패: %w", err)
		return result
	}

	prev, err := r.store.GetState(target.Name)
	if err != nil {
		result.Err = fmt.Errorf("이전 상태 조회 실패: %w", err)
		return result
	}

	now := r.now()
	result.NewValue = value
	result.IsFirst = prev == nil

	if prev != nil && prev.LastValue != value {
		result.Changed = true
		result.OldValue = prev.LastValue

		if err := r.store.RecordChange(target.Name, prev.LastValue, value, now); err != nil {
			result.Err = fmt.Errorf("변경 이력 저장 실패: %w", err)
			return result
		}

		message := fmt.Sprintf(
			"[%s] 변경 감지\n이전: %s\n현재: %s\n%s",
			target.Name, prev.LastValue, value, target.URL,
		)
		if err := r.notifier.Notify(ctx, message); err != nil {
			result.Err = fmt.Errorf("알림 전송 실패: %w", err)
			return result
		}
	}

	if err := r.store.SetState(target.Name, value, now); err != nil {
		result.Err = fmt.Errorf("상태 저장 실패: %w", err)
		return result
	}

	return result
}

// CheckAll은 모든 target을 순서대로 점검한다. 하나가 실패해도 나머지는 계속 점검한다.
func (r *Runner) CheckAll(ctx context.Context, targets []config.Target) []CheckResult {
	results := make([]CheckResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, r.CheckOne(ctx, target))
	}
	return results
}
