#!/usr/bin/env bash
# 빌드 VM을 기동한다. 콘솔은 이 터미널에 붙는다 (종료: VM 안에서 poweroff).
set -euo pipefail

DISK="vm/builder.qcow2"
FW="$(brew --prefix)/share/qemu/edk2-aarch64-code.fd"
CPUS="${ZEN_VM_CPUS:-8}"
MEM="${ZEN_VM_MEM:-16384}"

if [[ ! -f "$DISK" ]]; then
    echo "오류: $DISK 가 없습니다. scripts/vm-setup.sh 를 먼저 실행하세요." >&2
    exit 1
fi
if [[ ! -f "$FW" ]]; then
    echo "오류: UEFI 펌웨어를 찾을 수 없습니다: $FW" >&2
    exit 1
fi

exec qemu-system-aarch64 \
    -M virt -accel hvf -cpu host -smp "$CPUS" -m "$MEM" \
    -drive if=pflash,format=raw,file="$FW",readonly=on \
    -drive if=virtio,format=qcow2,file="$DISK" \
    -netdev user,id=n0,hostfwd=tcp:127.0.0.1:10022-:22 \
    -device virtio-net-pci,netdev=n0 \
    -nographic
