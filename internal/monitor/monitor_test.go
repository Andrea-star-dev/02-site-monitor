package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"site-monitor/internal/config"
	"site-monitor/internal/store"
)

type fakeExtractor struct {
	value string
	err   error
}

func (f *fakeExtractor) Extract(_ context.Context, _ config.Target) (string, error) {
	return f.value, f.err
}

type fakeNotifier struct {
	messages []string
	err      error
}

func (f *fakeNotifier) Notify(_ context.Context, message string) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, message)
	return nil
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open() 실패: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var sampleTarget = config.Target{
	Name:          "book-a",
	URL:           "https://example.com/book-a",
	Selector:      "p.price_color",
	CheckInterval: time.Hour,
}

func TestCheckOneFirstCheckDoesNotNotify(t *testing.T) {
	extractor := &fakeExtractor{value: "£51.77"}
	notify := &fakeNotifier{}
	s := newTestStore(t)
	r := New(extractor, s, notify)

	result := r.CheckOne(context.Background(), sampleTarget)

	if result.Err != nil {
		t.Fatalf("CheckOne() 에러: %v", result.Err)
	}
	if result.Changed {
		t.Error("첫 점검인데 변경으로 표시됨")
	}
	if !result.IsFirst {
		t.Error("첫 점검인데 IsFirst = false")
	}
	if len(notify.messages) != 0 {
		t.Errorf("첫 점검인데 알림이 발생함: %v", notify.messages)
	}

	state, err := s.GetState(sampleTarget.Name)
	if err != nil {
		t.Fatalf("GetState() 에러: %v", err)
	}
	if state == nil || state.LastValue != "£51.77" {
		t.Errorf("상태가 저장되지 않음: %+v", state)
	}
}

func TestCheckOneSameValueDoesNotNotify(t *testing.T) {
	extractor := &fakeExtractor{value: "£51.77"}
	notify := &fakeNotifier{}
	s := newTestStore(t)
	r := New(extractor, s, notify)
	ctx := context.Background()

	r.CheckOne(ctx, sampleTarget) // 첫 점검으로 기준값 설정
	result := r.CheckOne(ctx, sampleTarget)

	if result.Changed {
		t.Error("값이 같은데 변경으로 표시됨")
	}
	if result.IsFirst {
		t.Error("두번째 점검인데 IsFirst = true")
	}
	if len(notify.messages) != 0 {
		t.Errorf("값이 같은데 알림이 발생함: %v", notify.messages)
	}
}

func TestCheckOneChangedValueNotifiesAndRecordsHistory(t *testing.T) {
	extractor := &fakeExtractor{value: "£51.77"}
	notify := &fakeNotifier{}
	s := newTestStore(t)
	r := New(extractor, s, notify)
	ctx := context.Background()

	r.CheckOne(ctx, sampleTarget) // 기준값: £51.77

	extractor.value = "£45.00"
	result := r.CheckOne(ctx, sampleTarget)

	if result.Err != nil {
		t.Fatalf("CheckOne() 에러: %v", result.Err)
	}
	if !result.Changed {
		t.Fatal("값이 바뀌었는데 Changed = false")
	}
	if result.OldValue != "£51.77" || result.NewValue != "£45.00" {
		t.Errorf("OldValue/NewValue = %q/%q", result.OldValue, result.NewValue)
	}
	if len(notify.messages) != 1 {
		t.Fatalf("알림 개수 = %d, want 1", len(notify.messages))
	}

	history, err := s.History(sampleTarget.Name)
	if err != nil {
		t.Fatalf("History() 에러: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("이력 개수 = %d, want 1", len(history))
	}
	if history[0].OldValue != "£51.77" || history[0].NewValue != "£45.00" {
		t.Errorf("이력 값이 올바르지 않음: %+v", history[0])
	}

	state, _ := s.GetState(sampleTarget.Name)
	if state.LastValue != "£45.00" {
		t.Errorf("최신 상태가 갱신되지 않음: %q", state.LastValue)
	}
}

func TestCheckOneReturnsErrorWhenExtractFails(t *testing.T) {
	extractor := &fakeExtractor{err: errors.New("네트워크 실패")}
	notify := &fakeNotifier{}
	s := newTestStore(t)
	r := New(extractor, s, notify)

	result := r.CheckOne(context.Background(), sampleTarget)

	if result.Err == nil {
		t.Fatal("추출 실패인데 에러가 없음")
	}
	if len(notify.messages) != 0 {
		t.Error("추출 실패인데 알림이 발생함")
	}
}

func TestCheckAllContinuesAfterOneTargetFails(t *testing.T) {
	notify := &fakeNotifier{}
	s := newTestStore(t)

	failing := config.Target{Name: "fails", URL: "https://example.com/fails", Selector: "p", CheckInterval: time.Hour}
	ok := config.Target{Name: "ok", URL: "https://example.com/ok", Selector: "p", CheckInterval: time.Hour}

	// 타겟마다 다른 결과가 필요하므로, Extract 안에서 타겟 이름으로 분기하는 fake를 직접 작성.
	extractor := &perTargetExtractor{results: map[string]string{"ok": "value-1"}}
	r := New(extractor, s, notify)

	results := r.CheckAll(context.Background(), []config.Target{failing, ok})

	if len(results) != 2 {
		t.Fatalf("결과 개수 = %d, want 2", len(results))
	}
	if results[0].Err == nil {
		t.Error("failing 타겟인데 에러가 없음")
	}
	if results[1].Err != nil {
		t.Errorf("ok 타겟인데 에러 발생: %v", results[1].Err)
	}
}

type perTargetExtractor struct {
	results map[string]string
}

func (p *perTargetExtractor) Extract(_ context.Context, target config.Target) (string, error) {
	value, ok := p.results[target.Name]
	if !ok {
		return "", errors.New("값을 찾을 수 없음")
	}
	return value, nil
}
