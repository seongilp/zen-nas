package ui

import (
	"fmt"
	"strings"
)

func renderStatusbar(a *App, width int) string {
	var parts []string
	for _, p := range a.pools {
		parts = append(parts, fmt.Sprintf("%s %s", p.Name, p.Health))
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
	right := "[r]새로고침 [q]종료"
	pad := width - len([]rune(left)) - len([]rune(right))
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
