package utils

import "fmt"

func FmtMem(bytes uintptr) string {
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
		return fmt.Sprintf("%dTB %dGB %dMB %dKB %dB", t, rem/GB, (rem%GB)/MB, (rem%MB)/KB, rem%KB)
	case bytes >= GB:
		g := bytes / GB
		rem := bytes % GB
		return fmt.Sprintf("%dGB %dMB %dKB %dB", g, rem/MB, (rem%MB)/KB, rem%KB)
	case bytes >= MB:
		m := bytes / MB
		rem := bytes % MB
		return fmt.Sprintf("%dMB %dKB %dB", m, rem/KB, rem%KB)
	case bytes >= KB:
		k := bytes / KB
		return fmt.Sprintf("%dKB %dB", k, bytes%KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
