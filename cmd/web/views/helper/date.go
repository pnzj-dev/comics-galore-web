package helper

import "time"

func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("January 2, 2006")
}

// FormatDateTime : Converts time.Time to "Mon D, YYYY at H:MM PM" format.
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("Jan 2, 2006 at 3:04 PM")
}
func CurrentYear() int    { return time.Now().Year() }
func CurrentDate() string { return time.Now().Format("January 2, 2006") }
func IsOlderThanHours(givenDateStr string, hours int) bool {
	givenDate, err := time.Parse(time.RFC3339, givenDateStr)
	if err != nil {
		return false
	}
	threshold := time.Duration(hours) * time.Hour
	return time.Since(givenDate) > threshold
}
