package ui

import (
	"context"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"zenfs/zfs"
)

type refreshMsg struct {
	pools    []zfs.Pool
	datasets []zfs.Dataset
	statuses map[string]zfs.PoolStatus
	err      error
}

type opDoneMsg struct{ err error }

type App struct {
	runner    zfs.Runner
	pools     []zfs.Pool
	datasets  []zfs.Dataset
	statuses  map[string]zfs.PoolStatus
	nodes     []treeNode
	cursor    int
	collapsed map[string]bool
	width     int
	height    int
	modal     tea.Model // nil이면 모달 없음
	errText   string
}

func NewApp(r zfs.Runner) *App {
	a := &App{runner: r, collapsed: map[string]bool{}, statuses: map[string]zfs.PoolStatus{}}
	if os.Geteuid() != 0 {
		a.errText = "root가 아닙니다 — 변경 작업은 실패합니다"
	}
	return a
}

type scrubTickMsg struct{}

func scrubTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return scrubTickMsg{} })
}

func (a *App) scrubbing() bool {
	for _, st := range a.statuses {
		if strings.Contains(st.Scan, "in progress") {
			return true
		}
	}
	return false
}

func (a *App) Init() tea.Cmd { return a.refresh() }

func (a *App) refresh() tea.Cmd {
	r := a.runner
	return func() tea.Msg {
		ctx := context.Background()
		pools, err := zfs.ListPools(ctx, r)
		if err != nil {
			return refreshMsg{err: err}
		}
		datasets, err := zfs.ListDatasets(ctx, r)
		if err != nil {
			return refreshMsg{err: err}
		}
		statuses := map[string]zfs.PoolStatus{}
		for _, p := range pools {
			if st, err := zfs.GetPoolStatus(ctx, r, p.Name); err == nil {
				statuses[p.Name] = st
			}
		}
		return refreshMsg{pools: pools, datasets: datasets, statuses: statuses}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
	case refreshMsg:
		if m.err != nil {
			a.errText = m.err.Error()
			return a, nil
		}
		a.pools, a.datasets, a.statuses = m.pools, m.datasets, m.statuses
		a.nodes = buildTree(a.pools, a.datasets, a.collapsed)
		if a.cursor >= len(a.nodes) {
			a.cursor = max(0, len(a.nodes)-1)
		}
		if a.scrubbing() {
			return a, scrubTick()
		}
	case scrubTickMsg:
		return a, a.refresh()
	case opDoneMsg:
		a.modal = nil
		if m.err != nil {
			a.errText = m.err.Error()
		}
		return a, a.refresh()
	case tea.KeyMsg:
		if a.modal != nil {
			var cmd tea.Cmd
			a.modal, cmd = a.modal.Update(m)
			return a, cmd
		}
		switch m.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "r":
			return a, a.refresh()
		case "up", "k":
			if a.cursor > 0 {
				a.cursor--
			}
		case "down", "j":
			if a.cursor < len(a.nodes)-1 {
				a.cursor++
			}
		case "enter", " ":
			if n := a.current(); n != nil && n.kind == nodeDataset {
				a.collapsed[n.name] = !a.collapsed[n.name]
				a.nodes = buildTree(a.pools, a.datasets, a.collapsed)
			}
		default:
			return a, a.contextKey(m.String())
		}
	}
	return a, nil
}

func (a *App) current() *treeNode {
	if a.cursor < 0 || a.cursor >= len(a.nodes) {
		return nil
	}
	return &a.nodes[a.cursor]
}

func (a *App) execCmd(c zfs.Cmd) func() tea.Msg {
	r := a.runner
	return func() tea.Msg {
		return opDoneMsg{err: zfs.Execute(context.Background(), r, c)}
	}
}

// contextKey: 선택 항목 종류별 액션 키.
func (a *App) contextKey(k string) tea.Cmd {
	n := a.current()
	switch k {
	case "b":
		if n != nil && n.kind == nodePool {
			return a.execCmd(zfs.CmdScrubStart(n.name)) // 비파괴 — 확인 없이 즉시
		}
	case "B":
		if n != nil && n.kind == nodePool {
			return a.execCmd(zfs.CmdScrubStop(n.name))
		}
	case "D":
		if n == nil {
			return nil
		}
		switch n.kind {
		case nodePool:
			c := zfs.CmdDestroyPool(n.name)
			a.modal = newConfirm(c, n.name, "풀 파괴 — 모든 데이터가 사라집니다", a.execCmd(c))
		case nodeDataset:
			c := zfs.CmdDestroyDataset(n.name, a.hasChildren(n.name))
			a.modal = newConfirm(c, n.name, "데이터셋 파괴", a.execCmd(c))
		case nodeSnapshot:
			c := zfs.CmdDestroySnapshot(n.name)
			a.modal = newConfirm(c, n.name, "스냅샷 파괴", a.execCmd(c))
		}
	case "R":
		if n != nil && n.kind == nodeSnapshot {
			c := zfs.CmdRollback(n.name)
			m := newConfirm(c, n.name, "스냅샷 롤백", a.execCmd(c))
			m.warning = "경고: 이 스냅샷 이후에 생성된 스냅샷·데이터가 모두 삭제됩니다 (-r)."
			a.modal = m
		}
	}
	return nil
}

func (a *App) hasChildren(name string) bool {
	for _, d := range a.datasets {
		if d.Type != zfs.TypeSnapshot && strings.HasPrefix(d.Name, name+"/") {
			return true
		}
	}
	return false
}

func (a *App) View() string {
	if a.width == 0 {
		return "로딩 중..."
	}
	body := renderLayout(a)
	if a.modal != nil {
		return body + "\n" + a.modal.View()
	}
	return body
}

func renderLayout(a *App) string {
	treeW := a.width * 2 / 5
	detailW := a.width - treeW - 1
	h := a.height - 2
	tree := lipgloss.NewStyle().Width(treeW).Height(h).Render(renderTree(a.nodes, a.cursor, h))
	detail := lipgloss.NewStyle().Width(detailW).Height(h).PaddingLeft(1).
		BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).Render(renderDetail(a, detailW))
	main := lipgloss.JoinHorizontal(lipgloss.Top, tree, detail)
	return main + "\n" + renderStatusbar(a, a.width)
}
