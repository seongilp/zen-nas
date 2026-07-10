#!/usr/bin/env bash
# zen 이미지를 USB에 굽는다. 외장·이동식 디스크만 허용한다.
set -euo pipefail

# 외장 디스크가 아니면 즉시 중단한다.
assert_external_disk() {
    local disk="$1" fail_msg="$2" internal
    internal=$(diskutil info -plist "$disk" | plutil -extract Internal raw -)
    if [[ "$internal" != "false" ]]; then
        echo "오류: $fail_msg" >&2
        exit 1
    fi
}

IMG="${1:-build/zen.img}"
if [[ ! -f "$IMG" ]]; then
    echo "오류: 이미지 파일이 없습니다: $IMG" >&2
    echo "사용법: $0 [이미지경로]" >&2
    exit 1
fi

echo "== 연결된 외장 물리 디스크 =="
diskutil list external physical
echo

read -r -p "구울 대상 디스크 식별자 (예: disk4): " DISK
if [[ ! "$DISK" =~ ^disk[0-9]+$ ]]; then
    echo "오류: 'diskN' 형식이어야 합니다: $DISK" >&2
    exit 1
fi

# 안전장치 1: 내장 디스크 거부
assert_external_disk "$DISK" "$DISK 는 내장 디스크입니다. 중단합니다."

# 안전장치 2: 대상 정보를 보여주고 명시적 확인
diskutil info "$DISK" | grep -E "Device Node|Media Name|Disk Size"
echo
read -r -p "위 디스크의 모든 데이터가 삭제됩니다. 계속하려면 ERASE 입력: " CONFIRM
if [[ "$CONFIRM" != "ERASE" ]]; then
    echo "취소했습니다."
    exit 0
fi

# 안전장치 3: 확인 대기 중 디스크가 바뀌었을 수 있으므로 쓰기 직전 재검증
assert_external_disk "$DISK" "확인 후 $DISK 상태가 바뀌었습니다. 중단합니다."

diskutil unmountDisk "/dev/$DISK"
echo "== 굽는 중: $IMG → /dev/r$DISK =="
sudo dd if="$IMG" of="/dev/r$DISK" bs=4m status=progress
sync
diskutil eject "/dev/$DISK"
echo "완료. USB를 NAS에 꽂고 부팅하세요."
