package zfs

import (
	"context"
	"testing"
)

type call struct {
	Name string
	Args []string
}

// MockRunner: 호출 기록 + 키("name arg1 arg2...")별 응답 주입.
type MockRunner struct {
	Calls []call
	Out   map[string]string
	Err   map[string]error
}

func (m *MockRunner) Run(_ context.Context, name string, args ...string) (string, string, error) {
	m.Calls = append(m.Calls, call{name, args})
	key := Cmd{name, args}.String()
	return m.Out[key], "", m.Err[key]
}

func TestCmdString(t *testing.T) {
	c := Cmd{"zfs", []string{"destroy", "-r", "tank/data"}}
	if c.String() != "zfs destroy -r tank/data" {
		t.Fatalf("got %q", c.String())
	}
}
