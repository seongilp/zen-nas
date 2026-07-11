package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"zenfs/zfs"
)

// confirmModel: 위험 작업 확인 모달. 대상 이름을 정확히 타이핑해야 실행.
type confirmModel struct {
	cmd     zfs.Cmd
	target  string
	title   string
	warning string
	input   textinput.Model
	action  func() tea.Msg
	done    bool
}

func newConfirm(cmd zfs.Cmd, target, title string, action func() tea.Msg) *confirmModel {
	ti := textinput.New()
	ti.Focus()
	return &confirmModel{cmd: cmd, target: target, title: title, action: action, input: ti}
}

func (c *confirmModel) Init() tea.Cmd { return nil }

func (c *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if c.done { // 액션 발사 후 opDoneMsg 도착 전 재입력 무시 — 이중 실행 방지
		return c, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.Type {
		case tea.KeyEsc:
			return c, func() tea.Msg { return opDoneMsg{} }
		case tea.KeyEnter:
			if c.input.Value() == c.target {
				c.done = true
				return c, c.action
			}
			return c, nil
		}
	}
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

var modalStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(accent).Padding(1, 2)

// errModal: 작업 실패 시 stderr 전문을 표시하는 모달. Enter/Esc로 닫는다.
type errModal struct{ text string }

func (e *errModal) Init() tea.Cmd { return nil }

func (e *errModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && (k.Type == tea.KeyEsc || k.Type == tea.KeyEnter) {
		return e, func() tea.Msg { return opDoneMsg{} }
	}
	return e, nil
}

func (e *errModal) View() string {
	return modalStyle.Render("⚠ 작업 실패\n\n" + e.text + "\n\n(Enter/Esc 닫기)")
}

func (c *confirmModel) View() string {
	s := fmt.Sprintf("⚠ %s\n\n실행될 명령:\n  %s\n", c.title, c.cmd.String())
	if c.warning != "" {
		s += "\n" + c.warning + "\n"
	}
	s += fmt.Sprintf("\n계속하려면 대상 이름(%s)을 입력 후 Enter (Esc 취소):\n%s", c.target, c.input.View())
	return modalStyle.Render(s)
}
