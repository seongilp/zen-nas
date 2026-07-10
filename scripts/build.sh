#!/usr/bin/env bash
# 맥에서 실행: nanobsd 설정을 VM에 밀어넣고 빌드 후 이미지를 회수한다.
# 사용법: scripts/build.sh          # 전체 빌드 (최초, 수 시간)
#         scripts/build.sh -b      # world/kernel 재빌드 생략 (설정만 바꿨을 때, 수 분)
# 주의: embedded 프레임워크는 embedded/ 디렉토리 안에서 실행해야 한다
#       (common 소싱과 경로 파생이 pwd 기준 — Task 6 정찰로 확인).
set -euo pipefail
cd "$(dirname "$0")/.."

SSH_OPTS=(-p 10022 root@127.0.0.1)
IMG_REMOTE="/usr/embedded/images/_.disk.image.zen"   # embedded 프레임워크 산출물 경로

if [[ ! -f ~/.ssh/id_ed25519.pub ]]; then
    echo "오류: ~/.ssh/id_ed25519.pub 이 없습니다 (이미지의 root 로그인 키로 사용)" >&2
    exit 1
fi

echo "== 1/5 설정 동기화 =="
# 삭제된 로컬 파일이 VM에 남아 이미지에 섞이지 않도록, 이 단계가 소유한 경로만 초기화
# (/zen/packages 는 vm-fetch-pkgs.sh 소유이므로 건드리지 않는다)
ssh "${SSH_OPTS[@]}" 'rm -rf /zen/overlay /zen/zen.conf && mkdir -p /zen'
# --no-xattrs: macOS 확장 속성(com.apple.provenance)이 아카이브에 들어가면
#   FreeBSD 쪽 추출이 "Cannot restore extended attributes"로 실패한다 (dry-run으로 확인).
# --no-mac-metadata: --no-xattrs만 주면 bsdtar가 메타데이터를 AppleDouble(._*) 파일로
#   바꿔 담아 이미지 오버레이에 ._* 쓰레기가 섞인다 (dry-run으로 확인). 둘 다 필요.
tar --no-xattrs --no-mac-metadata -C nanobsd -cf - . | ssh "${SSH_OPTS[@]}" 'tar -xf - -C /zen'

echo "== 2/5 패키지 프리플라이트 =="
if ! ssh "${SSH_OPTS[@]}" "find /zen/packages -name '*.pkg' 2>/dev/null | grep -q ."; then
    echo "오류: /zen/packages 가 비어 있습니다. vm-fetch-pkgs.sh 를 먼저 실행하세요." >&2
    exit 1
fi

echo "== 3/5 SSH 공개키를 이미지 오버레이에 주입 =="
# 키를 셸 문자열에 삽입하지 않고 stdin으로 전달 (주석 필드의 특수문자 안전)
ssh "${SSH_OPTS[@]}" 'mkdir -p /zen/overlay/root/.ssh \
    && cat > /zen/overlay/root/.ssh/authorized_keys \
    && chmod 700 /zen/overlay/root/.ssh \
    && chmod 600 /zen/overlay/root/.ssh/authorized_keys' < ~/.ssh/id_ed25519.pub

echo "== 4/5 NanoBSD 빌드 (로그: VM의 /usr/embedded/obj/_.bw 등) =="
# 참고: 전달 인자는 공백 없는 단일 토큰 플래그만 지원 (-b 등)
# shellcheck disable=SC2029  # $*는 -b 같은 옵션을 원격 셸로 전달하려는 의도된 클라이언트측 확장이다
ssh "${SSH_OPTS[@]}" "cp /zen/zen.conf /usr/src/tools/tools/nanobsd/embedded/zen.conf \
    && cd /usr/src/tools/tools/nanobsd/embedded \
    && sh ../nanobsd.sh -c zen.conf $*"

echo "== 5/5 이미지 회수 =="
mkdir -p build
scp -P 10022 "root@127.0.0.1:${IMG_REMOTE}" build/zen.img
ls -lh build/zen.img
echo "완료. scripts/test-boot.sh 로 부팅 테스트하세요."
