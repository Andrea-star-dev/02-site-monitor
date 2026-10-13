// Package config는 감시 대상 목록(targets.yaml)을 읽고 검증한다.
package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Target은 감시할 페이지 하나를 나타낸다.
type Target struct {
	Name          string
	URL           string
	Selector      string
	CheckInterval time.Duration
}

// Config는 targets.yaml 전체 내용이다.
type Config struct {
	Targets []Target
}

// rawTarget은 YAML 그대로의 형태 (CheckInterval이 문자열).
type rawTarget struct {
	Name          string `yaml:"name"`
	URL           string `yaml:"url"`
	Selector      string `yaml:"selector"`
	CheckInterval string `yaml:"check_interval"`
}

type rawConfig struct {
	Targets []rawTarget `yaml:"targets"`
}

// Load는 path의 YAML 파일을 읽어 검증된 Config를 반환한다.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("설정 파일을 읽을 수 없습니다 (%s): %w", path, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패 (%s): %w", path, err)
	}

	if len(raw.Targets) == 0 {
		return nil, fmt.Errorf("%s에 targets가 하나도 없습니다", path)
	}

	seen := make(map[string]bool, len(raw.Targets))
	targets := make([]Target, 0, len(raw.Targets))

	for i, rt := range raw.Targets {
		target, err := parseTarget(rt)
		if err != nil {
			return nil, fmt.Errorf("targets[%d] (%s): %w", i, rt.Name, err)
		}
		if seen[target.Name] {
			return nil, fmt.Errorf("targets[%d]: 이름이 중복됩니다: %s", i, target.Name)
		}
		seen[target.Name] = true
		targets = append(targets, target)
	}

	return &Config{Targets: targets}, nil
}

func parseTarget(rt rawTarget) (Target, error) {
	if rt.Name == "" {
		return Target{}, fmt.Errorf("name이 비어있습니다")
	}
	if rt.Selector == "" {
		return Target{}, fmt.Errorf("selector가 비어있습니다")
	}

	parsedURL, err := url.Parse(rt.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return Target{}, fmt.Errorf("url이 올바르지 않습니다: %q", rt.URL)
	}

	interval, err := time.ParseDuration(rt.CheckInterval)
	if err != nil {
		return Target{}, fmt.Errorf("check_interval이 올바르지 않습니다 (%q): %w", rt.CheckInterval, err)
	}
	if interval <= 0 {
		return Target{}, fmt.Errorf("check_interval은 0보다 커야 합니다: %s", rt.CheckInterval)
	}

	return Target{
		Name:          rt.Name,
		URL:           rt.URL,
		Selector:      rt.Selector,
		CheckInterval: interval,
	}, nil
}
