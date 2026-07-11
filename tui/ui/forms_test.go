package ui

import (
	"strings"
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

// App.Update를 통해 P 마법사 → 확인 모달 전환이 실제로 일어나는지 검증하는 통합 테스트.
// (모달 델리게이션 `a.modal, cmd = a.modal.Update(m)`이 finish 클로저의
// a.modal side effect를 반환값으로 덮어쓰는 버그를 잡는다.)
func TestPoolWizardTransitionsToConfirmViaAppUpdate(t *testing.T) {
	a := NewApp(nil)
	a.disks = []zfs.Disk{{Name: "ada1"}}
	var m tea.Model = a
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	if _, ok := a.modal.(*poolForm); !ok {
		t.Fatalf("P 키인데 풀 마법사가 안 열림: %T", a.modal)
	}
	m = typeString(m, "tank")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // step 0: 이름 확정
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // step 1: single 선택
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // step 2: ada1 토글
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("마법사 완료인데 tea.Cmd 미반환 — 확인 모달 전환 불가")
	}
	m.Update(cmd()) // showModalMsg를 App으로 전달
	cm, ok := a.modal.(*confirmModel)
	if !ok {
		t.Fatalf("확인 모달로 전환 안 됨: 모달이 %T", a.modal)
	}
	if !strings.Contains(cm.cmd.String(), "zpool create tank ada1") {
		t.Fatalf("확인 모달 명령이 다름: %s", cm.cmd.String())
	}
}
