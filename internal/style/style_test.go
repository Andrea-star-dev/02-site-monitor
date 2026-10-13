package style

import (
	"bytes"
	"os"
	"testing"
)

func TestDisabledForNonFileWriter(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	if got := s.Bold("hello"); got != "hello" {
		t.Errorf("Bold() = %q, want plain text (bytes.Buffer is never a terminal)", got)
	}
	if got := s.Gold("hello"); got != "hello" {
		t.Errorf("Gold() = %q, want plain text", got)
	}
}

func TestDisabledWhenNoColorSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	// os.Stdout 자체가 터미널이 아닌 상태(go test)에서도 이미 비활성화되지만,
	// NO_COLOR가 우선권을 가져야 한다는 걸 명시적으로 확인한다.
	s := New(os.Stdout)
	if got := s.Indigo("x"); got != "x" {
		t.Errorf("Indigo() = %q, want plain text when NO_COLOR is set", got)
	}
}

func TestWrapFunctionsReturnOriginalTextContent(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	for _, fn := range []func(string) string{s.Bold, s.Indigo, s.Gold, s.Dim, s.Red} {
		if got := fn("payload"); got != "payload" {
			t.Errorf("styled text = %q, want to contain original payload unchanged when disabled", got)
		}
	}
}
