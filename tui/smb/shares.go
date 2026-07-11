// Package smb는 zenfs가 소유하는 /etc/smb-shares.conf를 관리한다.
// 포맷은 zenfs가 쓰는 것만 존재한다고 가정하는 단순 ini 블록.
package smb

import (
	"fmt"
	"os"
	"strings"
)

type Share struct {
	Name     string
	Path     string
	ReadOnly bool
}

func List(confPath string) ([]Share, error) {
	b, err := os.ReadFile(confPath)
	if err != nil {
		return nil, err
	}
	var shares []Share
	var cur *Share
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]"):
			if cur != nil {
				shares = append(shares, *cur)
			}
			cur = &Share{Name: strings.Trim(t, "[]")}
		case cur != nil && strings.HasPrefix(t, "path"):
			cur.Path = strings.TrimSpace(strings.SplitN(t, "=", 2)[1])
		case cur != nil && strings.HasPrefix(t, "read only"):
			cur.ReadOnly = strings.TrimSpace(strings.SplitN(t, "=", 2)[1]) == "yes"
		}
	}
	if cur != nil {
		shares = append(shares, *cur)
	}
	return shares, nil
}

func render(header string, shares []Share) string {
	var b strings.Builder
	b.WriteString(header)
	for _, s := range shares {
		ro := "no"
		if s.ReadOnly {
			ro = "yes"
		}
		fmt.Fprintf(&b, "\n[%s]\n    path = %s\n    read only = %s\n", s.Name, s.Path, ro)
	}
	return b.String()
}

// headerOf: 첫 공유 블록 전까지의 주석/공백을 보존한다.
func headerOf(content string) string {
	if i := strings.Index(content, "["); i >= 0 {
		return strings.TrimRight(content[:i], "\n") + "\n"
	}
	return strings.TrimRight(content, "\n") + "\n"
}

func Add(confPath string, s Share) error {
	b, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}
	shares, err := List(confPath)
	if err != nil {
		return err
	}
	for _, ex := range shares {
		if ex.Name == s.Name {
			return fmt.Errorf("공유 이름 %q 이미 존재", s.Name)
		}
	}
	return os.WriteFile(confPath, []byte(render(headerOf(string(b)), append(shares, s))), 0644)
}

func Remove(confPath string, name string) error {
	b, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}
	shares, err := List(confPath)
	if err != nil {
		return err
	}
	var kept []Share
	for _, s := range shares {
		if s.Name != name {
			kept = append(kept, s)
		}
	}
	return os.WriteFile(confPath, []byte(render(headerOf(string(b)), kept)), 0644)
}

func FindByPath(shares []Share, path string) (Share, bool) {
	for _, s := range shares {
		if s.Path == path {
			return s, true
		}
	}
	return Share{}, false
}
