package ui

import (
	"context"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"zenfs/smb"
	"zenfs/zfs"
)

type refreshMsg struct {
	pools    []zfs.Pool
	datasets []zfs.Dataset
	statuses map[string]zfs.PoolStatus
	disks    []zfs.Disk
	shares   []smb.Share
	err      error
}

type opDoneMsg struct{ err error }

// showModalMsg: 모달 교체 요청. 모달 델리게이션(a.modal, cmd = a.modal.Update(m))이
// 반환값으로 a.modal을 덮어쓰므로, 모달 안에서의 전환은 side effect 대신 메시지로 한다.
type showModalMsg struct{ m tea.Model }

type App struct {
	runner     zfs.Runner
	pools      []zfs.Pool
	datasets   []zfs.Dataset
	statuses   map[string]zfs.PoolStatus
	disks      []zfs.Disk
	sharesPath string      // /etc/smb-shares.conf — zenfs가 소유하는 SMB 공유 정의 파일
	shares     []smb.Share // refresh()마다 sharesPath에서 다시 로드되는 캐시
	nodes      []treeNode
	cursor     int
	collapsed  map[string]bool
	width      int
	height     int
	modal      tea.Model // nil이면 모달 없음
	errText    string    // 마지막 작업/새로고침 오류 — 성공 새로고침 시 지워짐
	warnText   string    // 상시 경고 (예: 비root) — 지워지지 않음
}

