package zfs

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// Runner는 외부 명령 실행을 추상화한다. 테스트에서 mock으로 대체.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr string, err error)
}

type ExecRunner struct {
	Timeout time.Duration // 0이면 30초
}

func (r ExecRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	t := r.Timeout
	if t == 0 {
		t = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}
