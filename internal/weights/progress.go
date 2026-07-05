package weights

import (
	"fmt"
	"os"
)

type ProgressFn func(current, total int64)

func NoopProgress(current, total int64) {}

func TextProgress(current, total int64) {
	if total <= 0 {
		fmt.Fprintf(os.Stderr, "\rDownloaded %d bytes", current)
		return
	}
	pct := float64(current) / float64(total) * 100
	fmt.Fprintf(os.Stderr, "\r%.0f%% (%d / %d bytes)", pct, current, total)
}
