package auth

import "github.com/gofiber/fiber/v3"

// GetClaims is a type-safe helper to get user data from the context
func GetClaims(c fiber.Ctx) *Claims {
	claims, ok := c.Locals("claims").(Claims)
	if !ok {
		return &Claims{}
	}
	return &claims
}
