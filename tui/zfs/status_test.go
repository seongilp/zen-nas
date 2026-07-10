package zfs

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

const statusFixture = `  pool: tank
 state: ONLINE
  scan: scrub repaired 0B in 00:00:01 with 0 errors on Fri Jul 11 01:00:00 2026
config:

	NAME        STATE     READ WRITE CKSUM
	tank        ONLINE       0     0     0
	  mirror-0  ONLINE       0     0     0
	    ada1    ONLINE       0     0     0
	    ada2    ONLINE       0     0     0

errors: No known data errors
`

func TestParsePoolStatus(t *testing.T) {
	st, err := parsePoolStatus(statusFixture)
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "tank" || st.State != "ONLINE" {
		t.Fatalf("bad status: %+v", st)
	}
	if !strings.Contains(st.Scan, "scrub repaired") {
		t.Fatalf("scan not captured: %q", st.Scan)
	}
	want := []VDev{{0, "tank", "ONLINE"}, {1, "mirror-0", "ONLINE"}, {2, "ada1", "ONLINE"}, {2, "ada2", "ONLINE"}}
	if !reflect.DeepEqual(st.VDevs, want) {
		t.Fatalf("vdevs = %+v", st.VDevs)
	}
	if st.Errors != "No known data errors" {
		t.Fatalf("errors = %q", st.Errors)
	}
}

func TestParsePoolStatusNoName(t *testing.T) {
	if _, err := parsePoolStatus("garbage\n"); err == nil {
		t.Fatal("expected error when pool name missing")
	}
}

func TestParseDiskinfo(t *testing.T) {
	d, err := parseDiskinfo("ada1\t512\t4294967296\t8388608\t0\t0\n")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "ada1" || d.SizeBytes != 4294967296 {
		t.Fatalf("bad disk: %+v", d)
	}
}

func TestRootDiskFromMount(t *testing.T) {
	if got := rootDiskFromMount("/dev/ada0p4 / ufs ro 1 1\ndevfs /dev devfs rw 0 0\n"); got != "ada0" {
		t.Fatalf("root disk = %q", got)
	}
	if got := rootDiskFromMount("tank/root / zfs rw 0 0\n"); got != "" {
		t.Fatalf("non-/dev root should give empty, got %q", got)
	}
}

func TestListDisksAnnotates(t *testing.T) {
	m := &MockRunner{Out: map[string]string{
		"sysctl -n kern.disks": "cd0 ada2 ada1 ada0\n",
		"mount -p":             "/dev/ada0p4 / ufs ro 1 1\n",
		"diskinfo ada2":        "ada2\t512\t2147483648\t4194304\t0\t0\n",
		"diskinfo ada1":        "ada1\t512\t4294967296\t8388608\t0\t0\n",
		"diskinfo ada0":        "ada0\t512\t1450387456\t2832788\t0\t0\n",
	}}
	statuses := []PoolStatus{{Name: "tank", VDevs: []VDev{{0, "tank", "ONLINE"}, {1, "ada1", "ONLINE"}}}}
	disks, err := ListDisks(context.Background(), m, statuses)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Disk{}
	for _, d := range disks {
		byName[d.Name] = d
	}
	if _, hasCD := byName["cd0"]; hasCD {
		t.Fatal("cd0 should be excluded")
	}
	if byName["ada0"].Note != "OS 디스크" {
		t.Fatalf("ada0 note = %q", byName["ada0"].Note)
	}
	if byName["ada1"].Note != "tank 소속" {
		t.Fatalf("ada1 note = %q", byName["ada1"].Note)
	}
	if byName["ada2"].Note != "" {
		t.Fatalf("ada2 note = %q", byName["ada2"].Note)
	}
}
