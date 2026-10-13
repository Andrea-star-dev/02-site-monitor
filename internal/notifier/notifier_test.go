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

func TestTelegramNotifierSendsExpectedRequest(t *testing.T) {
	var capturedPath string
	var capturedChatID, capturedText string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, _ := readBody(r)
		values, _ := url.ParseQuery(body)
		capturedChatID = values.Get("chat_id")
		capturedText = values.Get("text")
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
	if capturedText != "가격 변경: £51.77 -> £45.00" {
		t.Errorf("text = %q", capturedText)
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
