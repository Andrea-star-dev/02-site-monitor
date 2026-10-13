// Package ratelimit은 같은 대상(예: 같은 호스트)에 대한 요청 간격을 보장한다.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter는 key별로 최소 호출 간격을 강제한다.
type Limiter struct {
	mu          sync.Mutex
	lastAt      map[string]time.Time
	minInterval time.Duration
}

// New는 key당 최소 minInterval 간격을 강제하는 Limiter를 만든다.
func New(minInterval time.Duration) *Limiter {
	return &Limiter{
		lastAt:      make(map[string]time.Time),
		minInterval: minInterval,
	}
}

// Wait는 key에 대해 이전 호출로부터 minInterval이 지날 때까지 블록한다.
// 동시에 여러 goroutine이 같은 key로 호출해도 간격이 보장된다.
// ctx가 취소되면 대기를 중단하고 ctx.Err()를 반환한다.
func (l *Limiter) Wait(ctx context.Context, key string) error {
	l.mu.Lock()
	now := time.Now()
	wait := time.Duration(0)

	if last, ok := l.lastAt[key]; ok {
		if elapsed := now.Sub(last); elapsed < l.minInterval {
			wait = l.minInterval - elapsed
		}
	}
	// 다음 호출이 올바르게 줄을 서도록, 실제 대기 전에 예약된 시각을 기록한다.
	l.lastAt[key] = now.Add(wait)
	l.mu.Unlock()

	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
