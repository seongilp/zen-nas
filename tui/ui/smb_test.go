package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"zenfs/smb"
	"zenfs/zfs"
)

func tmpSharesConf(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "smb-shares.conf")
	if err := os.WriteFile(p, []byte("# zenfs managed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// m 키: 마운트된 데이터셋이 미공유 상태면 이름 입력 폼 → Enter로 공유 추가 + reload.
func TestMKeyAddsShareAndReloads(t *testing.T) {
	r := &fakeRunner{}
	a := NewApp(r)
	a.sharesPath = tmpSharesConf(t)
	a.datasets = []zfs.Dataset{{Name: "tank/data", Type: zfs.TypeFilesystem, Mountpoint: "/tank/data"}}
	a.nodes = []treeNode{{kind: nodeDataset, name: "tank/data"}}
	a.cursor = 0

	var m tea.Model = a
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	tf, ok := a.modal.(*textForm)
	if !ok {
		t.Fatalf("m 키인데 입력 폼이 안 열림: %T", a.modal)
	}
	if got := tf.input.Value(); got != "data" {
		t.Fatalf("기본 공유 이름이 데이터셋 shortName이 아님: %q", got)
	}
	// 폼은 shortName("tank/data")="data"로 이미 채워져 있으므로 그대로 Enter.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter인데 tea.Cmd 미반환")
	}
	m.Update(cmd())

	shares, err := smb.List(a.sharesPath)
	if err != nil || len(shares) != 1 || shares[0].Name != "data" || shares[0].Path != "/tank/data" {
		t.Fatalf("공유 파일 반영 안 됨: shares=%v err=%v", shares, err)
	}
	if len(r.calls) != 1 || !strings.Contains(r.calls[0], "service samba_server reload") {
		t.Fatalf("reload 미실행: %v", r.calls)
	}
}

// m 키: 이미 공유 중인 데이터셋이면 즉시 해제 + reload, 영속화 안내 표시.
func TestMKeyRemovesShareAndReloads(t *testing.T) {
	r := &fakeRunner{}
	a := NewApp(r)
	a.sharesPath = tmpSharesConf(t)
	if err := smb.Add(a.sharesPath, smb.Share{Name: "data", Path: "/tank/data"}); err != nil {
		t.Fatal(err)
	}
	a.shares, _ = smb.List(a.sharesPath)
	a.datasets = []zfs.Dataset{{Name: "tank/data", Type: zfs.TypeFilesystem, Mountpoint: "/tank/data"}}
	a.nodes = []treeNode{{kind: nodeDataset, name: "tank/data"}}
	a.cursor = 0

	var m tea.Model = a
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("m 키인데 tea.Cmd 미반환 (reload 실행)")
	}
	if a.modal != nil {
		t.Fatalf("공유 해제인데 모달이 열림: %T", a.modal)
	}
	m.Update(cmd())

	shares, err := smb.List(a.sharesPath)
	if err != nil || len(shares) != 0 {
		t.Fatalf("공유 해제 안 됨: shares=%v err=%v", shares, err)
	}
	if len(r.calls) != 1 || !strings.Contains(r.calls[0], "service samba_server reload") {
		t.Fatalf("reload 미실행: %v", r.calls)
	}
	if !strings.Contains(a.errText, "[w]") {
		t.Fatalf("영속화 안내 누락: %q", a.errText)
	}
}

// w 키: save_cfg 명령이 실행되어야 한다 (경로/상태 무관 — 전역 키).
func TestWKeySavesCfg(t *testing.T) {
	r := &fakeRunner{}
	a := NewApp(r)
	var m tea.Model = a
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd == nil {
		t.Fatal("w 키인데 tea.Cmd 미반환")
	}
	cmd()
	if len(r.calls) != 1 || !strings.Contains(r.calls[0], "save_cfg") {
		t.Fatalf("save_cfg 미실행: %v", r.calls)
	}
}

// mountpoint가 없는 데이터셋(예: 자식 없는 최상위 스냅샷 조상 등)에서는 m 키가 무시된다.
func TestMKeyIgnoredWithoutMountpoint(t *testing.T) {
	r := &fakeRunner{}
	a := NewApp(r)
	a.sharesPath = tmpSharesConf(t)
	a.datasets = []zfs.Dataset{{Name: "tank/data", Type: zfs.TypeFilesystem, Mountpoint: ""}}
	a.nodes = []treeNode{{kind: nodeDataset, name: "tank/data"}}
	a.cursor = 0

	var m tea.Model = a
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd != nil || a.modal != nil {
		t.Fatal("mountpoint 없는데 m 키가 동작함")
	}
}
