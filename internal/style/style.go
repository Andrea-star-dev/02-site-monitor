// Package style는 터미널에 출력할 때만 ANSI 색을 입힌다 (03 챗봇과 같은
// 인디고/골드 톤). 실제 터미널이 아니거나(파이프, 테스트의 bytes.Buffer 등)
// NO_COLOR가 설정돼 있으면 원문 그대로 반환한다.
package style

import (
	"io"
	"os"
)

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	indigo = "\x1b[38;5;105m" // 브랜드 인디고(#5B63D6)에 가장 가까운 256색
	gold   = "\x1b[38;5;179m" // 브랜드 골드(#D9BE6F)에 가장 가까운 256색
	dim    = "\x1b[2m"
	red    = "\x1b[38;5;203m"
)

// Styler는 하나의 출력 대상(io.Writer)에 대해 색 사용 여부를 한 번 정해두고 재사용한다.
type Styler struct {
	enabled bool
}

// New는 w에 실제로 쓸 때 색을 입힐지 미리 판단해둔다.
func New(w io.Writer) *Styler {
	return &Styler{enabled: isColorEnabled(w)}
}

func isColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (s *Styler) wrap(code, text string) string {
	if !s.enabled {
		return text
	}
	return code + text + reset
}

func (s *Styler) Bold(text string) string   { return s.wrap(bold, text) }
func (s *Styler) Indigo(text string) string { return s.wrap(indigo, text) }
func (s *Styler) Gold(text string) string   { return s.wrap(gold, text) }
func (s *Styler) Dim(text string) string    { return s.wrap(dim, text) }
func (s *Styler) Red(text string) string    { return s.wrap(red, text) }
