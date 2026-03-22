package auth

import (
	"bytes"
	"comics-galore-web/internal/auth"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"io"
)

func (h *handler) syncCookies(c fiber.Ctx) error {
	// 1. Extract the JWT from the Proxy Response Headers directly
	// Peek returns []byte, which is memory-efficient
	jwtBytes := c.Response().Header.Peek("Set-Auth-Jwt")

	if len(jwtBytes) == 0 {
		return nil
	}

	jwt := string(jwtBytes)

	// 2. Decode the body to get session details
	session, err := h.extractSessionData(c)
	if err != nil {
		return fmt.Errorf("could not extract session for cookie sync: %w", err)
	}

	// 3. Set the cookie on our domain
	c.Cookie(&fiber.Cookie{
		Name:     "comics-galore-jwt",
		Value:    jwt,
		Expires:  session.Session.ExpiresAt,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/", // Ensure the cookie is available site-wide
	})

	return nil
}

// Internal helper to handle Gzip/JSON decompression
func (h *handler) extractSessionData(c fiber.Ctx) (*auth.GetSession, error) {
	bodyBytes := c.Response().Body()
	if len(bodyBytes) == 0 {
		return nil, errors.New("empty response body")
	}

	var reader io.Reader = bytes.NewReader(bodyBytes)

	// Handle Gzip if the auth service compressed the response
	if string(c.Response().Header.Peek("Content-Encoding")) == "gzip" {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer func(gz *gzip.Reader) {
			err := gz.Close()
			if err != nil {
				log.Errorf("could not close gzip reader: %v", err)
			}
		}(gz)
		reader = gz
	}

	var sessionData auth.GetSession
	if err := json.NewDecoder(reader).Decode(&sessionData); err != nil {
		return nil, err
	}

	return &sessionData, nil
}
