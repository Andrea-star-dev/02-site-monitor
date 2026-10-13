package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("테스트 설정 파일 작성 실패: %v", err)
	}
	return path
}

func TestLoadParsesValidConfig(t *testing.T) {
	path := writeConfig(t, `
targets:
  - name: "book-a"
    url: "https://books.toscrape.com/a.html"
    selector: "p.price_color"
    check_interval: "30m"
  - name: "book-b"
    url: "https://books.toscrape.com/b.html"
    selector: "p.price_color"
    check_interval: "1h"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() 실패: %v", err)
	}
	if len(cfg.Targets) != 2 {
		t.Fatalf("targets 개수 = %d, want 2", len(cfg.Targets))
	}
	if cfg.Targets[0].Name != "book-a" {
		t.Errorf("Targets[0].Name = %q, want book-a", cfg.Targets[0].Name)
	}
	if cfg.Targets[1].CheckInterval != time.Hour {
		t.Errorf("Targets[1].CheckInterval = %v, want 1h", cfg.Targets[1].CheckInterval)
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	path := writeConfig(t, `
targets:
  - name: "dup"
    url: "https://example.com/a"
    selector: "p"
    check_interval: "1h"
  - name: "dup"
    url: "https://example.com/b"
    selector: "p"
    check_interval: "1h"
`)

	if _, err := Load(path); err == nil {
		t.Fatal("중복 이름인데 에러가 발생하지 않음")
	}
}

func TestLoadRejectsInvalidURL(t *testing.T) {
	path := writeConfig(t, `
targets:
  - name: "bad-url"
    url: "not-a-url"
    selector: "p"
    check_interval: "1h"
`)

	if _, err := Load(path); err == nil {
		t.Fatal("잘못된 URL인데 에러가 발생하지 않음")
	}
}

func TestLoadRejectsInvalidInterval(t *testing.T) {
	path := writeConfig(t, `
targets:
  - name: "bad-interval"
    url: "https://example.com"
    selector: "p"
    check_interval: "not-a-duration"
`)

	if _, err := Load(path); err == nil {
		t.Fatal("잘못된 interval인데 에러가 발생하지 않음")
	}
}

func TestLoadRejectsEmptyTargets(t *testing.T) {
	path := writeConfig(t, `targets: []`)

	if _, err := Load(path); err == nil {
		t.Fatal("targets가 비었는데 에러가 발생하지 않음")
	}
}

func TestLoadReturnsErrorForMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("존재하지 않는 파일인데 에러가 발생하지 않음")
	}
}
