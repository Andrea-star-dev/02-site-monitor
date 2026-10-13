// Package scraper는 대상 페이지에서 CSS 셀렉터로 값을 추출한다.
// robots.txt를 확인하고, 같은 호스트에 대해 최소 요청 간격을 지킨다.
package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"site-monitor/internal/config"
	"site-monitor/internal/ratelimit"
	"site-monitor/internal/robots"
)

const defaultUserAgent = "site-monitor/1.0 (+https://github.com/example/site-monitor)"

// Scraper는 HTTP로 페이지를 가져와 CSS 셀렉터로 텍스트를 추출한다.
type Scraper struct {
	client    *http.Client
	userAgent string
	robots    *robots.Checker
	limiter   *ratelimit.Limiter
}

// New는 minInterval 간격으로 같은 호스트를 매너있게 호출하는 Scraper를 만든다.
func New(minInterval time.Duration) *Scraper {
	client := &http.Client{Timeout: 15 * time.Second}
	return &Scraper{
		client:    client,
		userAgent: defaultUserAgent,
		robots:    robots.NewChecker(client, defaultUserAgent),
		limiter:   ratelimit.New(minInterval),
	}
}

// Extract는 target.URL을 가져와 target.Selector에 해당하는 첫 요소의 텍스트를 반환한다.
func (s *Scraper) Extract(ctx context.Context, target config.Target) (string, error) {
	u, err := url.Parse(target.URL)
	if err != nil {
		return "", fmt.Errorf("url 파싱 실패 (%s): %w", target.Name, err)
	}

	allowed, err := s.robots.Allowed(ctx, target.URL)
	if err != nil {
		return "", fmt.Errorf("robots.txt 확인 실패 (%s): %w", target.Name, err)
	}
	if !allowed {
		return "", fmt.Errorf("robots.txt가 이 경로의 수집을 금지함 (%s): %s", target.Name, target.URL)
	}

	if err := s.limiter.Wait(ctx, u.Host); err != nil {
		return "", fmt.Errorf("요청 간격 대기 중 취소됨 (%s): %w", target.Name, err)
	}

	body, err := s.fetch(ctx, target.URL)
	if err != nil {
		return "", fmt.Errorf("페이지 요청 실패 (%s): %w", target.Name, err)
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", fmt.Errorf("HTML 파싱 실패 (%s): %w", target.Name, err)
	}

	sel := doc.Find(target.Selector).First()
	if sel.Length() == 0 {
		return "", fmt.Errorf("셀렉터에 해당하는 요소를 찾을 수 없음 (%s): %q", target.Name, target.Selector)
	}

	return strings.TrimSpace(sel.Text()), nil
}

func (s *Scraper) fetch(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("응답 코드가 올바르지 않음: %d", resp.StatusCode)
	}
	return resp.Body, nil
}
