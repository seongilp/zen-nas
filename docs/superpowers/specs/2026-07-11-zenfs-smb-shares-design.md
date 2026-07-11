# zenfs SMB 공유 관리 설계

- 날짜: 2026-07-11
- 상태: 승인됨 (사용자 "go, 한큐에")

## 목적

zenfs에서 데이터셋을 바로 SMB 공유로 내보내고 해제한다. 읽기전용 루트를
rw 리마운트하지 않고, NanoBSD의 /etc(tmpfs)+/cfg 영속화 흐름에 올라탄다.

## 아키텍처

- **이미지 측**: `smb4.conf` [global]에 `include = /etc/smb-shares.conf` 추가.
  오버레이가 빈 `/etc/smb-shares.conf`를 이미지 base에 포함시켜 include가
  항상 성공하고, "수정된 파일"로 취급되어 save_cfg가 프롬프트 없이 영속화.
- **zenfs 측**: 새 패키지 `tui/smb/` — zenfs가 완전히 소유하는 단순 포맷의
  공유 블록을 `/etc/smb-shares.conf`에 읽고 쓴다 (경로 주입 가능 = 테스트).
  적용은 `service samba_server reload`, 영속화는 `yes n | sh /root/save_cfg`
  (수정 파일은 자동 저장, 무관한 신규 파일 프롬프트는 전부 n).

## 파일 포맷 (zenfs 소유 — 손편집 비권장 주석 포함)

```ini
# zenfs가 관리하는 SMB 공유 정의. smb4.conf가 include 한다.
[data]
    path = /tank/data
    read only = no
```

## 기능

| 키 | 위치 | 동작 |
|---|---|---|
| `m` | 데이터셋 상세 | 미공유 → 공유 이름 폼(기본값: 데이터셋 짧은 이름) → 블록 추가 + samba reload. 공유 중 → 블록 제거 + reload (설정 변경은 가역이라 이름 타이핑 확인 없음) |
| `w` | 전역 | `yes n \| sh /root/save_cfg` 실행 — 공유 설정 /cfg 영속화 |

- 데이터셋 상세 패널에 `smb 공유` 행 표시 (공유 이름 또는 "없음").
- 공유 추가/제거 후 상태바에 "저장하려면 [w]" 안내.
- 공유 이름 충돌 시 기존 블록 교체가 아니라 거부(에러 표면화).
- 마운트포인트 없는 데이터셋(예: zvol)에는 `m` 비활성.

## 범위 제외 (v1)

- NFS exports 관리(수동 유지), valid users 지정(전역 인증 정책 따름),
  Time Machine 옵션, smb4.conf 자체 편집.

## 테스트

- `tui/smb/` TDD: 파싱·추가·제거 라운드트립, 이름 충돌 거부, 빈 파일.
- UI: `m` 토글 통합 테스트 (임시 파일 경로 주입).
- E2E(QEMU): 풀·데이터셋 생성 → zenfs로 공유 추가 → `testparm -s`로 samba가
  공유를 인식하는지 확인 → `w` → /cfg에 반영 확인.
