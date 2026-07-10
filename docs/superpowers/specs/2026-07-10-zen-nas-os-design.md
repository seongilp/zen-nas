# zen — NanoBSD 기반 홈 NAS OS 설계

- 날짜: 2026-07-10
- 상태: 승인됨

## 목적

개인 홈 NAS를 위한 커스텀 OS. NanoBSD 프레임워크로 FreeBSD 기반의
읽기전용 OS 이미지를 빌드하고, USB에 구워 x86-64 PC에서 부팅한다.
데이터는 별도 디스크의 ZFS 풀에 저장한다.

## 요구사항

| 항목 | 결정 |
|---|---|
| 목적 | 개인 홈 NAS (실사용) |
| 타겟 하드웨어 | x86-64 PC/미니PC, USB 부팅 |
| 빌드 환경 | 맥북 (Apple M4 Pro, 48GB RAM) |
| 데이터 파일시스템 | ZFS |
| 관리 방식 | SSH + 설정파일 (웹 UI 없음) |
| 서비스 | SMB(Samba), NFS, rsync, iSCSI(ctld), WebDAV/FTP |

## 접근법 결정

**순수 NanoBSD 정석 경로.** FreeBSD 소스 트리의
`tools/tools/nanobsd/nanobsd.sh`를 그대로 사용하고, 이 리포는
커스텀 설정(`zen.conf`), 오버레이 파일, 패키지 목록, 빌드/굽기
스크립트만 관리한다.

검토 후 기각한 대안:

- **XigmaNAS 포크**: PHP 웹 UI 중심의 대형 레거시를 덜어내는 비용이
  새로 만드는 것보다 큼. SSH+설정파일 방향과 반대.
- **makefs/mkimg 직접 빌드**: NanoBSD가 이미 해결한 문제(듀얼 파티션,
  cfg 영속화, 이미지 생성)의 재발명.

## 전체 아키텍처

```
[맥북 M4 Pro — macOS는 변경 없음]
  ├─ ① QEMU로 FreeBSD aarch64 빌드 VM 부팅 (네이티브 가상화, 빠름)
  │      └─ VM 안에서 FreeBSD 소스 체크아웃
  │         → nanobsd.sh + zen.conf로 amd64 이미지 크로스 빌드
  ├─ ② scp로 zen.img 회수
  ├─ ③ QEMU x86-64 에뮬레이션으로 부팅 스모크 테스트 (선택)
  └─ ④ dd로 USB 굽기
         ↓
[x86-64 NAS 본체]
  ├─ USB: OS (읽기전용 UFS, 듀얼 슬라이스 A/B + /cfg 파티션)
  └─ SATA/NVMe 디스크: ZFS 풀 (데이터)
```

핵심 원칙:

- **macOS에는 설치물 없음.** Homebrew QEMU가 유일한 도구이고, VM은
  파일 하나로 존재한다. NanoBSD 빌드는 FreeBSD 호스트를 요구하므로
  VM이 필수다.
- **크로스 빌드.** FreeBSD는 aarch64 호스트에서 `TARGET_ARCH=amd64`
  크로스 빌드를 공식 지원한다. Apple Silicon에서 에뮬레이션 없이
  네이티브 속도로 빌드된다.
- **OS와 데이터의 물리적 분리.** USB가 죽어도 ZFS 풀은 무사하다.
  새 USB를 굽고 `zpool import`만 하면 복구된다.

## 리포 구조

```
zen/
├── nanobsd/
│   ├── zen.conf          # NanoBSD 빌드 설정 (이미지 크기, 훅, 커스터마이즈)
│   ├── overlay/          # 이미지에 덮어쓸 파일 트리
│   │   └── etc/          #   rc.conf, smb4.conf 템플릿, 부팅 스크립트 등
│   └── pkgs/             # 미리 설치할 패키지 목록
├── scripts/
│   ├── vm-setup.sh       # 맥: FreeBSD 빌드 VM 준비 (이미지 다운로드, 초기화)
│   ├── build.sh          # 맥: ssh로 VM 안 빌드 트리거 + 이미지 회수
│   ├── test-boot.sh      # 맥: QEMU x86-64로 이미지 부팅 테스트
│   └── flash-usb.sh      # 맥: USB 디스크 확인 후 dd로 굽기
├── docs/
└── README.md
```

FreeBSD 소스는 리포에 두지 않고 VM 안에 체크아웃한다.

## OS 디스크 레이아웃 (NanoBSD 표준)

| 파티션 | 내용 | 마운트 |
|---|---|---|
| s1 | OS 이미지 A | `/` 읽기전용 UFS |
| s2 | OS 이미지 B (업데이트용) | — |
| s3 | 설정 저장소 | `/cfg` (평소 언마운트) |

- `/etc`, `/var`는 부팅 시 tmpfs(MFS)로 생성되고 `/cfg` 내용이
  `/etc`로 복사된다.
- 설정 변경 후 `cfg_save`로 `/cfg`에 영속화한다. USB 쓰기를
  최소화해 수명을 보호하는 NanoBSD의 핵심 설계.

## 소프트웨어 구성

- **베이스**: FreeBSD 최신 RELEASE (빌드 시점에 14.x/15.x 확인),
  ZFS 커널 모듈 포함
- **base 내장 활용**: NFS, ctld(iSCSI), sshd, ftpd
- **pkg로 이미지에 포함**: Samba(SMB + macOS 호환 fruit 모듈),
  rsync, lighttpd(WebDAV)
- **커널**: GENERIC으로 시작. 커스텀 축소는 나중 최적화 과제.

## 업데이트 전략

1. 새 이미지를 빌드한다.
2. NanoBSD의 `updatep1`/`updatep2` 방식으로 비활성 슬라이스에 기록한다.
3. 부트 슬라이스를 전환하고 재부팅한다.
4. 문제가 생기면 이전 슬라이스로 롤백한다.

## 에러 처리 및 복구

- **USB 사망**: 새 USB에 이미지를 굽고 백업에서 `/cfg`를 복원한다.
  데이터 풀은 `zpool import`로 그대로 살아난다.
- **`/cfg` 백업**: `/cfg` 내용을 ZFS 풀에 주기적으로 자동 백업한다.
- **flash-usb.sh 안전장치**: 대상 디스크를 명시적으로 나열·확인시켜
  내장 디스크에 실수로 쓰는 사고를 방어한다.

## 테스트 전략

1. **부팅 스모크 테스트 (자동화)**: 맥에서 QEMU x86-64로 이미지를
   부팅해 SSH 접속과 서비스 기동을 확인한다.
2. **실기 검증 체크리스트**: 실제 하드웨어에서 디스크 인식, ZFS 풀
   생성, SMB/NFS 접속을 확인한다.
