package auth

import "strings"

func (h *handler) getFormTypeFromPath(path string) string {
	// 1. Remove the base API prefix
	// Result: "/sign-in/email" or "/sign-up/email" or "/reset-password"
	actionPath := strings.TrimPrefix(path, "/api/v1/auth")

	// 2. Simple mapping based on the contains logic
	// This handles variations like "/sign-in/email" vs "/sign-in/google"
	switch {
	case strings.Contains(actionPath, "sign-up"):
		return "sign-up"
	case strings.Contains(actionPath, "sign-in"):
		return "sign-in"
	case strings.Contains(actionPath, "sign-out"):
		return "sign-out"
	case strings.Contains(actionPath, "reset-password"):
		return "reset-password"
	default:
		return ""
	}
}
