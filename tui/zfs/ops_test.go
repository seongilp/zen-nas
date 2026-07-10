package zfs

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCmdBuilders(t *testing.T) {
	cases := []struct {
		got  Cmd
		want Cmd
	}{
		{CmdCreateDataset("tank/data", "lz4"), Cmd{"zfs", []string{"create", "-o", "compression=lz4", "tank/data"}}},
		{CmdCreateDataset("tank/data", ""), Cmd{"zfs", []string{"create", "tank/data"}}},
		{CmdDestroyDataset("tank/data", false), Cmd{"zfs", []string{"destroy", "tank/data"}}},
		{CmdDestroyDataset("tank/data", true), Cmd{"zfs", []string{"destroy", "-r", "tank/data"}}},
		{CmdCreateSnapshot("tank/data", "manual-1"), Cmd{"zfs", []string{"snapshot", "tank/data@manual-1"}}},
		{CmdRollback("tank/data@s1"), Cmd{"zfs", []string{"rollback", "-r", "tank/data@s1"}}},
		{CmdDestroySnapshot("tank/data@s1"), Cmd{"zfs", []string{"destroy", "tank/data@s1"}}},
		{CmdCreatePool("tank", "single", []string{"ada1"}), Cmd{"zpool", []string{"create", "tank", "ada1"}}},
		{CmdCreatePool("tank", "mirror", []string{"ada1", "ada2"}), Cmd{"zpool", []string{"create", "tank", "mirror", "ada1", "ada2"}}},
		{CmdCreatePool("tank", "raidz1", []string{"ada1", "ada2", "ada3"}), Cmd{"zpool", []string{"create", "tank", "raidz1", "ada1", "ada2", "ada3"}}},
		{CmdDestroyPool("tank"), Cmd{"zpool", []string{"destroy", "tank"}}},
		{CmdScrubStart("tank"), Cmd{"zpool", []string{"scrub", "tank"}}},
		{CmdScrubStop("tank"), Cmd{"zpool", []string{"scrub", "-s", "tank"}}},
		{CmdReplaceDisk("tank", "ada1", "ada3"), Cmd{"zpool", []string{"replace", "tank", "ada1", "ada3"}}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("got %q want %q", c.got, c.want)
		}
	}
}

func TestExecuteRunsCommand(t *testing.T) {
	m := &MockRunner{Out: map[string]string{}}
	if err := Execute(context.Background(), m, CmdScrubStart("tank")); err != nil {
		t.Fatal(err)
	}
	if len(m.Calls) != 1 || m.Calls[0].Name != "zpool" {
		t.Fatalf("calls = %+v", m.Calls)
	}
}

func TestExecuteReturnsErrorOnFailure(t *testing.T) {
	m := &MockRunner{
		Out: map[string]string{},
		Err: map[string]error{"zpool destroy tank": errors.New("exit 1")},
	}
	err := Execute(context.Background(), m, CmdDestroyPool("tank"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "zpool destroy tank") {
		t.Fatalf("error should contain the command: %v", err)
	}
}
