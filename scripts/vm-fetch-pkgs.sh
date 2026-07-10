#!/bin/sh
# VM 안에서 실행: 이미지에 넣을 amd64 패키지를 미리 받는다.
# 사용법(맥에서): ssh -p 10022 root@127.0.0.1 sh < scripts/vm-fetch-pkgs.sh
set -eu

MAJOR=$(freebsd-version -u | cut -d. -f1)
ABI="FreeBSD:${MAJOR}:amd64"
PKGDIR="/zen/packages"
DBDIR="/zen/pkgdb-amd64"
# 이전 실행의 낡은 패키지가 이미지에 섞이지 않도록 매번 비우고 새로 받는다
rm -rf "$PKGDIR"
mkdir -p "$PKGDIR" "$DBDIR"

pkg_amd64() {
    pkg -o ABI="$ABI" -o PKG_DBDIR="$DBDIR" -o IGNORE_OSVERSION=yes "$@"
}

pkg_amd64 update

# samba 포트는 버전이 이름에 붙으므로(samba423 등) 최신을 동적으로 고른다
# samba 4.x 계열에 고정 (5.x 전환 시 이 정규식 갱신 필요)
SAMBA=$(pkg_amd64 search -q '^samba4[0-9]*-4' | sort -V | tail -1 | sed 's/-[0-9].*//')
if [ -z "$SAMBA" ]; then
    echo "오류: samba 패키지를 찾지 못했습니다" >&2
    exit 1
fi
echo "== 선택된 samba: $SAMBA =="

# pkg 자체도 함께 받는다. freebsd-ftpd: 15.x부터 base에서 제거되어 패키지로 필요.
pkg_amd64 fetch -d -y -o "$PKGDIR" pkg "$SAMBA" rsync lighttpd freebsd-ftpd

echo "== 받은 패키지 =="
find "$PKGDIR" -name '*.pkg'
