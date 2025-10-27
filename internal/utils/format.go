package utils

import "fmt"

// BytesToHuman converts bytes into a human-readable string (KiB, MiB, GiB).
func BytesToHuman(b int64) string {
	if b < 0 {
		return "0 B"
	}
	const (
		KiB = 1024
		MiB = KiB * 1024
		GiB = MiB * 1024
	)
	switch {
	case b >= GiB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GiB))
	case b >= MiB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MiB))
	case b >= KiB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
