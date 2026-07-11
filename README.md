# zen — NanoBSD 기반 홈 NAS OS

맥북에서 빌드해 USB로 굽는 FreeBSD/NanoBSD 커스텀 NAS OS.

- 설계: `docs/superpowers/specs/2026-07-10-zen-nas-os-design.md`
- 계획: `docs/superpowers/plans/2026-07-10-zen-nas-os.md`

## 빠른 시작

```sh
scripts/vm-setup.sh        # 1. 빌드 VM 준비 (최초 1회)
scripts/vm-run.sh          # 2. VM 기동 → docs/vm-bootstrap.md 절차로 SSH 활성화
ssh -p 10022 root@127.0.0.1 sh < scripts/vm-provision.sh   # 3. 빌드 환경 구축
ssh -p 10022 root@127.0.0.1 sh < scripts/vm-fetch-pkgs.sh  # 4. 패키지 확보
scripts/build.sh           # 5. 이미지 빌드 (최초 수 시간)
scripts/test-boot.sh       # 6. QEMU 부팅 테스트
scripts/flash-usb.sh       # 7. USB 굽기
```

## 운영

- **설정 변경 후에는 반드시 `/root/save_cfg`** (PATH에 없음) — 루트는 읽기전용이고 /etc는 tmpfs라서, 안 하면 재부팅 시 증발한다.
- `/cfg`는 매일 03:00에 `tank/backup/cfg`로 자동 백업된다 (풀·데이터셋 존재 시; 실패는 syslog에 기록).
- 패키지는 빌드 시점에 이미지에 구워졌고 `pkg info`로 목록을 확인할 수 있다 (DB는 이미지 내 /pkg/db). 런타임 `pkg add`는 읽기전용 루트라 비영속.
- USB가 죽으면: 새 USB에 굽고 → 부팅 → `zpool import -f tank` → 백업에서 /cfg 복원.
- 실기 배포: `docs/hardware-checklist.md` 참조.

## 아키텍처 요약

| 구성 | 선택 |
|---|---|
| 베이스 | FreeBSD 15.1-RELEASE (NanoBSD embedded, GPT UEFI+BIOS 하이브리드) |
| OS 디스크 | USB — 읽기전용 루트, A/B 슬라이스 업데이트, /cfg 설정 파티션 |
| 데이터 | 별도 디스크 ZFS 풀 (OS와 완전 분리) |
| 서비스 | SMB(fruit)·NFS·rsyncd·WebDAV(lighttpd:8080)·FTP·iSCSI(기본 꺼짐) |
| 관리 | SSH(root, 키 전용) + 설정파일 + /root/save_cfg |
| 빌드 | 맥 → FreeBSD aarch64 VM(QEMU/HVF) → amd64 크로스 빌드 |

## 검증 기록 (2026-07-11, QEMU x86-64)

- BIOS(SeaBIOS)·UEFI(OVMF) 양쪽에서 로그인 프롬프트 도달 (`machdep.bootmethod` 로 확인)
- 6개 서비스 전부 기동: sshd, samba_server(smbd×3), nfsd, rsyncd, lighttpd, ftpd
- `pkg info` 89개 패키지 정상 등록
- ZFS: 풀 생성 → 데이터 기록 → 재부팅 → 자동 import·마운트·데이터 보존 확인
- `/root/save_cfg` 영속화, `zen-cfg-backup` 실행 확인

## 알려진 제약 (v1)

- **A/B 무중단 업데이트 절차는 문서화만 되어 있고 실기 검증 전** — 첫 업데이트는 수동 확인 권장.
- **UEFI 부팅은 슬라이스 A(p4)에 고정** (ESP의 `/efi/freebsd/loader.env`). UEFI에서 B로 전환하려면 이 파일 수정 필요; BIOS 경로는 gptboot bootme로 A/B 전환.
- **WebDAV는 수동 셋업 필요**: `tank/webdav` 데이터셋 + htpasswd 사용자 추가 전까지 401 응답.
- **iSCSI 기본 꺼짐**: ctl.conf 작성 + ctld_enable=YES 필요.
- **패키지 ABI가 FreeBSD 15에 고정** (`zen.conf`) — 베이스 메이저 업그레이드 시 함께 수정.
- **SSH는 키 전용, root 비밀번호 없음** — 키 분실 시 물리 콘솔로 복구.
- **새 빌드 VM은 최초 1회 수동 SSH 개통 필요** (`docs/vm-bootstrap.md`).
- zenfs 데이터셋 생성 압축은 lz4 고정 (선택 UI 없음)

## zenfs — ZFS 관리 TUI

SSH 접속 후 `zenfs` 실행. 좌측 트리(풀→데이터셋→스냅샷)를 `j`/`k`로 탐색하고,
우측 패널의 단축키로 관리한다: `[s]`스냅샷 `[c]`데이터셋 생성 `[P]`풀 생성(마법사)
`[b]`scrub `[D]`파괴 `[R]`롤백 `[x]`디스크 교체. 파괴·롤백·교체는 대상 이름을 그대로
타이핑해야 실행되며, 실행될 CLI 명령 전문이 확인 창에 표시된다. 명령 실패 시 stderr
전문이 모달로 표시된다. 소스는 `tui/`, 빌드는 build.sh가 자동 수행.
