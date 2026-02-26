package utils

import (
	"time"
)

func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006 15:04:05")
}

func IsOlderThanHours(givenDate time.Time, hours int) bool {
	if givenDate.IsZero() {
		return false
	}
	threshold := time.Duration(hours) * time.Hour
	return time.Since(givenDate) > threshold
}
