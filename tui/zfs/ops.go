package zfs

import (
	"context"
	"fmt"
)

// CmdCreateDataset: compression이 ""이면 -o compression= 옵션을 생략한다.
func CmdCreateDataset(name, compression string) Cmd {
	args := []string{"create"}
	if compression != "" {
		args = append(args, "-o", "compression="+compression)
	}
	return Cmd{"zfs", append(args, name)}
}

// CmdDestroyDataset: recursive면 -r을 붙인다 (하위 데이터셋/스냅샷까지 삭제되므로 UI 경고 필수).
func CmdDestroyDataset(name string, recursive bool) Cmd {
	args := []string{"destroy"}
	if recursive {
		args = append(args, "-r")
	}
	return Cmd{"zfs", append(args, name)}
}

func CmdCreateSnapshot(dataset, snap string) Cmd {
	return Cmd{"zfs", []string{"snapshot", dataset + "@" + snap}}
}

// CmdRollback: -r 필수 — 대상 이후 스냅샷 삭제를 동반한다 (UI 경고 필수).
func CmdRollback(fullSnap string) Cmd {
	return Cmd{"zfs", []string{"rollback", "-r", fullSnap}}
}

func CmdDestroySnapshot(fullSnap string) Cmd {
	return Cmd{"zfs", []string{"destroy", fullSnap}}
}

// CmdCreatePool: vdevType은 "single" | "mirror" | "raidz1". single은 vdev 타입 인자를 생략한다.
func CmdCreatePool(name, vdevType string, disks []string) Cmd {
	args := []string{"create", name}
	if vdevType != "single" {
		args = append(args, vdevType)
	}
	return Cmd{"zpool", append(args, disks...)}
}

func CmdDestroyPool(name string) Cmd { return Cmd{"zpool", []string{"destroy", name}} }

func CmdScrubStart(name string) Cmd { return Cmd{"zpool", []string{"scrub", name}} }

func CmdScrubStop(name string) Cmd { return Cmd{"zpool", []string{"scrub", "-s", name}} }

func CmdReplaceDisk(pool, old, new string) Cmd {
	return Cmd{"zpool", []string{"replace", pool, old, new}}
}

// Execute: Cmd를 Runner로 실행하고, 실패 시 명령 전문·원인·stderr를 포함한 에러를 반환한다.
func Execute(ctx context.Context, r Runner, c Cmd) error {
	_, stderr, err := r.Run(ctx, c.Name, c.Args...)
	if err != nil {
		return fmt.Errorf("%s 실패: %w\n%s", c.String(), err, stderr)
	}
	return nil
}
