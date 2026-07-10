package zfs

// Pool: zpool list 한 행.
type Pool struct {
	Name   string
	Size   uint64
	Alloc  uint64
	Free   uint64
	Health string
}

type DatasetType string

const (
	TypeFilesystem DatasetType = "filesystem"
	TypeVolume     DatasetType = "volume"
	TypeSnapshot   DatasetType = "snapshot"
)

// Dataset: zfs list 한 행 (스냅샷 포함).
type Dataset struct {
	Name        string
	Type        DatasetType
	Used        uint64
	Avail       uint64 // 스냅샷은 0
	Refer       uint64
	Mountpoint  string // 스냅샷은 ""
	Compression string // 스냅샷은 ""
}

// PoolStatus: zpool status 파싱 결과.
type PoolStatus struct {
	Name   string
	State  string
	Scan   string // "scan:" 줄 원문 (없으면 "")
	VDevs  []VDev // config 트리를 깊이와 함께 평탄화
	Errors string
}

type VDev struct {
	Depth int // config 들여쓰기 단계 (0=풀 자체)
	Name  string
	State string
}

// Disk: 풀 생성/교체 후보 디스크.
type Disk struct {
	Name      string // 예: ada1
	SizeBytes uint64
	Note      string // "" | "OS 디스크" | "<pool> 소속"
}

// Cmd: 실행 전 사용자에게 전문을 보여주기 위한 명령 값.
type Cmd struct {
	Name string
	Args []string
}

func (c Cmd) String() string {
	s := c.Name
	for _, a := range c.Args {
		s += " " + a
	}
	return s
}
