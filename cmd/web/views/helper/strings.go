package helper

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"unicode"
)

func RandomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func Capitalize(s string) string {
	var result strings.Builder
	prevSpace := true
	for _, r := range s {
		if prevSpace && unicode.IsLetter(r) {
			result.WriteRune(unicode.ToUpper(r))
		} else {
			result.WriteRune(r)
		}
		prevSpace = unicode.IsSpace(r)
	}
	return result.String()
}

func TruncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if lastSpace := strings.LastIndex(s[:maxLen], " "); lastSpace > 0 {
		return s[:lastSpace] + "..."
	}
	return s[:maxLen] + "..."
}

// FormatUnits handles scaling for any unit (Bytes, Grams, Liters, or plain numbers).
// value:   The number to format
// unit:    The symbol (e.g., "B", "g", "L"). Pass "" for plain counts.
// divisor: Usually 1000 (standard/metric) or 1024 (binary/data).
func FormatUnits(value int64, unit string, divisor float64) string {
	// Handle the unit-less space: if there's a unit, we want a space (e.g., "1.2 kB")
	// If no unit, we usually don't want a space (e.g., "1.2k")
	space := ""
	if unit != "" {
		space = " "
	}

	if float64(value) < divisor {
		return fmt.Sprintf("%d%s%s", value, space, unit)
	}

	// k = kilo/kibi, M = Mega/Mebi, G = Giga/Gibi, T = Tera/Tebi
	prefixes := []string{"", "k", "M", "G", "T", "P"}

	// Calculate the power/exponent
	exp := int(math.Log(float64(value)) / math.Log(divisor))
	if exp >= len(prefixes) {
		exp = len(prefixes) - 1
	}

	val := float64(value) / math.Pow(divisor, float64(exp))

	// For whole numbers, show no decimals. For others, show 1 (e.g., 1.5k)
	format := "%.1f"
	if val == math.Floor(val) {
		format = "%.0f"
	}

	return fmt.Sprintf(format+"%s%s%s", val, prefixes[exp], space, unit)
}
