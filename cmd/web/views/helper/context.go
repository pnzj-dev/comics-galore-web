package helper

import (
	"comics-galore-web/internal/view"
	"context"
)

// Use a private type to avoid collisions
type contextKey string

const appContextKey contextKey = "app_context"

// WithAppContext adds the AppContext struct to the context
func WithAppContext(ctx context.Context, s *view.AppContext) context.Context {
	return context.WithValue(ctx, appContextKey, *s)
}

// GetAppContext retrieves the AppContext struct from the context
func GetAppContext(ctx context.Context) view.AppContext {
	if s, ok := ctx.Value(appContextKey).(view.AppContext); ok {
		return s
	}
	return view.AppContext{}
}
