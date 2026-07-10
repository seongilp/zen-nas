package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"zenfs/zfs"
)

type nodeKind int

const (
	nodePool nodeKind = iota
	nodeDataset
	nodeSnapshot
)

type treeNode struct {
	kind  nodeKind
	name  string // 전체 이름 (tank, tank/data, tank/data@s1)
	label string // 표시용
	depth int
}

// buildTree: 풀·데이터셋·스냅샷을 표시 순서대로 평탄화.
// collapsed에 든 데이터셋의 스냅샷은 숨긴다.
func buildTree(pools []zfs.Pool, datasets []zfs.Dataset, collapsed map[string]bool) []treeNode {
	var nodes []treeNode
	snapsOf := map[string][]zfs.Dataset{}
	var fsList []zfs.Dataset
	for _, d := range datasets {
		if d.Type == zfs.TypeSnapshot {
			base, _, _ := strings.Cut(d.Name, "@")
			snapsOf[base] = append(snapsOf[base], d)
		} else {
			fsList = append(fsList, d)
		}
	}
	for _, p := range pools {
		nodes = append(nodes, treeNode{nodePool, p.Name,
			fmt.Sprintf("%s (%s)", p.Name, p.Health), 0})
		for _, d := range fsList {
			if d.Name != p.Name && !strings.HasPrefix(d.Name, p.Name+"/") {
				continue
			}
			depth := strings.Count(d.Name, "/") + 1
			nodes = append(nodes, treeNode{nodeDataset, d.Name,
				fmt.Sprintf("%s  %s", shortName(d.Name), human(d.Used)), depth})
			if !collapsed[d.Name] {
				for _, s := range snapsOf[d.Name] {
					_, snap, _ := strings.Cut(s.Name, "@")
					nodes = append(nodes, treeNode{nodeSnapshot, s.Name, "@" + snap, depth + 1})
				}
			}
		}
	}
	return nodes
}

func shortName(full string) string {
	if i := strings.LastIndex(full, "/"); i >= 0 {
		return full[i+1:]
	}
	return full
}

func human(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	v := float64(b) / float64(div)
	if v >= 1023.95 { // %.1f 반올림이 1024.0으로 표기되는 경계면 다음 단위로 승격
		v /= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", v, "KMGTPE"[exp])
}

var selStyle = lipgloss.NewStyle().Reverse(true)

func renderTree(nodes []treeNode, cursor, height int) string {
	var b strings.Builder
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	for i := start; i < len(nodes) && i-start < height; i++ {
		line := strings.Repeat("  ", nodes[i].depth) + nodes[i].label
		if i == cursor {
			line = selStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
