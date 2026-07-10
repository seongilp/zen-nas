#!/usr/bin/env bash
# FreeBSD aarch64 빌드 VM 디스크를 다운로드·준비한다 (최초 1회).
set -euo pipefail

cd "$(dirname "$0")/.."

FREEBSD_VER="${FREEBSD_VER:-15.1}"
VM_DIR="vm"
DISK="$VM_DIR/builder.qcow2"
# 참고: download.freebsd.org 는 최근 릴리스만 유지한다 (구버전은 archive.freebsd.org 로 이동).
# 15.x 부터는 접미사 없는 파일이 없고 -ufs/-zfs 변형만 제공된다. 15.1-RELEASE 로 검증됨.
URL="https://download.freebsd.org/releases/VM-IMAGES/${FREEBSD_VER}-RELEASE/aarch64/Latest/FreeBSD-${FREEBSD_VER}-RELEASE-arm64-aarch64-ufs.qcow2.xz"

if ! command -v qemu-system-aarch64 >/dev/null; then
    echo "오류: qemu가 없습니다. brew install qemu 를 먼저 실행하세요." >&2
    exit 1
fi

if [[ -f "$DISK" ]]; then
    echo "이미 존재합니다: $DISK (다시 만들려면 삭제 후 재실행)"
    exit 0
fi

# 압축 해제 중간물(.part)은 중단 시 반드시 지운다.
# 다운로드 중간물(.xz.part)은 curl -C - 이어받기를 위해 남겨 둔다.
trap 'rm -f "$DISK.part"' EXIT

mkdir -p "$VM_DIR"

if [[ ! -f "$DISK.xz" ]]; then
    echo "== 다운로드: $URL =="
    curl -fL -C - --progress-bar -o "$DISK.xz.part" "$URL"
    mv "$DISK.xz.part" "$DISK.xz"
fi

echo "== 압축 해제 =="
xz -dc "$DISK.xz" > "$DISK.part"

echo "== 이미지 무결성 확인 =="
qemu-img check "$DISK.part" >/dev/null

mv "$DISK.part" "$DISK"
rm -f "$DISK.xz"

echo "== 디스크 80G로 확장 (FreeBSD가 첫 부팅 때 자동 growfs) =="
qemu-img resize "$DISK" 80G
echo "완료. scripts/vm-run.sh 로 부팅한 뒤 docs/vm-bootstrap.md 절차를 따르세요."
