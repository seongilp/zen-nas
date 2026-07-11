package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"zenfs/zfs"
)

// textForm: 이름 하나 입력받아 build(값)로 Cmd를 만들어 실행.
type textForm struct {
	title string
	input textinput.Model
	build func(string) zfs.Cmd
	exec  func(zfs.Cmd) func() tea.Msg
}

func newTextForm(title, initial string, build func(string) zfs.Cmd, exec func(zfs.Cmd) func() tea.Msg) *textForm {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.Focus()
	return &textForm{title: title, input: ti, build: build, exec: exec}
}

func (f *textForm) Init() tea.Cmd { return nil }

func (f *textForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.Type {
		case tea.KeyEsc:
			return f, func() tea.Msg { return opDoneMsg{} }
		case tea.KeyEnter:
			v := strings.TrimSpace(f.input.Value())
			if v == "" || strings.HasPrefix(v, "-") { // ★ 플래그 주입 방지
				return f, nil
			}
			return f, f.exec(f.build(v))
		}
	}
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

func (f *textForm) View() string {
	return modalStyle.Render(fmt.Sprintf("%s\n\n%s\n\n(Enter 실행 / Esc 취소)", f.title, f.input.View()))
}

// poolForm: 이름 → vdev 타입 → 디스크 다중 선택 3단계 마법사.
type poolForm struct {
	step     int // 0=이름, 1=타입, 2=디스크
	name     textinput.Model
	vdevIdx  int
	disks    []zfs.Disk
	selected map[int]bool
	finish   func(name, vdevType string, disks []string) tea.Cmd
	cursor   int
}

var vdevTypes = []string{"single", "mirror", "raidz1"}

func newPoolForm(disks []zfs.Disk, finish func(string, string, []string) tea.Cmd) *poolForm {
	ti := textinput.New()
	ti.Focus()
	return &poolForm{name: ti, disks: disks, selected: map[int]bool{}, finish: finish}
}

func (f *poolForm) Init() tea.Cmd { return nil }

func (f *poolForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}
	if k.Type == tea.KeyEsc {
		return f, func() tea.Msg { return opDoneMsg{} }
	}
	switch f.step {
	case 0:
		if k.Type == tea.KeyEnter {
			v := strings.TrimSpace(f.name.Value())
			if v != "" && !strings.HasPrefix(v, "-") { // ★
				f.step = 1
			}
			return f, nil
		}
		var cmd tea.Cmd
		f.name, cmd = f.name.Update(msg)
		return f, cmd
	case 1:
		switch k.String() {
		case "up", "k":
			if f.vdevIdx > 0 {
				f.vdevIdx--
			}
		case "down", "j":
			if f.vdevIdx < len(vdevTypes)-1 {
				f.vdevIdx++
			}
		case "enter":
			f.step = 2
		}
	case 2:
		switch k.String() {
		case "up", "k":
			if f.cursor > 0 {
				f.cursor--
			}
		case "down", "j":
			if f.cursor < len(f.disks)-1 {
				f.cursor++
			}
		case " ":
			if len(f.disks) > 0 && f.disks[f.cursor].Note == "" {
				f.selected[f.cursor] = !f.selected[f.cursor]
			}
		case "enter":
			var names []string
			for i, on := range f.selected {
				if on {
					names = append(names, f.disks[i].Name)
				}
			}
			if len(names) == 0 {
				return f, nil
			}
			return f, f.finish(strings.TrimSpace(f.name.Value()), vdevTypes[f.vdevIdx], names)
		}
	}
	return f, nil
}

func (f *poolForm) View() string {
	var b strings.Builder
	b.WriteString("새 ZFS 풀 생성\n\n")
	switch f.step {
	case 0:
		fmt.Fprintf(&b, "풀 이름: %s\n(Enter 다음 / Esc 취소)", f.name.View())
	case 1:
		b.WriteString("구성 선택:\n")
		for i, v := range vdevTypes {
			cur := "  "
			if i == f.vdevIdx {
				cur = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", cur, v)
		}
		b.WriteString("(Enter 다음 / Esc 취소)")
	case 2:
		b.WriteString("디스크 선택 (Space 토글, Enter 완료):\n")
		for i, d := range f.disks {
			cur := "  "
			if i == f.cursor {
				cur = "> "
			}
			mark := "[ ]"
			if f.selected[i] {
				mark = "[x]"
			}
			note := ""
			if d.Note != "" {
				note = "  ← " + d.Note + " (선택 불가)"
			}
			fmt.Fprintf(&b, "%s%s %s  %s%s\n", cur, mark, d.Name, human(d.SizeBytes), note)
		}
	}
	return modalStyle.Render(b.String())
}

func defaultSnapName() string {
	return time.Now().Format("manual-20060102-1504")
}
