package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"zenfs/zfs"
)

// fakeRunner: 실행된 명령 전문을 기록하는 테스트용 Runner.
type fakeRunner struct{ calls []string }

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return "", "", nil
}

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

// 디스크 교체(x)도 파괴·롤백처럼 이름 타이핑 확인 게이트를 거쳐야 한다.
func TestReplaceGoesThroughConfirm(t *testing.T) {
	r := &fakeRunner{}
	a := NewApp(r)
	a.nodes = []treeNode{{kind: nodePool, name: "tank"}}
	var m tea.Model = a
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if _, ok := a.modal.(*textForm); !ok {
		t.Fatalf("x 키인데 입력 폼이 안 열림: %T", a.modal)
	}
	m = typeString(m, "ada1 ada3")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter인데 tea.Cmd 미반환")
	}
	m.Update(cmd())
	cm, ok := a.modal.(*confirmModel)
	if !ok {
		t.Fatalf("확인 게이트 없이 진행됨: 모달이 %T (실행 기록: %v)", a.modal, r.calls)
	}
	if !strings.Contains(cm.cmd.String(), "zpool replace tank ada1 ada3") {
		t.Fatalf("확인 모달 명령이 다름: %s", cm.cmd.String())
	}
	if len(r.calls) != 0 {
		t.Fatalf("확인 전에 이미 실행됨: %v", r.calls)
	}
	// 풀 이름 타이핑 후 Enter → 그제서야 실행
	m = typeString(m, "tank")
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("이름 일치인데 실행 cmd 미반환")
	}
	m.Update(cmd())
	if len(r.calls) != 1 || !strings.Contains(r.calls[0], "zpool replace tank ada1 ada3") {
		t.Fatalf("실행 기록이 다름: %v", r.calls)
	}
}
