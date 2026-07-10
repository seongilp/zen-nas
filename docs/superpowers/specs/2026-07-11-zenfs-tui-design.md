# zenfs — ZFS 관리 TUI 설계

- 날짜: 2026-07-11
- 상태: 승인됨

## 목적

zen NAS에서 SSH 접속 후 `zenfs` 한 번으로 ZFS 전반(풀·데이터셋·스냅샷)을
조회·관리하는 터미널 UI. CLI 명령을 외우지 않고도 일상 관리와 풀 수준
작업(생성·파괴·scrub·디스크 교체)까지 수행한다.

## 요구사항

| 항목 | 결정 |
|---|---|
| 기능 범위 | 풀 관리까지 전부 (조회 + 데이터셋/스냅샷 + 풀 생성·파괴·scrub·replace) |
| 기술 | Go + bubbletea, `CGO_ENABLED=0` 정적 바이너리 |
| ZFS 인터페이스 | `zfs`/`zpool` CLI 실행 + 머신 파싱 출력(`-H -p`) — libzfs cgo 기각 |
| 레이아웃 | 좌측 트리(풀→데이터셋→스냅샷) + 우측 상세 패널 + 하단 상태바 |
| 실행 환경 | zen 이미지(FreeBSD 15 amd64, 읽기전용 루트), root로 실행 |
| 도구 이름 | `zenfs` (`/usr/local/bin/zenfs`) |

## 아키텍처

### 배포 경로

- 코드: zen 리포 `tui/` (Go 모듈). 바이너리는 git에 커밋하지 않는다.
- `build.sh`에 컴파일 단계 추가: 맥에서 `GOOS=freebsd GOARCH=amd64
  CGO_ENABLED=0 go build` → SSH 키 주입과 같은 방식으로 빌드 시점에
  `/zen/overlay/usr/local/bin/zenfs`로 주입 → 이미지에 포함.
- 맥에 Go가 없으면 build.sh가 명확한 에러로 중단.

### 코드 구조 (UI ↔ ZFS 계층 분리)

```
tui/
├── go.mod, main.go      # 엔트리
├── zfs/                 # ZFS 인터페이스 계층 — bubbletea 의존 없음
│   ├── exec.go          #   Runner 인터페이스 (Run(ctx, name, args...) →
│   │                    #   stdout/stderr/err). 실구현 + 테스트 mock.
│   ├── types.go         #   Pool / Dataset / Snapshot / Disk 구조체
│   ├── list.go          #   조회 파싱: zpool list/status, zfs list, geom
│   └── ops.go           #   변경 작업: 명령 조립 (create/destroy/snapshot/
│                        #   rollback/scrub/replace)
└── ui/                  # bubbletea — zfs 패키지에만 의존
    ├── app.go           #   루트 모델, 패널 포커스 전환, 키맵
    ├── tree.go          #   좌측 트리
    ├── detail.go        #   우측 상세 + 컨텍스트 액션 목록
    ├── forms.go         #   생성 폼 (풀 생성 시 geom 기반 디스크 선택)
    ├── confirm.go       #   위험 작업 확인 모달
    └── statusbar.go     #   풀 건강도·scrub 진행률·단축키 힌트
```

### 데이터 흐름

- 조회는 CLI 머신 출력 파싱: `zpool list -Hp`, `zpool status`,
  `zfs list -Hp -t filesystem,volume,snapshot -o ...`, 디스크 후보는
  `geom disk list` 계열.
- 리프레시: 수동(`r`) + 모든 변경 작업 직후 자동. scrub 진행 중일 때만
  2초 주기 폴링. (상시 자동 폴링은 YAGNI)
- 변경 작업은 `ops.go`가 명령을 조립해 Runner로 실행 — UI는 명령 문자열을
  받아 확인 모달에 그대로 표시한다.

## 기능 매핑

| 대상 | 조회 | 변경 |
|---|---|---|
| 풀 | 목록·상태(vdev 트리)·용량·scrub 진행률·오류 카운트 | 생성(디스크 선택 + single/mirror/raidz1), 파괴, scrub 시작/중지, 디스크 replace |
| 데이터셋 | 트리·used/avail·compression·mountpoint·스냅샷 수 | 생성(이름 + compression 선택), 파괴(-r은 하위 존재 시 명시) |
| 스냅샷 | 데이터셋별 목록·used·생성시각 | 생성(기본 이름 `@manual-YYYYMMDD-HHMM` 제안), 롤백, 파괴 |

풀 생성 시 디스크 선택 목록에는 이미 풀에 소속된 디스크와 OS 디스크
(루트가 마운트된 디스크)를 제외하거나 명확히 경고 표시한다.

## 안전장치

- **파괴(풀·데이터셋)·롤백·replace는 대상 이름을 그대로 타이핑해야 실행**
  (flash-usb.sh의 ERASE 확인과 동일한 철학).
- 확인 모달에 **실제 실행될 CLI 명령 전문을 표시**한다.
- 롤백은 대상 스냅샷 이후의 스냅샷이 함께 삭제됨을 경고에 명시한다.
- 명령 실패 시 stderr 전문을 모달로 표시하고 TUI는 계속 동작한다.
- root가 아니면 시작 시 경고 배너(조회는 동작할 수 있으나 변경은 실패).

## 에러 처리

- 모든 Runner 호출에 타임아웃(기본 30초; scrub 시작 등 즉시 반환 명령 기준).
- 파싱 실패(예: 예상 밖 출력)는 해당 항목만 비우고 상태바에 경고 — 전체
  크래시 금지.
- ZFS 모듈 미로드/풀 없음 상태에서도 빈 트리 + 안내 문구로 정상 기동.

## 테스트 전략

1. **파서 유닛 테스트**: 실제 FreeBSD 15 `zpool`/`zfs` 출력을 testdata
   픽스처로 저장해 검증 (QEMU 게스트에서 채취).
2. **ops 유닛 테스트**: mock Runner로 "정확한 인자 배열이 조립되는가" 검증
   (TDD — 명령 조립이 이 도구의 안전성 핵심).
3. **UI 모델 테스트**: 핵심 상호작용(트리 이동, 확인 모달의 이름 불일치
   거부)만 bubbletea Update 단위 테스트.
4. **통합 스모크**: 이미지 빌드 후 QEMU 게스트에서 `zenfs`가 존재하고
   `--version` 실행되는지 확인 (TUI 자체는 수동 확인).

## 기각한 대안

- **libzfs cgo 바인딩**: FreeBSD 크로스 cgo 툴체인 부담 + libzfs ABI
  불안정. 맥 크로스 빌드 흐름과 충돌.
- **데몬 + 클라이언트 분리**: 단일 사용자 로컬 도구에 과설계.
- **sh + bsddialog / Python textual**: 각각 표현력 한계 / 이미지 의존성
  비대화.
