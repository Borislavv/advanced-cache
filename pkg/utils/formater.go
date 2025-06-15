package utils

import "fmt"

func FmtMem(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		t := bytes / TB
		rem := bytes % TB
		return fmt.Sprintf("%dTB %dGB", t, rem/GB)
	case bytes >= GB:
		g := bytes / GB
		rem := bytes % GB
		return fmt.Sprintf("%dGB %dMB", g, rem/MB)
	case bytes >= MB:
		m := bytes / MB
		rem := bytes % MB
		return fmt.Sprintf("%dMB %dKB", m, rem/KB)
	case bytes >= KB:
		k := bytes / KB
		return fmt.Sprintf("%dKB %dB", k, bytes%KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
