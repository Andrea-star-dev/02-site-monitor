// Package notifier는 변경 알림을 보낸다. 텔레그램 설정이 없으면 콘솔 출력으로 대체한다.
package notifier

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"site-monitor/internal/style"
)

// Notifier는 메시지 하나를 보낸다.
type Notifier interface {
	Notify(ctx context.Context, message string) error
}

// ConsoleNotifier는 메시지를 out에 출력한다 (텔레그램 설정이 없을 때 폴백).
// 실제 터미널에 출력할 때는 첫 줄을 굵은 인디고로, 나머지는 옅게 강조한다.
type ConsoleNotifier struct {
	out   io.Writer
	style *style.Styler
}

// NewConsole은 ConsoleNotifier를 만든다.
func NewConsole(out io.Writer) *ConsoleNotifier {
	return &ConsoleNotifier{out: out, style: style.New(out)}
}

// Notify는 message를 출력한다 (첫 줄 강조, 나머지는 옅게).
func (c *ConsoleNotifier) Notify(_ context.Context, message string) error {
	lines := strings.SplitN(message, "\n", 2)
	first := c.style.Bold(c.style.Indigo(lines[0]))

	if len(lines) == 1 {
		_, err := fmt.Fprintln(c.out, first)
		return err
	}

	rest := c.style.Dim(lines[1])
	_, err := fmt.Fprintln(c.out, first+"\n"+rest)
	return err
}

// formatTelegramHTML은 메시지 첫 줄을 굵게 표시하고 전체를 텔레그램 HTML
// parse_mode 기준으로 이스케이프한다 (<, >, & 가 포함된 URL이나 값이 있어도 안전).
func formatTelegramHTML(message string) string {
	lines := strings.SplitN(message, "\n", 2)
	first := "<b>" + html.EscapeString(lines[0]) + "</b>"
	if len(lines) == 1 {
		return first
	}
	return first + "\n" + html.EscapeString(lines[1])
}

const telegramAPIBase = "https://api.telegram.org"

// TelegramNotifier는 텔레그램 봇 API로 메시지를 보낸다.
type TelegramNotifier struct {
	client  *http.Client
	token   string
	chatID  string
	apiBase string // 테스트에서 가짜 서버로 바꿔 끼우기 위한 필드. 기본값은 실제 텔레그램 API.
}

// NewTelegram은 TelegramNotifier를 만든다.
func NewTelegram(client *http.Client, token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{client: client, token: token, chatID: chatID, apiBase: telegramAPIBase}
}

// Notify는 텔레그램 sendMessage API를 호출한다. 메시지 첫 줄(대상 이름 + 상태)을
// 굵게 표시해 콘솔 출력과 같은 느낌을 낸다 (HTML parse_mode).
func (t *TelegramNotifier) Notify(ctx context.Context, message string) error {
	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.token)

	form := url.Values{
		"chat_id":    {t.chatID},
		"text":       {formatTelegramHTML(message)},
		"parse_mode": {"HTML"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("텔레그램 요청 생성 실패: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("텔레그램 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("텔레그램 응답 코드가 올바르지 않음: %d", resp.StatusCode)
	}
	return nil
}

// Resolve는 token/chatID가 둘 다 있으면 TelegramNotifier, 없으면 ConsoleNotifier를 반환한다.
func Resolve(client *http.Client, token, chatID string, consoleOut io.Writer) Notifier {
	if token == "" || chatID == "" {
		return NewConsole(consoleOut)
	}
	return NewTelegram(client, token, chatID)
}
