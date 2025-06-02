package utils

import (
	"fmt"
	"strconv"
)

func FmtMemory(bytes uintptr) string {
	b := int(bytes)

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		t := b / TB
		rem := b % TB
		return strconv.Itoa(t) + "TB " +
			strconv.Itoa(rem/GB) + "GB " +
			strconv.Itoa((rem%GB)/MB) + "MB " +
			strconv.Itoa((rem%MB)/KB) + "KB " +
			strconv.Itoa(rem%KB) + "B"
	case b >= GB:
		g := b / GB
		rem := b % GB
		return strconv.Itoa(g) + "GB " +
			strconv.Itoa(rem/MB) + "MB " +
			strconv.Itoa((rem%MB)/KB) + "KB " +
			strconv.Itoa(rem%KB) + "B"
	case b >= MB:
		m := b / MB
		rem := b % MB
		return strconv.Itoa(m) + "MB " +
			strconv.Itoa(rem/KB) + "KB " +
			strconv.Itoa(rem%KB) + "B"
	case b >= KB:
		k := b / KB
		return strconv.Itoa(k) + "KB " +
			strconv.Itoa(b%KB) + "B"
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func BoolToString(val bool) string {
	if val {
		return "true"
	}
	return "false"
}
