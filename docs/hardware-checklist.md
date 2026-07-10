# 실기 배포 체크리스트

## 준비물
- [ ] 8GB 이상 USB 메모리 (OS용 — 데이터와 분리)
- [ ] x86-64 PC + 데이터용 SATA/NVMe 디스크 1개 이상
- [ ] 유선 랜 연결

## 굽기와 부팅
- [ ] `scripts/flash-usb.sh` 로 USB 굽기
- [ ] BIOS/UEFI 설정에서 USB 부팅 순서 지정 (UEFI·레거시 모두 지원)
- [ ] `zen login:` 프롬프트 확인 (모니터 또는 시리얼 콘솔)
- [ ] 첫 부팅은 파티션 자동 확장 때문에 한 번 더 재부팅될 수 있음
- [ ] 공유기에서 zen의 IP 확인 → `ssh root@<IP>` 접속 (키 인증)

## 초기 설정 (SSH 접속 후)
- [ ] 디스크 확인: `camcontrol devlist` / `nvmecontrol devlist`
- [ ] ZFS 풀 생성 (예: 미러):
      `zpool create tank mirror ada0 ada1`
- [ ] 데이터셋 생성:
      `zfs create tank/data && zfs create -p tank/backup/cfg`
- [ ] 사용자 생성: `pw useradd zen -m -s /bin/sh && passwd zen && smbpasswd -a zen`
- [ ] `/root/save_cfg` 실행으로 /etc 변경 영속화 (passwd/smbpasswd 파일 포함 여부 확인)
- [ ] `/usr/local/etc/smb4.conf` 공유 주석 해제 → `service samba_server restart` → `/root/save_cfg`
- [ ] 맥 Finder에서 `smb://zen.local` 접속 확인
- [ ] WebDAV 사용 시: htpasswd에 사용자 추가 후 `service lighttpd restart`
      `printf 'zen:%s\n' "$(openssl passwd -apr1)" >> /usr/local/etc/lighttpd/htpasswd`
      (lighttpd mod_authn_file htpasswd 백엔드는 APR-MD5(`$apr1$`) 해시 지원; 이후 `/root/save_cfg` 불필요 — /usr/local/etc는 이미지 영역이므로 업데이트 시 다시 추가)
- [ ] iSCSI 사용 시: `/etc/ctl.conf.sample` 참고해 /etc/ctl.conf 작성, rc.conf.local에서 ctld_enable=YES → `/root/save_cfg`
- [ ] 재부팅 내구성 확인: `shutdown -r now` 후 풀 자동 import·서비스 자동 기동·설정 유지 확인

## 업데이트 절차 (가동 중 무중단 갱신)
1. 맥: `scripts/build.sh -b` 로 새 이미지 빌드 (설정만 바뀐 경우 수 분)
2. 슬라이스 이미지를 NAS로 전송 (경로·절차는 Task 11 검증 후 확정)
3. NAS: `/root/updatep1` 또는 `/root/updatep2` (현재 부팅 중이 아닌 쪽) 실행
4. 재부팅 → 문제 시 부트 메뉴에서 이전 슬라이스로 롤백

## 재해 복구
- USB 사망 시: 새 USB에 `scripts/flash-usb.sh` → 부팅 → `zpool import -f tank`
  → `/tank/backup/cfg` 백업에서 /cfg 복원 (`mount /cfg && rsync -a /tank/backup/cfg/ /cfg/ && umount /cfg`) → 재부팅
- 데이터 풀은 OS와 완전히 분리되어 있어 OS 재설치와 무관하게 보존된다
