package robots

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServer(t *testing.T, robotsBody string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(robotsBody))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAllowedWhenNoRobotsTxt(t *testing.T) {
	srv := newServer(t, "", http.StatusNotFound)
	c := NewChecker(srv.Client(), "test-agent/1.0")

	allowed, err := c.Allowed(context.Background(), srv.URL+"/any/page")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if !allowed {
		t.Fatal("robots.txt가 없는데 차단됨")
	}
}

func TestDisallowedPathIsBlocked(t *testing.T) {
	robotsBody := "User-agent: *\nDisallow: /private/\n"
	srv := newServer(t, robotsBody, http.StatusOK)
	c := NewChecker(srv.Client(), "test-agent/1.0")

	allowed, err := c.Allowed(context.Background(), srv.URL+"/private/secret.html")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if allowed {
		t.Fatal("/private/ 경로가 Disallow인데 허용됨")
	}

	allowed, err = c.Allowed(context.Background(), srv.URL+"/public/page.html")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if !allowed {
		t.Fatal("/public/ 경로는 허용돼야 하는데 차단됨")
	}
}

func TestMoreSpecificAllowOverridesDisallow(t *testing.T) {
	robotsBody := "User-agent: *\nDisallow: /catalogue/\nAllow: /catalogue/page-1.html\n"
	srv := newServer(t, robotsBody, http.StatusOK)
	c := NewChecker(srv.Client(), "test-agent/1.0")

	allowed, err := c.Allowed(context.Background(), srv.URL+"/catalogue/page-1.html")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if !allowed {
		t.Fatal("더 구체적인 Allow 규칙이 있는데 차단됨")
	}

	allowed, err = c.Allowed(context.Background(), srv.URL+"/catalogue/page-2.html")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if allowed {
		t.Fatal("Allow 규칙에 해당하지 않는 다른 하위 경로가 허용됨")
	}
}

func TestIgnoresOtherUserAgentGroups(t *testing.T) {
	robotsBody := "User-agent: SomeBot\nDisallow: /\n\nUser-agent: *\nDisallow: /private/\n"
	srv := newServer(t, robotsBody, http.StatusOK)
	c := NewChecker(srv.Client(), "test-agent/1.0")

	allowed, err := c.Allowed(context.Background(), srv.URL+"/public/page.html")
	if err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if !allowed {
		t.Fatal("다른 봇 전용 규칙 때문에 잘못 차단됨 (우리는 * 그룹만 봐야 함)")
	}
}

func TestResultIsCachedPerHost(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /private/\n"))
	}))
	defer srv.Close()

	c := NewChecker(srv.Client(), "test-agent/1.0")
	ctx := context.Background()

	if _, err := c.Allowed(ctx, srv.URL+"/a"); err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}
	if _, err := c.Allowed(ctx, srv.URL+"/b"); err != nil {
		t.Fatalf("Allowed() 에러: %v", err)
	}

	if calls != 1 {
		t.Fatalf("robots.txt 요청 횟수 = %d, want 1 (캐시돼야 함)", calls)
	}
}
