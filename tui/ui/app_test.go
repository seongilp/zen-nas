package ui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"zenfs/zfs"
)

func fixtureData() ([]zfs.Pool, []zfs.Dataset) {
	pools := []zfs.Pool{{Name: "tank", Health: "ONLINE"}}
	datasets := []zfs.Dataset{
		{Name: "tank", Type: zfs.TypeFilesystem},
		{Name: "tank/data", Type: zfs.TypeFilesystem, Used: 1024},
		{Name: "tank/data@s1", Type: zfs.TypeSnapshot},
	}
	return pools, datasets
}

func TestBuildTreeOrderAndCollapse(t *testing.T) {
	pools, datasets := fixtureData()
	nodes := buildTree(pools, datasets, map[string]bool{})
	if len(nodes) != 4 { // pool, tank, tank/data, @s1
		t.Fatalf("got %d nodes: %+v", len(nodes), nodes)
	}
	if nodes[0].kind != nodePool || nodes[3].kind != nodeSnapshot {
		t.Fatalf("order wrong: %+v", nodes)
	}
	collapsed := buildTree(pools, datasets, map[string]bool{"tank/data": true})
	if len(collapsed) != 3 {
		t.Fatalf("collapse failed: %+v", collapsed)
	}
}

func TestHuman(t *testing.T) {
	if human(1024) != "1.0K" || human(4294967296) != "4.0G" || human(512) != "512B" {
		t.Fatalf("human: %s %s %s", human(1024), human(4294967296), human(512))
	}
	// %.1f 반올림이 단위 경계(1024.0)를 넘는 경우 다음 단위로 승격
	if human(1073741823) != "1.0G" || human(1048575) != "1.0M" {
		t.Fatalf("human boundary: %s %s", human(1073741823), human(1048575))
	}
}

func TestHasChildrenCountsSnapshots(t *testing.T) {
	a := NewApp(nil)
	a.datasets = []zfs.Dataset{
		{Name: "tank", Type: zfs.TypeFilesystem},
		{Name: "tank/data", Type: zfs.TypeFilesystem},
		{Name: "tank/data@s1", Type: zfs.TypeSnapshot}, // 하위 데이터셋 없이 스냅샷만
	}
	if !a.hasChildren("tank/data") {
		t.Fatal("스냅샷 보유 데이터셋인데 hasChildren=false — destroy에 -r 누락됨")
	}
	if !a.hasChildren("tank") {
		t.Fatal("하위 데이터셋 보유인데 hasChildren=false")
	}
	// 접두사 오탐 방지: "tank/da"는 "tank/data"의 부모가 아니다
	if a.hasChildren("tank/da") {
		t.Fatal("접두사 오탐: tank/da는 자식이 없어야 함")
	}
}

func TestDestroyDatasetRecursiveSetsWarning(t *testing.T) {
	a := NewApp(nil)
	a.datasets = []zfs.Dataset{
		{Name: "tank/data", Type: zfs.TypeFilesystem},
		{Name: "tank/data@s1", Type: zfs.TypeSnapshot},
	}
	a.nodes = []treeNode{{kind: nodeDataset, name: "tank/data"}}
	a.cursor = 0
	a.contextKey("D")
	m, ok := a.modal.(*confirmModel)
	if !ok {
		t.Fatal("D 키인데 확인 모달이 안 열림")
	}
	if !strings.Contains(m.cmd.String(), "-r") {
		t.Fatalf("recursive destroy인데 -r 누락: %s", m.cmd.String())
	}
	if m.warning == "" {
		t.Fatal("recursive destroy인데 경고 문구 없음")
	}
}

func TestStatusbarShowsError(t *testing.T) {
	a := NewApp(nil)
	a.errText = "boom\nsecond line"
	bar := renderStatusbar(a, 80)
	if !strings.Contains(bar, "boom") || strings.Contains(bar, "second line") {
		t.Fatalf("statusbar: %q", bar)
	}
}

// 작업 실패 시 stderr 전문(여러 줄)이 모달로 표시되고, Enter로 닫힌다.
func TestOpErrorOpensErrModal(t *testing.T) {
	a := NewApp(nil)
	a.Update(opDoneMsg{err: errors.New("zpool destroy tank 실패\nstderr 전문 두 번째 줄")})
	em, ok := a.modal.(*errModal)
	if !ok {
		t.Fatalf("실패인데 에러 모달이 안 열림: 모달이 %T", a.modal)
	}
	if !strings.Contains(em.View(), "stderr 전문 두 번째 줄") {
		t.Fatalf("stderr 전문 누락: %q", em.View())
	}
	// Enter로 닫기
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter인데 닫기 cmd 미반환")
	}
	a.Update(cmd())
	if a.modal != nil {
		t.Fatalf("닫은 뒤에도 모달이 남음: %T", a.modal)
	}
}

// 성공 새로고침은 스테일 errText를 지우되, 상시 경고(warnText)는 유지한다.
func TestRefreshSuccessClearsErrTextKeepsWarn(t *testing.T) {
	a := NewApp(nil)
	a.errText = "boom"
	a.Update(refreshMsg{})
	if a.errText != "" {
		t.Fatalf("성공 새로고침 후에도 errText 잔존: %q", a.errText)
	}
	if os.Geteuid() != 0 && a.warnText == "" {
		t.Fatal("비root 경고(warnText)가 없음")
	}
	if os.Geteuid() != 0 && !strings.Contains(renderStatusbar(a, 120), "root가 아닙니다") {
		t.Fatal("상태바에 비root 경고 미표시")
	}
}

// 큰 터미널에서 콘텐츠가 좌상단에 눌어붙지 않고 화면 중앙에 배치되는지 확인.
func TestViewCentersContent(t *testing.T) {
	a := NewApp(nil)
	a.width, a.height = 200, 50
	pools, datasets := fixtureData()
	a.pools, a.datasets = pools, datasets
	a.nodes = buildTree(pools, datasets, a.collapsed)
	view := a.View()
	if !strings.Contains(view, "zen zfs") {
		t.Fatal("타이틀 'zen zfs'가 뷰에 없음")
	}
	if got := lipgloss.Height(view); got != a.height {
		t.Fatalf("뷰 높이가 터미널 높이와 다름: got %d want %d", got, a.height)
	}
	lines := strings.Split(view, "\n")
	if strings.TrimSpace(lines[0]) != "" {
		t.Fatalf("콘텐츠가 중앙 배치되지 않음 — 첫 줄이 비어있지 않음: %q", lines[0])
	}
}

// 아주 작은 터미널(20x10)에서도 패닉 없이 렌더링되고, 콘텐츠 폭이 화면 안에 들어와야 한다.
func TestViewTinyTerminalNoPanic(t *testing.T) {
	sizes := [][2]int{{20, 10}, {10, 5}, {1, 1}}
	for _, sz := range sizes {
		a := NewApp(nil)
		a.width, a.height = sz[0], sz[1]
		pools, datasets := fixtureData()
		a.pools, a.datasets = pools, datasets
		a.nodes = buildTree(pools, datasets, a.collapsed)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%dx%d에서 패닉: %v", sz[0], sz[1], r)
				}
			}()
			_ = a.View()
		}()
	}
}
