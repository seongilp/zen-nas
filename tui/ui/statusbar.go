package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderStatusbar(a *App, width int) string {
	var parts []string
	for _, p := range a.pools {
		parts = append(parts, fmt.Sprintf("%s %s", p.Name, healthStyle(p.Health).Render(p.Health)))
	}
	left := strings.Join(parts, "  ")
	if left == "" {
		left = "풀 없음"
	}
	if a.warnText != "" {
		left += "  ⚠ " + firstLine(a.warnText)
	}
	if a.errText != "" {
		left += "  ⚠ " + firstLine(a.errText)
	}
	right := dimStyle.Render("[r]새로고침 [w]저장 [q]종료")
	// ANSI 색상 코드가 섞인 문자열은 rune 개수와 화면 폭이 다르므로 lipgloss.Width()로 계산한다.
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	line := left + strings.Repeat(" ", pad) + right
	// 풀이 많거나 오류 메시지가 길면 pad 최소값(1)을 적용해도 width를 넘을 수 있다 —
	// 그 경우 상태바가 콘텐츠 박스 폭을 밀어내지 않도록 잘라낸다 (ANSI-safe).
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
