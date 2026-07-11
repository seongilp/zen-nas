# zenfs SMB 공유 관리 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** zenfs에서 `m` 키로 데이터셋 SMB 공유 토글, `w` 키로 /cfg 영속화.

**Architecture:** 이미지의 smb4.conf가 `/etc/smb-shares.conf`(tmpfs, save_cfg 영속화 대상)를 include. `tui/smb/` 패키지가 그 파일을 소유·관리하고, samba reload와 save_cfg는 Runner로 실행. UI는 데이터셋 상세에 공유 상태 표시 + m/w 키.

**Tech Stack:** 기존 zenfs 스택 (Go/bubbletea), Samba include 지시자, NanoBSD save_cfg.

**스펙:** docs/superpowers/specs/2026-07-11-zenfs-smb-shares-design.md
**브랜치:** impl/zenfs 이어서 사용.

---

### Task 1: 이미지 오버레이 — include + 빈 shares 파일

**Files:** Modify `nanobsd/overlay/usr/local/etc/smb4.conf`; Create `nanobsd/overlay/etc/smb-shares.conf`

- [ ] smb4.conf [global] 마지막에 추가:
```ini
    # zenfs가 관리하는 공유 정의 (tmpfs /etc — /root/save_cfg로 영속화)
    include = /etc/smb-shares.conf
```
- [ ] `nanobsd/overlay/etc/smb-shares.conf` 생성:
```
# zenfs가 관리하는 SMB 공유 정의 — 직접 편집하지 말 것 (zenfs [m] 키 사용)
```
- [ ] 검증: 두 파일 존재·문법. Commit: `feat: smb4.conf include + zenfs 공유 정의 파일`

### Task 2: tui/smb 패키지 (TDD)

**Files:** Create `tui/smb/shares.go`, `tui/smb/shares_test.go`

- [ ] RED: 테스트 먼저 —
```go
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
```
- [ ] GREEN: `shares.go` —
```go
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
```
- [ ] gofmt/vet/test. Commit: `feat: zenfs smb 공유 정의 파일 관리 패키지`

### Task 3: reload·save_cfg 명령 + UI 연결 (TDD)

**Files:** Modify `tui/ui/app.go`, `tui/ui/detail.go`; test `tui/ui/app_test.go` 또는 새 파일

- [ ] App에 `sharesPath string` 필드 (NewApp 기본 `/etc/smb-shares.conf`), `shares []smb.Share` 캐시 — refresh()에서 `smb.List(a.sharesPath)` (에러 무시, 빈 목록).
- [ ] zfs.Cmd 재사용: reload = `Cmd{"service", []string{"samba_server", "reload"}}`, save = `Cmd{"sh", []string{"-c", "yes n | sh /root/save_cfg"}}` — app.go 안에 로컬 헬퍼로 정의.
- [ ] contextKey `m` (nodeDataset, Mountpoint != ""): 공유 중(FindByPath) → `smb.Remove` + reload 실행 + errText 안내 "[w]로 영속화"; 미공유 → newTextForm("SMB 공유 이름", shortName(ds)) → Add + reload. Add/Remove는 파일 작업이라 동기 실행 후 reload만 execCmd 경유. 에러는 errModal 흐름 재사용 (opDoneMsg{err}).
- [ ] contextKey `w` (전역): execCmd(save) — 성공 시 상태바 정상 갱신.
- [ ] detail.go 데이터셋 뷰에 `smb 공유   <이름|없음>` 행 + 힌트에 `[m]SMB [w]저장` 추가. 상태바 우측 힌트에 `[w]저장` 추가.
- [ ] 테스트: 임시 conf 경로 주입한 App으로 — m 키 → textForm → 이름 Enter → 파일에 블록 존재 + reload Cmd가 fakeRunner에 기록; 다시 m → 블록 제거. (fakeRunner는 forms_test.go에 이미 있음)
- [ ] 전체 -race/gofmt/vet/크로스빌드. Commit: `feat: zenfs SMB 공유 토글·영속화 키`

### Task 4: E2E (QEMU) + 문서 + 머지

- [ ] 빌드 VM 기동(`scripts/vm-run.sh -d`) → `scripts/build.sh -b` (kill 시 원격 폴백 회수 패턴).
- [ ] QEMU 부팅(스크래치 디스크 2) → CLI로 `zpool create tank mirror ada1 ada2 && zfs create tank/data`.
- [ ] expect로 zenfs 구동: tank/data로 이동 → `m` → Enter(기본 이름) → 종료. 게스트에서 `cat /etc/smb-shares.conf`에 [data] 블록, `testparm -s 2>/dev/null | grep -A2 "\[data\]"`로 samba 인식 확인.
- [ ] `w` 동작: expect로 `w` 전송 후 게스트에서 `mount /cfg && cat /cfg/smb-shares.conf && umount /cfg` — 블록 존재.
- [ ] 맥 Finder 스모크는 실기 과제로 문서화. README zenfs 섹션에 `[m]`·`[w]` 안내 추가. 체크리스트의 수동 smb4.conf 단계를 "zenfs [m] 사용(권장) 또는 수동" 으로 갱신.
- [ ] Commit: `docs: zenfs SMB 공유 사용 안내` → main 머지.

## 리스크

| 리스크 | 대응 |
|---|---|
| samba include가 reload로 반영 안 됨 | testparm으로 검증; 필요시 restart로 전환 |
| save_cfg가 /etc/smb-shares.conf를 신규 파일로 취급 | 이미지 base에 포함되므로 수정 파일 — E2E에서 실측 확인 |
| smbd가 include 파일 문법 오류에 민감 | 파일은 zenfs 단독 소유 + 라운드트립 테스트 |
