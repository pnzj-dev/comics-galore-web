package view

import (
	"context"
	"github.com/gofiber/fiber/v3/log"
)

// Use a private type to avoid collisions
type contextKey string

const appContextKey contextKey = "app_context"

// WithAppContext adds the AppContext struct to the context
func WithAppContext(ctx context.Context, s *AppContext) context.Context {
	return context.WithValue(ctx, appContextKey, s)
}

func GetAppContext2(ctx context.Context) *AppContext {
	val := ctx.Value(appContextKey)

	// 1. Check if the key exists at all
	if val == nil {
		log.Warn("GetAppContext2: Key 'app_context' is MISSING from context. Ensure WithAppContext was called.")
		return &AppContext{UserInfo: nil}
	}

	// 2. Check for type mismatch (The most common cause of 'ok' being false)
	s, ok := val.(*AppContext)
	if !ok {
		// This will print exactly what Go found instead of what you expected
		log.Errorf("GetAppContext2: Type mismatch! Expected *view.AppContext, but got %T", val)
		return &AppContext{UserInfo: nil}
	}

	// 3. Optional: Log successful retrieval for heavy debugging
	// log.Debug("GetAppContext: Successfully retrieved AppContext")

	return s
}
