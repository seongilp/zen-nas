package ui

import (
	"fmt"
	"strings"

	"zenfs/zfs"
)

func renderDetail(a *App, width int) string {
	n := a.current()
	if n == nil {
		return "항목 없음 — 풀을 생성하려면 [P]"
	}
	var b strings.Builder
	switch n.kind {
	case nodePool:
		st := a.statuses[n.name]
		var p zfs.Pool
		for _, pp := range a.pools {
			if pp.Name == n.name {
				p = pp
			}
		}
		fmt.Fprintf(&b, "풀 %s  [%s]\n\n", p.Name, p.Health)
		fmt.Fprintf(&b, "크기   %s\n사용   %s\n여유   %s\n\n", human(p.Size), human(p.Alloc), human(p.Free))
		if st.Scan != "" {
			fmt.Fprintf(&b, "scan: %s\n", st.Scan)
		}
		fmt.Fprintf(&b, "errors: %s\n\nvdev:\n", st.Errors)
		for _, v := range st.VDevs {
			fmt.Fprintf(&b, "%s%s  %s\n", strings.Repeat("  ", v.Depth), v.Name, v.State)
		}
		b.WriteString("\n[c]데이터셋 생성 [b]scrub 시작 [B]scrub 중지\n[x]디스크 교체 [D]풀 파괴 [P]새 풀")
	case nodeDataset:
		var d zfs.Dataset
		for _, dd := range a.datasets {
			if dd.Name == n.name {
				d = dd
			}
		}
		snaps := 0
		for _, dd := range a.datasets {
			if dd.Type == zfs.TypeSnapshot && strings.HasPrefix(dd.Name, n.name+"@") {
				snaps++
			}
		}
		fmt.Fprintf(&b, "%s\n\nused        %s\navail       %s\nrefer       %s\n", d.Name, human(d.Used), human(d.Avail), human(d.Refer))
		fmt.Fprintf(&b, "compress    %s\nmountpoint  %s\nsnapshots   %d\n", d.Compression, d.Mountpoint, snaps)
		b.WriteString("\n[s]스냅샷 [c]하위 생성 [D]파괴")
	case nodeSnapshot:
		fmt.Fprintf(&b, "%s\n", n.name)
		b.WriteString("\n[R]롤백 [D]파괴")
	}
	return b.String()
}
