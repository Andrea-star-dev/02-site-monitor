// Package robots는 robots.txt를 확인해 특정 경로를 수집해도 되는지 판단한다.
//
// 완전한 스펙 구현은 아니고, User-agent: * 그룹의 Allow/Disallow만
// (가장 긴 경로가 우선, 동률이면 Allow 우선) 해석하는 단순화된 버전이다.
package robots

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type rule struct {
	path  string
	allow bool
}

// Checker는 호스트별 robots.txt를 가져와 캐싱하고 판단한다.
type Checker struct {
	client    *http.Client
	userAgent string

	mu    sync.Mutex
	cache map[string][]rule // key: scheme://host
}

// NewChecker를 만든다. client는 robots.txt를 가져올 때 재사용된다.
func NewChecker(client *http.Client, userAgent string) *Checker {
	return &Checker{
		client:    client,
		userAgent: userAgent,
		cache:     make(map[string][]rule),
	}
}

// Allowed는 rawURL의 경로를 수집해도 되는지 반환한다.
// robots.txt가 없으면(404) 제한 없음으로 간주해 true를 반환한다.
func (c *Checker) Allowed(ctx context.Context, rawURL string) (bool, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("url 파싱 실패: %w", err)
	}

	base := u.Scheme + "://" + u.Host

	c.mu.Lock()
	rules, cached := c.cache[base]
	c.mu.Unlock()

	if !cached {
		rules, err = c.fetchRules(ctx, base)
		if err != nil {
			return false, err
		}
		c.mu.Lock()
		c.cache[base] = rules
		c.mu.Unlock()
	}

	path := u.Path
	if path == "" {
		path = "/"
	}
	return isAllowed(rules, path), nil
}

func (c *Checker) fetchRules(ctx context.Context, base string) ([]rule, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/robots.txt", nil)
	if err != nil {
		return nil, fmt.Errorf("robots.txt 요청 생성 실패: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("robots.txt 요청 실패 (%s): %w", base, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // robots.txt가 없으면 제한 없음
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("robots.txt 응답 코드가 올바르지 않음 (%s): %d", base, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("robots.txt 본문 읽기 실패 (%s): %w", base, err)
	}

	return parseStarGroup(string(body)), nil
}

// parseStarGroup은 robots.txt 본문에서 User-agent: * 그룹의 규칙만 뽑아낸다.
func parseStarGroup(body string) []rule {
	var rules []rule

	inStarGroup := false
	groupHasRules := false

	for _, line := range strings.Split(body, "\n") {
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch key {
		case "user-agent":
			if groupHasRules {
				// 규칙이 이미 나온 뒤에 새 User-agent가 나오면 새 그룹 시작.
				inStarGroup = false
				groupHasRules = false
			}
			if value == "*" {
				inStarGroup = true
			}
		case "disallow":
			if inStarGroup {
				groupHasRules = true
				if value != "" {
					rules = append(rules, rule{path: value, allow: false})
				}
			}
		case "allow":
			if inStarGroup {
				groupHasRules = true
				if value != "" {
					rules = append(rules, rule{path: value, allow: true})
				}
			}
		}
	}

	return rules
}

// isAllowed는 가장 긴 경로가 매칭되는 규칙을 우선하고, 길이가 같으면 allow를 우선한다.
func isAllowed(rules []rule, path string) bool {
	bestLen := -1
	bestAllow := true

	for _, r := range rules {
		if !strings.HasPrefix(path, r.path) {
			continue
		}
		if len(r.path) > bestLen || (len(r.path) == bestLen && r.allow && !bestAllow) {
			bestLen = len(r.path)
			bestAllow = r.allow
		}
	}
	return bestAllow
}
