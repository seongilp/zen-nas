package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
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
	return &App{runner: r, collapsed: map[string]bool{}, statuses: map[string]zfs.PoolStatus{}}
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

// contextKey: 선택 항목 종류별 액션 키. Task 8·9에서 구현.
func (a *App) contextKey(k string) tea.Cmd { return nil }

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

// renderLayout: Task 7에서 2패널로 확장. 지금은 트리만.
func renderLayout(a *App) string {
	h := a.height - 2
	return renderTree(a.nodes, a.cursor, h)
}
