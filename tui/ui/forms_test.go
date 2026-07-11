package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"zenfs/zfs"
)

func TestTextFormRejectsEmptyAndFlagInput(t *testing.T) {
	execCount := 0
	f := newTextForm("t", "", func(v string) zfs.Cmd { return zfs.Cmd{Name: "x"} },
		func(zfs.Cmd) func() tea.Msg { return func() tea.Msg { execCount++; return nil } })
	// 빈 입력 Enter
	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("빈 입력인데 실행")
	}
	// 플래그 주입 시도
	m := typeString(tea.Model(f), "-f")
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("플래그 입력인데 실행")
	}
}

func TestPoolFormSkipsInUseDisks(t *testing.T) {
	disks := []zfs.Disk{{Name: "ada0", Note: "OS 디스크"}, {Name: "ada1"}}
	var got []string
	f := newPoolForm(disks, func(name, vt string, ds []string) tea.Cmd {
		got = ds
		return nil
	})
	// step 0: 이름 입력
	m := typeString(tea.Model(f), "tank")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// step 1: single 선택
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// step 2: 커서는 ada0(사용 불가) — Space 토글 시도 후 Enter (선택 0개 → 무시)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("선택 0개인데 완료됨")
	}
	// ada1로 이동 후 선택
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(got) != 1 || got[0] != "ada1" {
		t.Fatalf("선택 결과: %v", got)
	}
}
