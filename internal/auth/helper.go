package auth

import "github.com/gofiber/fiber/v3"

// GetClaims is a type-safe helper to get user data from the context
func GetClaims(c fiber.Ctx) *BetterAuthClaims {
	claims, ok := c.Locals("claims").(*BetterAuthClaims)
	if !ok {
		return nil
	}
	return claims
}
