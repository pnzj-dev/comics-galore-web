package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

func FormatSize(size int64) string {
	if size <= 0 {
		return "0 B"
	} // Return with unit
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	val := float64(size)
	// Calculate appropriate unit, preventing index out of bounds
	for val >= 1024 && unitIndex < len(units)-1 {
		val /= 1024
		unitIndex++
	}
	// Use %.1f for KB and up, %d for Bytes
	if unitIndex == 0 {
		return fmt.Sprintf("%d %s", int64(val), units[unitIndex])
	}
	return fmt.Sprintf("%.1f %s", val, units[unitIndex])
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // Character pool
	randomString := make([]byte, n)
	for i := range randomString {
		randomIndex := seededRand.Intn(len(letters))
		randomString[i] = letters[randomIndex]
	}
	return string(randomString)
}

func GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(slug, "")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Trim(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	return slug
}
