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

func (h *handler) getCookieName() string {
	if h.cfg.Get().AppEnv == "production" {
		return "__Secure-cg-auth"
	}
	return "cg-auth-dev"
}

func (h *handler) createCookie(c fiber.Ctx) error {
	if err := c.Next(); err != nil {
		return err
	}

	// 1. Get the raw data from the proxy response body
	userSession, err := h.getUserSession(c)
	if err != nil {
		return nil
	}

	// 2. Transform the struct into a signed JWT string
	signedToken, err := h.generateJWT(userSession)
	if err != nil {
		log.Errorf("JWT generation failed: %v", err)
		return nil
	}

	// 3. Clean up headers from the proxy
	c.Response().Header.Del(fiber.HeaderSetCookie)

	// 4. Set the cookie
	c.Cookie(&fiber.Cookie{
		Name:     "cg-auth-local", //TODO: __Secure-cg-auth (in production) => use getCookieName()
		Value:    signedToken,
		Expires:  userSession.Session.ExpiresAt,
		HTTPOnly: true,
		Secure:   false, //TODO: set to true in production
		SameSite: "Lax",
		Path:     "/",
	})

	return nil
}

func (h *handler) getUserSession(c fiber.Ctx) (*auth.UserSession, error) {
	bodyBytes := c.Response().Body()
	if len(bodyBytes) == 0 {
		return nil, errors.New("empty response body")
	}

	var reader io.Reader = bytes.NewReader(bodyBytes)

	// Check for Gzip encoding from the proxy response
	if string(c.Response().Header.Peek(fiber.HeaderContentEncoding)) == "gzip" {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip reader failed: %w", err)
		}
		defer func(gz *gzip.Reader) {
			err := gz.Close()
			if err != nil {
				log.Errorf("could not close gzip reader: %v", err)
			}
		}(gz)
		reader = gz
	}

	var userSession auth.UserSession
	if err := json.NewDecoder(reader).Decode(&userSession); err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	// Clear the body from the response
	c.Response().ResetBody()

	return &userSession, nil
}
