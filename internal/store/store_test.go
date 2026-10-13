package store

import (
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() 실패: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenCreatesMissingParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "dir", "site_monitor.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() 실패 (상위 폴더가 없을 때): %v", err)
	}
	defer s.Close()

	if err := s.SetState("book-a", "£51.77", time.Now()); err != nil {
		t.Fatalf("SetState() 실패: %v", err)
	}
}

func TestGetStateReturnsNilWhenNeverChecked(t *testing.T) {
	s := openTestStore(t)

	state, err := s.GetState("unknown-target")
	if err != nil {
		t.Fatalf("GetState() 에러: %v", err)
	}
	if state != nil {
		t.Fatalf("state = %+v, want nil", state)
	}
}

func TestSetStateThenGetStateRoundTrips(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	if err := s.SetState("book-a", "£51.77", now); err != nil {
		t.Fatalf("SetState() 에러: %v", err)
	}

	state, err := s.GetState("book-a")
	if err != nil {
		t.Fatalf("GetState() 에러: %v", err)
	}
	if state == nil {
		t.Fatal("state = nil, want non-nil")
	}
	if state.LastValue != "£51.77" {
		t.Errorf("LastValue = %q, want £51.77", state.LastValue)
	}
	if !state.LastCheckedAt.Equal(now) {
		t.Errorf("LastCheckedAt = %v, want %v", state.LastCheckedAt, now)
	}
}

func TestSetStateUpsertsExistingTarget(t *testing.T) {
	s := openTestStore(t)
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(time.Hour)

	if err := s.SetState("book-a", "£51.77", t1); err != nil {
		t.Fatalf("첫 SetState() 에러: %v", err)
	}
	if err := s.SetState("book-a", "£45.00", t2); err != nil {
		t.Fatalf("두번째 SetState() 에러: %v", err)
	}

	state, err := s.GetState("book-a")
	if err != nil {
		t.Fatalf("GetState() 에러: %v", err)
	}
	if state.LastValue != "£45.00" {
		t.Errorf("LastValue = %q, want £45.00 (최신값으로 덮어써야 함)", state.LastValue)
	}
}

func TestRecordChangeAndHistory(t *testing.T) {
	s := openTestStore(t)
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(time.Hour)

	if err := s.RecordChange("book-a", "£51.77", "£45.00", t1); err != nil {
		t.Fatalf("RecordChange() 에러: %v", err)
	}
	if err := s.RecordChange("book-a", "£45.00", "£40.00", t2); err != nil {
		t.Fatalf("RecordChange() 에러: %v", err)
	}

	history, err := s.History("book-a")
	if err != nil {
		t.Fatalf("History() 에러: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	// 최신순이어야 함
	if history[0].NewValue != "£40.00" {
		t.Errorf("history[0].NewValue = %q, want £40.00 (최신순)", history[0].NewValue)
	}
}

func TestHistoryEmptyForUnknownTarget(t *testing.T) {
	s := openTestStore(t)

	history, err := s.History("never-recorded")
	if err != nil {
		t.Fatalf("History() 에러: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("len(history) = %d, want 0", len(history))
	}
}
