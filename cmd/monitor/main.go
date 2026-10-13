// site-monitor는 지정한 웹페이지의 가격/게시글 변화를 감지해 알린다.
//
// 설정 파일(targets.yaml)에 있는 모든 대상을 한 번 점검하고, 이전 값과
// 다르면 텔레그램으로 알린다 (TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID가 없으면
// 콘솔에 출력). 주기적으로 실행하려면 cron이나 systemd timer, 또는 Docker의
// 재시작 정책과 함께 사용한다.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"site-monitor/internal/config"
	"site-monitor/internal/monitor"
	"site-monitor/internal/notifier"
	"site-monitor/internal/scraper"
	"site-monitor/internal/store"
	"site-monitor/internal/style"
)

func main() {
	targetsPath := flag.String("targets", "config/targets.yaml", "감시 대상 YAML 경로")
	dbPath := flag.String("db", "site_monitor.db", "SQLite DB 파일 경로")
	minInterval := flag.Duration("min-interval", 3*time.Second, "같은 호스트에 대한 최소 요청 간격")
	flag.Parse()

	if err := run(*targetsPath, *dbPath, *minInterval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(targetsPath, dbPath string, minInterval time.Duration) error {
	cfg, err := config.Load(targetsPath)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer db.Close()

	notify := notifier.Resolve(
		&http.Client{Timeout: 10 * time.Second},
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_CHAT_ID"),
		os.Stdout,
	)

	s := scraper.New(minInterval)
	runner := monitor.New(s, db, notify)

	fmt.Printf("감시 대상 %d개 점검 시작 (DB: %s)\n", len(cfg.Targets), dbPath)

	results := runner.CheckAll(context.Background(), cfg.Targets)

	out := style.New(os.Stdout)
	errOut := style.New(os.Stderr)

	hadError := false
	for _, r := range results {
		name := out.Indigo(r.Target.Name)
		switch {
		case r.Err != nil:
			hadError = true
			fmt.Fprintf(os.Stderr, "  [%s] %s: %v\n", errOut.Indigo(r.Target.Name), errOut.Red("실패"), r.Err)
		case r.Changed:
			fmt.Printf("  [%s] %s: %s -> %s\n", name, out.Bold(out.Gold("변경됨")), r.OldValue, r.NewValue)
		case r.IsFirst:
			fmt.Printf("  [%s] %s: %s\n", name, out.Dim("최초 등록"), r.NewValue)
		default:
			fmt.Printf("  [%s] %s (%s)\n", name, out.Dim("변경 없음"), r.NewValue)
		}
	}

	if hadError {
		return fmt.Errorf("일부 대상 점검에 실패했습니다")
	}
	return nil
}
