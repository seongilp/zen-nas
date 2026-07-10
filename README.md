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

- **설정 변경 후에는 반드시 `save_cfg`** — 루트는 읽기전용이고 /etc는 tmpfs라서, 안 하면 재부팅 시 증발한다.
- `/cfg`는 매일 03:00에 `tank/backup/cfg`로 자동 백업된다 (풀·데이터셋 존재 시; 실패는 syslog에 기록).
- `pkg info`가 비어 보이는 것은 정상 — 패키지는 빌드 시점에 이미지에 구워졌고 런타임 pkg DB(/var)는 tmpfs다.
- USB가 죽으면: 새 USB에 굽고 → 부팅 → `zpool import -f tank` → 백업에서 /cfg 복원.
- 실기 배포: `docs/hardware-checklist.md` 참조.

## 아키텍처 요약

| 구성 | 선택 |
|---|---|
| 베이스 | FreeBSD 15.1-RELEASE (NanoBSD embedded, GPT UEFI+BIOS 하이브리드) |
| OS 디스크 | USB — 읽기전용 루트, A/B 슬라이스 업데이트, /cfg 설정 파티션 |
| 데이터 | 별도 디스크 ZFS 풀 (OS와 완전 분리) |
| 서비스 | SMB(fruit)·NFS·rsyncd·WebDAV(lighttpd:8080)·FTP·iSCSI(기본 꺼짐) |
| 관리 | SSH(root, 키 전용) + 설정파일 + save_cfg |
| 빌드 | 맥 → FreeBSD aarch64 VM(QEMU/HVF) → amd64 크로스 빌드 |
