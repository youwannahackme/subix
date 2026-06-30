package utils

import (
	"fmt"
	"strings"
)

// Color constants for terminal output
const (
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
)

// ColorCyan wraps text in cyan color
func ColorCyan(text string) string {
	return colorCyan + text + colorReset
}

// ColorGreen wraps text in green color
func ColorGreen(text string) string {
	return colorGreen + text + colorReset
}

// ColorRed wraps text in red color
func ColorRed(text string) string {
	return colorRed + text + colorReset
}

// ColorYellow wraps text in yellow color
func ColorYellow(text string) string {
	return colorYellow + text + colorReset
}

// ColorBold wraps text in bold
func ColorBold(text string) string {
	return colorBold + text + colorReset
}

// PadRight pads a string on the right to reach the target width
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// PadLeft pads a string on the left to reach the target width
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// FormatNumber formats a number with comma separators
func FormatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i := len(s) - 1; i >= 0; i-- {
		pos := len(s) - 1 - i
		if pos > 0 && pos%3 == 0 {
			result = append([]byte{','}, result...)
		}
		result = append([]byte{s[i]}, result...)
	}
	return string(result)
}

// TruncateString truncates a string to maxLen and adds "..." if needed
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ContainsString checks if a slice contains a string
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// UniqueStrings removes duplicates from a string slice
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
