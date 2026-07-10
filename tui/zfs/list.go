package zfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// num: -p 출력의 숫자. 스냅샷의 "-"는 0.
func num(s string) (uint64, error) {
	if s == "-" {
		return 0, nil
	}
	return strconv.ParseUint(s, 10, 64)
}

func dash(s string) string {
	if s == "-" {
		return ""
	}
	return s
}

func parsePools(out string) ([]Pool, error) {
	var pools []Pool
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			return nil, fmt.Errorf("zpool list: 예상 밖 행: %q", line)
		}
		size, err := num(f[1])
		if err != nil {
			return nil, err
		}
		alloc, err := num(f[2])
		if err != nil {
			return nil, err
		}
		free, err := num(f[3])
		if err != nil {
			return nil, err
		}
		pools = append(pools, Pool{f[0], size, alloc, free, f[4]})
	}
	return pools, nil
}

func parseDatasets(out string) ([]Dataset, error) {
	var ds []Dataset
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 7 {
			return nil, fmt.Errorf("zfs list: 예상 밖 행: %q", line)
		}
		used, err := num(f[2])
		if err != nil {
			return nil, err
		}
		avail, err := num(f[3])
		if err != nil {
			return nil, err
		}
		refer, err := num(f[4])
		if err != nil {
			return nil, err
		}
		ds = append(ds, Dataset{
			Name: f[0], Type: DatasetType(f[1]),
			Used: used, Avail: avail, Refer: refer,
			Mountpoint: dash(f[5]), Compression: dash(f[6]),
		})
	}
	return ds, nil
}

func ListPools(ctx context.Context, r Runner) ([]Pool, error) {
	out, stderr, err := r.Run(ctx, "zpool", "list", "-Hp", "-o", "name,size,alloc,free,health")
	if err != nil {
		return nil, fmt.Errorf("zpool list: %w: %s", err, stderr)
	}
	return parsePools(out)
}

func ListDatasets(ctx context.Context, r Runner) ([]Dataset, error) {
	out, stderr, err := r.Run(ctx, "zfs", "list", "-Hp",
		"-t", "filesystem,volume,snapshot",
		"-o", "name,type,used,avail,refer,mountpoint,compression")
	if err != nil {
		return nil, fmt.Errorf("zfs list: %w: %s", err, stderr)
	}
	return parseDatasets(out)
}
