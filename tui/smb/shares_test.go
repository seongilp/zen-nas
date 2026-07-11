package smb

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpConf(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "smb-shares.conf")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

const header = "# zenfs managed\n"

func TestListEmpty(t *testing.T) {
	p := tmpConf(t, header)
	shares, err := List(p)
	if err != nil || len(shares) != 0 {
		t.Fatalf("shares=%v err=%v", shares, err)
	}
}

func TestAddListRemoveRoundtrip(t *testing.T) {
	p := tmpConf(t, header)
	if err := Add(p, Share{Name: "data", Path: "/tank/data", ReadOnly: false}); err != nil {
		t.Fatal(err)
	}
	shares, err := List(p)
	if err != nil || len(shares) != 1 {
		t.Fatalf("shares=%v err=%v", shares, err)
	}
	if shares[0].Name != "data" || shares[0].Path != "/tank/data" || shares[0].ReadOnly {
		t.Fatalf("share=%+v", shares[0])
	}
	if err := Remove(p, "data"); err != nil {
		t.Fatal(err)
	}
	shares, _ = List(p)
	if len(shares) != 0 {
		t.Fatalf("remove 실패: %v", shares)
	}
	// 헤더 주석은 보존
	b, _ := os.ReadFile(p)
	if string(b[:len(header)]) != header {
		t.Fatalf("헤더 소실: %q", b)
	}
}

func TestAddDuplicateRejected(t *testing.T) {
	p := tmpConf(t, header)
	_ = Add(p, Share{Name: "data", Path: "/tank/data"})
	if err := Add(p, Share{Name: "data", Path: "/tank/other"}); err == nil {
		t.Fatal("중복 이름인데 성공")
	}
}

func TestFindByPath(t *testing.T) {
	p := tmpConf(t, header)
	_ = Add(p, Share{Name: "data", Path: "/tank/data"})
	s, ok := FindByPath(mustList(t, p), "/tank/data")
	if !ok || s.Name != "data" {
		t.Fatalf("s=%+v ok=%v", s, ok)
	}
	if _, ok := FindByPath(mustList(t, p), "/tank/x"); ok {
		t.Fatal("없는 경로가 검색됨")
	}
}

func mustList(t *testing.T, p string) []Share {
	t.Helper()
	shares, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	return shares
}
