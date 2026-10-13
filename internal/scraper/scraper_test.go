package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"site-monitor/internal/config"
)

func newTestServer(t *testing.T, robotsBody, pageHTML string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		if robotsBody == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(robotsBody))
	})
	mux.HandleFunc("/page.html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageHTML))
	})
	mux.HandleFunc("/private/page.html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageHTML))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

const samplePageHTML = `
<html><body>
  <div class="product"><p class="price_color">£51.77</p></div>
</body></html>
`

func newTestScraper() *Scraper {
	// 테스트에서는 간격을 짧게 둬서 느려지지 않게 함.
	return New(time.Millisecond)
}

func TestExtractReturnsSelectorText(t *testing.T) {
	srv := newTestServer(t, "", samplePageHTML)
	s := newTestScraper()

	target := config.Target{
		Name:          "book-a",
		URL:           srv.URL + "/page.html",
		Selector:      "p.price_color",
		CheckInterval: time.Hour,
	}

	value, err := s.Extract(context.Background(), target)
	if err != nil {
		t.Fatalf("Extract() 에러: %v", err)
	}
	if value != "£51.77" {
		t.Errorf("value = %q, want £51.77", value)
	}
}

func TestExtractReturnsErrorWhenSelectorNotFound(t *testing.T) {
	srv := newTestServer(t, "", samplePageHTML)
	s := newTestScraper()

	target := config.Target{
		Name:          "book-a",
		URL:           srv.URL + "/page.html",
		Selector:      "p.does-not-exist",
		CheckInterval: time.Hour,
	}

	if _, err := s.Extract(context.Background(), target); err == nil {
		t.Fatal("셀렉터가 없는데 에러가 발생하지 않음")
	}
}

func TestExtractBlockedByRobotsTxt(t *testing.T) {
	robotsBody := "User-agent: *\nDisallow: /private/\n"
	srv := newTestServer(t, robotsBody, samplePageHTML)
	s := newTestScraper()

	target := config.Target{
		Name:          "blocked",
		URL:           srv.URL + "/private/page.html",
		Selector:      "p.price_color",
		CheckInterval: time.Hour,
	}

	if _, err := s.Extract(context.Background(), target); err == nil {
		t.Fatal("robots.txt에서 금지된 경로인데 에러가 발생하지 않음")
	}
}

func TestExtractAllowedWhenNoRobotsTxt(t *testing.T) {
	srv := newTestServer(t, "", samplePageHTML)
	s := newTestScraper()

	target := config.Target{
		Name:          "book-a",
		URL:           srv.URL + "/private/page.html",
		Selector:      "p.price_color",
		CheckInterval: time.Hour,
	}

	if _, err := s.Extract(context.Background(), target); err != nil {
		t.Fatalf("robots.txt가 없는데 차단됨: %v", err)
	}
}

func TestExtractRespectsRateLimitAcrossCalls(t *testing.T) {
	srv := newTestServer(t, "", samplePageHTML)
	minInterval := 50 * time.Millisecond
	s := New(minInterval)

	target := config.Target{
		Name:          "book-a",
		URL:           srv.URL + "/page.html",
		Selector:      "p.price_color",
		CheckInterval: time.Hour,
	}

	start := time.Now()
	if _, err := s.Extract(context.Background(), target); err != nil {
		t.Fatalf("첫 Extract() 에러: %v", err)
	}
	if _, err := s.Extract(context.Background(), target); err != nil {
		t.Fatalf("두번째 Extract() 에러: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < minInterval {
		t.Fatalf("elapsed = %v, want >= %v (같은 호스트 요청 간격을 지키지 않음)", elapsed, minInterval)
	}
}
