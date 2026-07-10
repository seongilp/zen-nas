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

var modalStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).Padding(1, 2)

func (c *confirmModel) View() string {
	s := fmt.Sprintf("⚠ %s\n\n실행될 명령:\n  %s\n", c.title, c.cmd.String())
	if c.warning != "" {
		s += "\n" + c.warning + "\n"
	}
	s += fmt.Sprintf("\n계속하려면 대상 이름(%s)을 입력 후 Enter (Esc 취소):\n%s", c.target, c.input.View())
	return modalStyle.Render(s)
}
