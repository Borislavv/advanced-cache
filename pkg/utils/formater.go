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
		gb := (bytes % TB) / GB
		if gb > 0 {
			return fmt.Sprintf("%dTB %dGB", t, gb)
		}
		return fmt.Sprintf("%dTB", t)

	case bytes >= GB:
		g := bytes / GB
		mb := (bytes % GB) / MB
		if mb > 0 {
			return fmt.Sprintf("%dGB %dMB", g, mb)
		}
		return fmt.Sprintf("%dGB", g)

	case bytes >= MB:
		m := bytes / MB
		kb := (bytes % MB) / KB
		if kb > 0 {
			return fmt.Sprintf("%dMB %dKB", m, kb)
		}
		return fmt.Sprintf("%dMB", m)

	case bytes >= KB:
		k := bytes / KB
		return fmt.Sprintf("%dKB", k)

	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
