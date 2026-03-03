package modal

import (
	"comics-galore-web/internal/view"
	"encoding/json"
	"fmt"
	"time"
)

// MarshalPlans safely converts the plan slice to a JSON string for Alpine.js
func MarshalPlans(plans []view.SubscriptionPlan) string {
	b, err := json.Marshal(plans)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// FormatTimeAgo converts a time.Time into a relative string like "5m ago"
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration.Seconds() < 60:
		return "Just now"
	case duration.Minutes() < 60:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration.Hours() < 24:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	case duration.Hours() < 48:
		return "Yesterday"
	default:
		return t.Format("Jan 02, 2006")
	}
}
