package zfs

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

func parsePoolStatus(out string) (PoolStatus, error) {
	var st PoolStatus
	inConfig := false
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "pool:"):
			st.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "pool:"))
		case strings.HasPrefix(trimmed, "state:"):
			st.State = strings.TrimSpace(strings.TrimPrefix(trimmed, "state:"))
		case strings.HasPrefix(trimmed, "scan:"):
			st.Scan = strings.TrimSpace(strings.TrimPrefix(trimmed, "scan:"))
		case strings.HasPrefix(trimmed, "errors:"):
			st.Errors = strings.TrimSpace(strings.TrimPrefix(trimmed, "errors:"))
			inConfig = false
		case strings.HasPrefix(trimmed, "config:"):
			inConfig = true
		case inConfig:
			if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
				continue
			}
			body := strings.TrimPrefix(line, "\t")
			depth := (len(body) - len(strings.TrimLeft(body, " "))) / 2
			f := strings.Fields(trimmed)
			if len(f) < 2 {
				continue
			}
			st.VDevs = append(st.VDevs, VDev{Depth: depth, Name: f[0], State: f[1]})
		}
	}
	if st.Name == "" {
		return st, fmt.Errorf("zpool status: pool 이름을 찾지 못함")
	}
	return st, nil
}

func GetPoolStatus(ctx context.Context, r Runner, name string) (PoolStatus, error) {
	out, stderr, err := r.Run(ctx, "zpool", "status", name)
	if err != nil {
		return PoolStatus{}, fmt.Errorf("zpool status: %w: %s", err, stderr)
	}
	return parsePoolStatus(out)
}

func parseDiskinfo(out string) (Disk, error) {
	f := strings.Split(strings.TrimSpace(out), "\t")
	if len(f) < 3 {
		return Disk{}, fmt.Errorf("diskinfo: 예상 밖 출력: %q", out)
	}
	size, err := num(f[2])
	if err != nil {
		return Disk{}, err
	}
	return Disk{Name: f[0], SizeBytes: size}, nil
}

var rootDevRe = regexp.MustCompile(`(?m)^/dev/([a-z]+[0-9]+)p?[0-9]*\s+/\s`)

func rootDiskFromMount(out string) string {
	m := rootDevRe.FindStringSubmatch(out)
	if m == nil {
		return ""
	}
	return m[1]
}

// ListDisks: kern.disks의 각 디스크를 diskinfo로 조사하고,
// OS 디스크와 풀 소속 디스크에 Note를 단다. cd(광학)는 제외.
func ListDisks(ctx context.Context, r Runner, statuses []PoolStatus) ([]Disk, error) {
	out, stderr, err := r.Run(ctx, "sysctl", "-n", "kern.disks")
	if err != nil {
		return nil, fmt.Errorf("kern.disks: %w: %s", err, stderr)
	}
	mountOut, _, err := r.Run(ctx, "mount", "-p")
	if err != nil {
		return nil, err
	}
	rootDisk := rootDiskFromMount(mountOut)

	inPool := map[string]string{}
	for _, st := range statuses {
		for _, v := range st.VDevs {
			if v.Depth >= 1 && !strings.Contains(v.Name, "-") {
				inPool[v.Name] = st.Name
			}
		}
	}

	var disks []Disk
	for _, name := range strings.Fields(strings.TrimSpace(out)) {
		if strings.HasPrefix(name, "cd") {
			continue
		}
		dOut, _, err := r.Run(ctx, "diskinfo", name)
		if err != nil {
			continue
		}
		d, err := parseDiskinfo(dOut)
		if err != nil {
			continue
		}
		if name == rootDisk {
			d.Note = "OS 디스크"
		} else if pool, ok := inPool[name]; ok {
			d.Note = pool + " 소속"
		}
		disks = append(disks, d)
	}
	return disks, nil
}
