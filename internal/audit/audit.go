package audit

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

type Result struct {
	OS                 string
	Arch               string
	AvailableRAMMB     int
	AvailableStorageMB int64
	NPUAvailable       bool
	CPUCores           int
}

func Run() (*Result, error) {
	r := &Result{}
	r.OS = runtime.GOOS
	r.Arch = runtime.GOARCH
	r.AvailableRAMMB = readMemInfo()
	storagePath := "/cognitiveos"
	if d := os.Getenv("CPM_PATCHES_DIR"); d != "" {
		storagePath = filepath.Dir(d)
	}
	r.AvailableStorageMB = freeStorage(storagePath)
	r.NPUAvailable = hasNPU()
	r.CPUCores = numCPU()
	return r, nil
}

func Check(req *archive.HardwareReq, res *Result) error {
	if req == nil {
		return nil
	}

	if len(req.OS) > 0 {
		supported := false
		for _, os := range req.OS {
			if os == res.OS {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("unsupported OS: requires one of %v, current is %s", req.OS, res.OS)
		}
	}

	if len(req.Arch) > 0 {
		supported := false
		for _, arch := range req.Arch {
			if arch == res.Arch {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("unsupported architecture: requires one of %v, current is %s", req.Arch, res.Arch)
		}
	}

	if req.MinRAMMB > 0 && res.AvailableRAMMB < req.MinRAMMB {
		return fmt.Errorf("requires %d MB RAM, available %d MB", req.MinRAMMB, res.AvailableRAMMB)
	}
	if req.MinStorageMB > 0 && res.AvailableStorageMB < int64(req.MinStorageMB) {
		return fmt.Errorf("requires %d MB storage, available %d MB", req.MinStorageMB, res.AvailableStorageMB)
	}
	if req.NPURequired && !res.NPUAvailable {
		return fmt.Errorf("NPU required but not available")
	}
	return nil
}

func readMemInfo() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		var key string
		var val int
		if _, err := fmt.Sscanf(s.Text(), "%s %d", &key, &val); err == nil && key == "MemAvailable:" {
			return val / 1024
		}
	}
	return 0
}

func freeStorage(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	return int64(stat.Bavail) * int64(stat.Bsize) / (1024 * 1024)
}

func hasNPU() bool {
	if _, err := os.Stat("/sys/class/npu"); err == nil {
		return true
	}
	return false
}

func numCPU() int {
	return int(math.Max(1, float64(runtime.NumCPU())))
}
