package ui

import (
	"strings"
	"testing"

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

func TestStatusbarShowsError(t *testing.T) {
	a := NewApp(nil)
	a.errText = "boom\nsecond line"
	bar := renderStatusbar(a, 80)
	if !strings.Contains(bar, "boom") || strings.Contains(bar, "second line") {
		t.Fatalf("statusbar: %q", bar)
	}
}
