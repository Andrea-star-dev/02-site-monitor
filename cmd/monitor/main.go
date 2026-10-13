// site-monitor는 지정한 웹페이지의 가격/게시글 변화를 감지해 알린다.
//
// 현재(Day 1)는 설정 파일을 읽고 DB를 초기화하는 뼈대만 동작한다.
// 실제 스크래핑/알림 로직은 이후 단계에서 추가된다.
package main

import (
	"flag"
	"fmt"
	"os"

	"site-monitor/internal/config"
	"site-monitor/internal/store"
)

func main() {
	targetsPath := flag.String("targets", "config/targets.yaml", "감시 대상 YAML 경로")
	dbPath := flag.String("db", "site_monitor.db", "SQLite DB 파일 경로")
	flag.Parse()

	cfg, err := config.Load(*targetsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "설정 로드 실패: %v\n", err)
		os.Exit(1)
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB 열기 실패: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("감시 대상 %d개를 불러왔습니다 (DB: %s)\n", len(cfg.Targets), *dbPath)
	for _, t := range cfg.Targets {
		fmt.Printf("  - %s (%s, 주기 %s)\n", t.Name, t.URL, t.CheckInterval)
	}
}
