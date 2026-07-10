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
