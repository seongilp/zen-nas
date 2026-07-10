#!/usr/bin/env bash
# FreeBSD aarch64 빌드 VM 디스크를 다운로드·준비한다 (최초 1회).
set -euo pipefail

FREEBSD_VER="${FREEBSD_VER:-15.1}"
VM_DIR="vm"
DISK="$VM_DIR/builder.qcow2"
URL="https://download.freebsd.org/releases/VM-IMAGES/${FREEBSD_VER}-RELEASE/aarch64/Latest/FreeBSD-${FREEBSD_VER}-RELEASE-arm64-aarch64-ufs.qcow2.xz"

if ! command -v qemu-system-aarch64 >/dev/null; then
    echo "오류: qemu가 없습니다. brew install qemu 를 먼저 실행하세요." >&2
    exit 1
fi

if [[ -f "$DISK" ]]; then
    echo "이미 존재합니다: $DISK (다시 만들려면 삭제 후 재실행)"
    exit 0
fi

mkdir -p "$VM_DIR"
echo "== 다운로드: $URL =="
curl -fL --progress-bar -o "$DISK.xz" "$URL"
echo "== 압축 해제 =="
xz -d "$DISK.xz"
echo "== 디스크 80G로 확장 (FreeBSD가 첫 부팅 때 자동 growfs) =="
qemu-img resize "$DISK" 80G
echo "완료. scripts/vm-run.sh 로 부팅한 뒤 docs/vm-bootstrap.md 절차를 따르세요."
