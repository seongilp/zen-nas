package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"zenfs/zfs"
)

func typeString(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestConfirmRejectsWrongName(t *testing.T) {
	fired := false
	c := newConfirm(zfs.CmdDestroyPool("tank"), "tank", "풀 파괴", func() tea.Msg { fired = true; return nil })
	m := typeString(tea.Model(c), "tonk")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if fired {
		t.Fatal("이름 불일치인데 실행됨")
	}
	if m.(*confirmModel).done {
		t.Fatal("불일치인데 done")
	}
}

func TestConfirmFiresOnExactName(t *testing.T) {
	fired := false
	c := newConfirm(zfs.CmdDestroyPool("tank"), "tank", "풀 파괴", func() tea.Msg { fired = true; return nil })
	m := typeString(tea.Model(c), "tank")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("일치인데 tea.Cmd 미반환")
	}
	cmd()
	if !fired {
		t.Fatal("액션 미실행")
	}
}

func TestConfirmDoesNotFireTwice(t *testing.T) {
	count := 0
	c := newConfirm(zfs.CmdDestroyPool("tank"), "tank", "풀 파괴", func() tea.Msg { count++; return nil })
	m := typeString(tea.Model(c), "tank")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("첫 Enter인데 tea.Cmd 미반환")
	}
	cmd()
	// opDoneMsg 도착 전 두 번째 Enter — 재실행되면 안 된다
	_, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd2 != nil {
		cmd2()
	}
	if count != 1 {
		t.Fatalf("액션이 %d회 실행됨 (1회여야 함)", count)
	}
}

func TestConfirmEscCancels(t *testing.T) {
	c := newConfirm(zfs.CmdDestroyPool("tank"), "tank", "풀 파괴", func() tea.Msg { return nil })
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc는 opDoneMsg를 반환해야 함")
	}
	if _, ok := cmd().(opDoneMsg); !ok {
		t.Fatal("Esc 결과가 opDoneMsg가 아님")
	}
}

func TestConfirmViewShowsCommand(t *testing.T) {
	c := newConfirm(zfs.CmdDestroyPool("tank"), "tank", "풀 파괴", func() tea.Msg { return nil })
	v := c.View()
	if !strings.Contains(v, "zpool destroy tank") {
		t.Fatalf("명령 전문이 보이지 않음: %q", v)
	}
}
