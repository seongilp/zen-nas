#!/usr/bin/env bash
# 빌드 VM을 기동한다. 콘솔은 이 터미널에 붙는다 (종료: VM 안에서 poweroff).
# -d: 데몬 모드로 기동한다 (콘솔은 유닉스 소켓으로, 터미널은 즉시 반환).
set -euo pipefail

cd "$(dirname "$0")/.."

DISK="vm/builder.qcow2"
FW="$(brew --prefix)/share/qemu/edk2-aarch64-code.fd"
CPUS="${ZEN_VM_CPUS:-8}"
MEM="${ZEN_VM_MEM:-16384}"
PIDFILE="vm/qemu.pid"
CONSOLE_SOCK="vm/console.sock"
DAEMON=0

while getopts "d" opt; do
    case "$opt" in
        d) DAEMON=1 ;;
        *)
            echo "사용법: $0 [-d]" >&2
            exit 1
            ;;
    esac
done

if [[ ! -f "$DISK" ]]; then
    echo "오류: $DISK 가 없습니다. scripts/vm-setup.sh 를 먼저 실행하세요." >&2
    exit 1
fi
if [[ ! -f "$FW" ]]; then
    echo "오류: UEFI 펌웨어를 찾을 수 없습니다: $FW" >&2
    exit 1
fi

if [[ "$DAEMON" -eq 1 ]]; then
    # pidfile 이 유실돼도 QEMU 자체의 디스크 이미지 잠금이 이중 기동을 막는 최후 방어선이다.
    if [[ -f "$PIDFILE" ]]; then
        OLD_PID="$(cat "$PIDFILE")"
        if [[ -n "$OLD_PID" ]] && kill -0 "$OLD_PID" 2>/dev/null; then
            echo "오류: VM이 이미 실행 중입니다 (pid $OLD_PID). 먼저 종료하세요." >&2
            exit 1
        fi
        echo "낡은 pidfile을 제거합니다: $PIDFILE"
        rm -f "$PIDFILE"
    fi
fi

# 쓰기 가능한 VARS pflash 는 의도적으로 붙이지 않는다: FreeBSD 는 ESP 폴백 경로
# (\EFI\BOOT\bootaa64.efi)로 부팅하므로 UEFI 변수 영속화가 필요 없다.
# shellcheck disable=SC2054  # qemu 옵션의 콤마는 의도된 문법
QEMU_ARGS=(
    -M virt -accel hvf -cpu host -smp "$CPUS" -m "$MEM"
    -drive if=pflash,format=raw,file="$FW",readonly=on
    -drive if=virtio,format=qcow2,file="$DISK"
    -netdev user,id=n0,hostfwd=tcp:127.0.0.1:10022-:22
    -device virtio-net-pci,netdev=n0
)

if [[ "$DAEMON" -eq 1 ]]; then
    qemu-system-aarch64 "${QEMU_ARGS[@]}" \
        -display none -serial "unix:$CONSOLE_SOCK,server=on,wait=off" \
        -daemonize -pidfile "$PIDFILE"
    echo "VM을 데몬 모드로 기동했습니다 (pid $(cat "$PIDFILE"))."
    echo "콘솔 접속: nc -U $CONSOLE_SOCK"
    echo "종료: ssh -p 10022 root@127.0.0.1 poweroff"
else
    exec qemu-system-aarch64 "${QEMU_ARGS[@]}" -nographic
fi
