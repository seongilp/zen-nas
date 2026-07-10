#!/bin/sh
# VM 안에서 실행: NanoBSD 크로스 빌드 환경을 구축한다.
# 사용법(맥에서): ssh -p 10022 root@127.0.0.1 sh < scripts/vm-provision.sh
set -eu

echo "== 1/2 필수 패키지 설치 =="
env ASSUME_ALWAYS_YES=yes pkg bootstrap
pkg install -y git-lite rsync

echo "== 2/2 FreeBSD 소스 체크아웃 =="
REL=$(freebsd-version -u | sed 's/-.*//')          # 예: 15.1
BRANCH="releng/${REL}"
if [ ! -d /usr/src/.git ]; then
    git clone --depth 1 --branch "$BRANCH" https://git.freebsd.org/src.git /usr/src
else
    echo "/usr/src 이미 존재 — 건너뜀"
fi
mkdir -p /zen
echo "프로비저닝 완료: 소스 브랜치 $BRANCH"
