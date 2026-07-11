package ui

import "github.com/charmbracelet/lipgloss"

const (
	maxContentWidth  = 104
	maxContentHeight = 32
)

var (
	accent      = lipgloss.Color("6") // cyan 계열
	dimColor    = lipgloss.Color("8")
	greenColor  = lipgloss.Color("2")
	yellowColor = lipgloss.Color("3")
	redColor    = lipgloss.Color("1")

	appBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(dimColor)
	healthGood = lipgloss.NewStyle().Foreground(greenColor)
	healthWarn = lipgloss.NewStyle().Foreground(yellowColor)
	healthBad  = lipgloss.NewStyle().Foreground(redColor)
)

// healthStyle: 풀 상태 문자열에 대응하는 색상 스타일.
func healthStyle(h string) lipgloss.Style {
	switch h {
	case "ONLINE":
		return healthGood
	case "DEGRADED":
		return healthWarn
	default:
		return healthBad
	}
}
