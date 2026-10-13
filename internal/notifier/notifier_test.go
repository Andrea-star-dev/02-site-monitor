package notifier

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestConsoleNotifierWritesMessage(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf)

	if err := c.Notify(context.Background(), "가격이 바뀌었습니다"); err != nil {
		t.Fatalf("Notify() 에러: %v", err)
	}
	if got := buf.String(); got != "가격이 바뀌었습니다\n" {
		t.Errorf("출력 = %q, want %q", got, "가격이 바뀌었습니다\n")
	}
}

func TestConsoleNotifierWritesMultiLineMessage(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf)

	message := "[book-a] 변경 감지\n이전: £51.77\n현재: £45.00"
	if err := c.Notify(context.Background(), message); err != nil {
		t.Fatalf("Notify() 에러: %v", err)
	}
	// bytes.Buffer는 터미널이 아니므로 색 코드 없이 그대로 출력돼야 한다.
	if got := buf.String(); got != message+"\n" {
		t.Errorf("출력 = %q, want %q", got, message+"\n")
	}
}

func TestTelegramNotifierSendsExpectedRequest(t *testing.T) {
	var capturedPath string
	var capturedChatID, capturedText string

	var capturedParseMode string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := readBody(r)
		values, _ := url.ParseQuery(body)
		capturedChatID = values.Get("chat_id")
		capturedText = values.Get("text")
		capturedParseMode = values.Get("parse_mode")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewTelegram(srv.Client(), "fake-token", "12345")
	n.apiBase = srv.URL // 테스트에서만 실제 텔레그램 서버 대신 가짜 서버로 보냄

	if err := n.Notify(context.Background(), "가격 변경: £51.77 -> £45.00"); err != nil {
		t.Fatalf("Notify() 에러: %v", err)
	}

	if !strings.Contains(capturedPath, "/botfake-token/sendMessage") {
		t.Errorf("요청 경로 = %q, want 포함 /botfake-token/sendMessage", capturedPath)
	}
	if capturedChatID != "12345" {
		t.Errorf("chat_id = %q, want 12345", capturedChatID)
	}
	if capturedParseMode != "HTML" {
		t.Errorf("parse_mode = %q, want HTML", capturedParseMode)
	}
	// 첫 줄(유일한 줄)은 굵게 표시된다.
	if capturedText != "<b>가격 변경: £51.77 -&gt; £45.00</b>" {
		t.Errorf("text = %q", capturedText)
	}
}

func TestTelegramNotifierBoldsOnlyFirstLine(t *testing.T) {
	var capturedText string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := readBody(r)
		values, _ := url.ParseQuery(body)
		capturedText = values.Get("text")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewTelegram(srv.Client(), "fake-token", "12345")
	n.apiBase = srv.URL

	message := "[book-a] 변경 감지\n이전: £51.77\n현재: £45.00"
	if err := n.Notify(context.Background(), message); err != nil {
		t.Fatalf("Notify() 에러: %v", err)
	}

	want := "<b>[book-a] 변경 감지</b>\n이전: £51.77\n현재: £45.00"
	if capturedText != want {
		t.Errorf("text = %q, want %q", capturedText, want)
	}
}

func TestTelegramNotifierReturnsErrorOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	n := NewTelegram(srv.Client(), "fake-token", "12345")
	n.apiBase = srv.URL

	if err := n.Notify(context.Background(), "test"); err == nil {
		t.Fatal("403 응답인데 에러가 발생하지 않음")
	}
}

func TestResolveFallsBackToConsoleWhenTokenMissing(t *testing.T) {
	var buf bytes.Buffer
	n := Resolve(http.DefaultClient, "", "", &buf)

	if _, ok := n.(*ConsoleNotifier); !ok {
		t.Fatalf("Resolve() 타입 = %T, want *ConsoleNotifier", n)
	}
}

func TestResolveFallsBackToConsoleWhenChatIDMissing(t *testing.T) {
	var buf bytes.Buffer
	n := Resolve(http.DefaultClient, "some-token", "", &buf)

	if _, ok := n.(*ConsoleNotifier); !ok {
		t.Fatalf("Resolve() 타입 = %T, want *ConsoleNotifier", n)
	}
}

func TestResolveReturnsTelegramWhenBothSet(t *testing.T) {
	var buf bytes.Buffer
	n := Resolve(http.DefaultClient, "some-token", "some-chat-id", &buf)

	if _, ok := n.(*TelegramNotifier); !ok {
		t.Fatalf("Resolve() 타입 = %T, want *TelegramNotifier", n)
	}
}

func readBody(r *http.Request) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	return buf.String(), err
}
