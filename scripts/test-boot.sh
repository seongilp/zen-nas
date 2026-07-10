#!/usr/bin/env bash
# 맥에서 실행: 빌드된 이미지를 QEMU x86-64(TCG 에뮬레이션)로 부팅한다.
# 사용법: scripts/test-boot.sh          # BIOS(SeaBIOS) 부팅
#         scripts/test-boot.sh --uefi   # UEFI(OVMF) 부팅
# 로그인 프롬프트가 뜨면 성공. 다른 터미널에서 ssh -p 10122 root@127.0.0.1 로 접속 확인.
# 종료: Ctrl-a x
set -euo pipefail
cd "$(dirname "$0")/.."

IMG="build/zen.img"
if [[ ! -f "$IMG" ]]; then
    echo "오류: $IMG 가 없습니다. scripts/build.sh 를 먼저 실행하세요." >&2
    exit 1
fi

EXTRA=()
if [[ "${1:-}" == "--uefi" ]]; then
    FW="$(brew --prefix)/share/qemu/edk2-x86_64-code.fd"
    if [[ ! -f "$FW" ]]; then
        echo "오류: UEFI 펌웨어를 찾을 수 없습니다: $FW" >&2
        exit 1
    fi
    EXTRA=(-drive "if=pflash,format=raw,file=$FW,readonly=on")
fi

exec qemu-system-x86_64 \
    -m 2048 -smp 2 \
    -drive "file=$IMG,format=raw" \
    -netdev user,id=n0,hostfwd=tcp:127.0.0.1:10122-:22 \
    -device virtio-net-pci,netdev=n0 \
    ${EXTRA[@]+"${EXTRA[@]}"} \
    -nographic