func NewApp(r zfs.Runner) *App {
	a := &App{runner: r, collapsed: map[string]bool{}, statuses: map[string]zfs.PoolStatus{},
		sharesPath: "/etc/smb-shares.conf"}
	if os.Geteuid() != 0 {
		a.warnText = "root가 아닙니다 — 변경 작업은 실패합니다"
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
	sharesPath := a.sharesPath
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
		var stList []zfs.PoolStatus
		for _, st := range statuses {
			stList = append(stList, st)
		}
		disks, _ := zfs.ListDisks(ctx, r, stList) // 실패해도 빈 목록으로 진행
		shares, _ := smb.List(sharesPath)         // 실패해도 빈 목록으로 진행 (파일 부재 등)
		return refreshMsg{pools: pools, datasets: datasets, statuses: statuses, disks: disks, shares: shares}
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
		a.pools, a.datasets, a.statuses, a.disks, a.shares = m.pools, m.datasets, m.statuses, m.disks, m.shares
		a.errText = "" // 성공 새로고침 — 스테일 오류 제거 (상시 경고는 warnText에 유지)
		a.nodes = buildTree(a.pools, a.datasets, a.collapsed)
		if a.cursor >= len(a.nodes) {
			a.cursor = max(0, len(a.nodes)-1)
		}
		if a.scrubbing() {
			return a, scrubTick()
		}
	case scrubTickMsg:
		return a, a.refresh()
	case showModalMsg:
		a.modal = m.m
		return a, nil
	case opDoneMsg:
		if m.err != nil {
			// 실패: stderr 전문을 모달로 표시 (모달 자신의 Enter/Esc가 opDoneMsg{}를
			// 다시 보내 성공 경로로 닫힌다 — 루프 없음).
			a.errText = m.err.Error()
			a.modal = &errModal{text: m.err.Error()}
		} else {
			a.modal = nil
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

// cmdSambaReload: SMB 공유 정의(smb-shares.conf) 변경분을 samba에 반영.
func cmdSambaReload() zfs.Cmd {
	return zfs.Cmd{Name: "service", Args: []string{"samba_server", "reload"}}
}

// cmdSaveCfg: NanoBSD /cfg 영속화. save_cfg가 대화형으로 묻는 확인 프롬프트를
// "yes n"으로 자동 응답해 비대화식 실행이 가능하게 한다.
func cmdSaveCfg() zfs.Cmd {
	return zfs.Cmd{Name: "sh", Args: []string{"-c", "yes n | sh /root/save_cfg"}}
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
			recursive := a.hasChildren(n.name)
			c := zfs.CmdDestroyDataset(n.name, recursive)
			m := newConfirm(c, n.name, "데이터셋 파괴", a.execCmd(c))
			if recursive {
				m.warning = "경고: 하위 데이터셋·스냅샷이 함께 삭제됩니다 (-r)."
			}
			a.modal = m
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
	case "s":
		if n != nil && n.kind == nodeDataset {
			ds := n.name
			a.modal = newTextForm("스냅샷 이름 — "+ds+"@", defaultSnapName(),
				func(v string) zfs.Cmd { return zfs.CmdCreateSnapshot(ds, v) }, a.execCmd)
		}
	case "c":
		if n != nil && n.kind != nodeSnapshot {
			base := n.name
			a.modal = newTextForm("새 데이터셋 — "+base+"/", "",
				func(v string) zfs.Cmd { return zfs.CmdCreateDataset(base+"/"+v, "lz4") }, a.execCmd)
		}
	case "P":
		a.modal = newPoolForm(a.disks, func(name, vt string, disks []string) tea.Cmd {
			c := zfs.CmdCreatePool(name, vt, disks)
			confirm := newConfirm(c, name, "풀 생성 — 선택 디스크의 기존 데이터가 사라집니다", a.execCmd(c))
			return func() tea.Msg { return showModalMsg{m: confirm} }
		})
	case "x":
		if n != nil && n.kind == nodePool {
			pool := n.name
			a.modal = newTextForm("교체: <기존디스크> <새디스크> (공백 구분)", "",
				func(v string) zfs.Cmd {
					tok := strings.Fields(v)
					// ★ 두 토큰 필수 + 플래그 주입 방지 (형식 오류는 false 실행으로 표면화)
					if len(tok) != 2 || strings.HasPrefix(tok[0], "-") || strings.HasPrefix(tok[1], "-") {
						return zfs.Cmd{Name: "false"}
					}
					return zfs.CmdReplaceDisk(pool, tok[0], tok[1])
				},
				// 즉시 실행 대신 이름 타이핑 확인 게이트를 거친다 (파괴·롤백과 동일).
				func(c zfs.Cmd) func() tea.Msg {
					if c.Name == "false" { // 형식 오류는 그대로 표면화
						return a.execCmd(c)
					}
					confirm := newConfirm(c, pool, "디스크 교체 — 리실버가 시작됩니다", a.execCmd(c))
					return func() tea.Msg { return showModalMsg{m: confirm} }
				})
		}
	case "m":
		if n == nil || n.kind != nodeDataset {
			return nil
		}
		var d zfs.Dataset
		found := false
		for _, dd := range a.datasets {
			if dd.Name == n.name {
				d, found = dd, true
				break
			}
		}
		if !found || d.Mountpoint == "" {
			return nil
		}
		mp := d.Mountpoint
		if sh, ok := smb.FindByPath(a.shares, mp); ok {
			// 이미 공유 중 — 동기적으로 파일에서 제거 후 reload만 execCmd로 실행.
			if err := smb.Remove(a.sharesPath, sh.Name); err != nil {
				return func() tea.Msg { return opDoneMsg{err: err} }
			}
			a.errText = "공유 해제됨 — [w]로 영속화"
			return a.execCmd(cmdSambaReload())
		}
		// 미공유 — 이름 입력 폼. textForm은 build(v) zfs.Cmd 시그니처만 지원하고
		// smb.Add는 파일 쓰기(부수효과)라 Cmd를 만들지 않으므로, build 클로저 안에서
		// Add를 동기 실행하고 그 성패를 실행할 zfs.Cmd로 인코딩하는 어댑터로 쓴다:
		// 성공 시 reload 명령을, 실패 시 반드시 실패하는 false를 반환해 exec(a.execCmd)의
		// opDoneMsg{err} → errModal 경로로 실패가 표면화되게 한다 (원인은 errText에도 기록).
		a.modal = newTextForm("SMB 공유 이름", shortName(n.name),
			func(v string) zfs.Cmd {
				if err := smb.Add(a.sharesPath, smb.Share{Name: v, Path: mp}); err != nil {
					a.errText = err.Error()
					return zfs.Cmd{Name: "false"}
				}
				a.errText = "공유 추가됨 — [w]로 영속화"
				return cmdSambaReload()
			}, a.execCmd)
	case "w":
		return a.execCmd(cmdSaveCfg())
	}
	return nil
}

// hasChildren: 하위 데이터셋 또는 스냅샷(자신·후손의 것 포함)이 있으면 destroy에 -r이 필요하다.
func (a *App) hasChildren(name string) bool {
	for _, d := range a.datasets {
		if strings.HasPrefix(d.Name, name+"/") || strings.HasPrefix(d.Name, name+"@") {
			return true
		}
	}
	return false
}

// minContentWidth/minContentHeight: 극단적으로 작은 터미널(예: 20x10 미만)에서도
// 트리/디테일 패널 폭·높이가 음수로 내려가 레이아웃이 무너지지 않도록 하는 하한선.
const (
	minContentWidth  = 10
	minContentHeight = 4
)

func (a *App) View() string {
	if a.width == 0 {
		return "로딩 중..."
	}
	if a.modal != nil {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, a.modal.View())
	}
	cw := min(a.width-4, maxContentWidth)
	ch := min(a.height-3, maxContentHeight)
	cw = max(cw, minContentWidth)
	ch = max(ch, minContentHeight)
	content := renderLayout(a, cw, ch) // 내부 콘텐츠를 cw×ch로
	box := appBorder.Render(titleStyle.Render(" zen zfs ") + "\n" + content)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

func renderLayout(a *App, w, h int) string {
	treeW := w * 2 / 5
	detailW := w - treeW - 1
	innerH := max(h-2, 1) // 타이틀·상태바를 제외한 트리/디테일 영역 높이
	tree := lipgloss.NewStyle().Width(treeW).Height(innerH).Render(renderTree(a.nodes, a.cursor, innerH))
	detail := lipgloss.NewStyle().Width(detailW).Height(innerH).PaddingLeft(1).
		BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).Render(renderDetail(a, detailW))
	main := lipgloss.JoinHorizontal(lipgloss.Top, tree, detail)
	return main + "\n" + renderStatusbar(a, w)
}
