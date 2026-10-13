package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestWaitDoesNotBlockOnFirstCall(t *testing.T) {
	l := New(50 * time.Millisecond)

	start := time.Now()
	if err := l.Wait(context.Background(), "host-a"); err != nil {
		t.Fatalf("Wait() 에러: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Fatalf("첫 호출인데 %v나 대기함", elapsed)
	}
}

func TestWaitEnforcesMinIntervalForSameKey(t *testing.T) {
	minInterval := 50 * time.Millisecond
	l := New(minInterval)
	ctx := context.Background()

	start := time.Now()
	if err := l.Wait(ctx, "host-a"); err != nil {
		t.Fatalf("첫 Wait() 에러: %v", err)
	}
	if err := l.Wait(ctx, "host-a"); err != nil {
		t.Fatalf("두번째 Wait() 에러: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < minInterval {
		t.Fatalf("elapsed = %v, want >= %v (최소 간격을 지키지 않음)", elapsed, minInterval)
	}
}

func TestWaitDoesNotBlockAcrossDifferentKeys(t *testing.T) {
	minInterval := 50 * time.Millisecond
	l := New(minInterval)
	ctx := context.Background()

	if err := l.Wait(ctx, "host-a"); err != nil {
		t.Fatalf("Wait(host-a) 에러: %v", err)
	}

	start := time.Now()
	if err := l.Wait(ctx, "host-b"); err != nil {
		t.Fatalf("Wait(host-b) 에러: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Fatalf("다른 key인데 %v나 대기함", elapsed)
	}
}

func TestWaitReturnsErrorWhenContextCancelled(t *testing.T) {
	l := New(time.Hour)
	ctx := context.Background()

	if err := l.Wait(ctx, "host-a"); err != nil {
		t.Fatalf("첫 Wait() 에러: %v", err)
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := l.Wait(cancelCtx, "host-a")
	if err == nil {
		t.Fatal("ctx가 취소됐는데 에러가 없음")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("취소 후에도 %v나 블록됨", elapsed)
	}
}
