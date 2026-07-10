package zfs

import (
	"context"
	"testing"
)

const poolsFixture = "tank\t4227858432\t147456\t4227710976\tONLINE\n"

func TestParsePools(t *testing.T) {
	pools, err := parsePools(poolsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 1 {
		t.Fatalf("got %d pools", len(pools))
	}
	p := pools[0]
	if p.Name != "tank" || p.Size != 4227858432 || p.Health != "ONLINE" {
		t.Fatalf("bad pool: %+v", p)
	}
}

func TestParsePoolsEmpty(t *testing.T) {
	pools, err := parsePools("")
	if err != nil || len(pools) != 0 {
		t.Fatalf("empty input: pools=%v err=%v", pools, err)
	}
}

const datasetsFixture = "tank\tfilesystem\t147456\t4093475840\t25088\t/tank\ton\n" +
	"tank/data\tfilesystem\t25088\t4093475840\t25088\t/tank/data\ton\n" +
	"tank/data@s1\tsnapshot\t13312\t-\t25088\t-\t-\n"

func TestParseDatasets(t *testing.T) {
	ds, err := parseDatasets(datasetsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 3 {
		t.Fatalf("got %d datasets", len(ds))
	}
	if ds[2].Type != TypeSnapshot || ds[2].Avail != 0 || ds[2].Name != "tank/data@s1" {
		t.Fatalf("bad snapshot row: %+v", ds[2])
	}
	if ds[2].Mountpoint != "" || ds[2].Compression != "" {
		t.Fatalf("dash fields not normalized: %+v", ds[2])
	}
	if ds[1].Mountpoint != "/tank/data" {
		t.Fatalf("bad mountpoint: %+v", ds[1])
	}
}

func TestParseDatasetsBadRow(t *testing.T) {
	if _, err := parseDatasets("tank\tonly-two\n"); err == nil {
		t.Fatal("expected error for malformed row")
	}
}

func TestListPoolsUsesCorrectCommand(t *testing.T) {
	m := &MockRunner{Out: map[string]string{
		"zpool list -Hp -o name,size,alloc,free,health": poolsFixture,
	}}
	if _, err := ListPools(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	want := "zpool list -Hp -o name,size,alloc,free,health"
	if got := (Cmd{m.Calls[0].Name, m.Calls[0].Args}).String(); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestListDatasetsUsesCorrectCommand(t *testing.T) {
	m := &MockRunner{Out: map[string]string{
		"zfs list -Hp -t filesystem,volume,snapshot -o name,type,used,avail,refer,mountpoint,compression": datasetsFixture,
	}}
	if _, err := ListDatasets(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	want := "zfs list -Hp -t filesystem,volume,snapshot -o name,type,used,avail,refer,mountpoint,compression"
	if got := (Cmd{m.Calls[0].Name, m.Calls[0].Args}).String(); got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}
