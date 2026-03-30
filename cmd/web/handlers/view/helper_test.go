package view

import (
	"context"
	"testing"
)

func TestWithAppContext(t *testing.T) {
	// 1. Setup a base context and a sample AppContext pointer
	bgCtx := context.Background()
	expected := &AppContext{
		Title: "Test Comic Store",
		UserInfo: &ComicsGaloreClaims{
			UserID: "user_123",
			Email:  "test@example.com",
		},
	}

	// 2. Execute the function under test
	ctx := WithAppContext(bgCtx, expected)

	// 3. Verify the value exists in the context using the internal key
	val := ctx.Value(appContextKey)
	if val == nil {
		t.Fatal("expected AppContext to be present in context, but got nil")
	}

	// 4. Verify the type and pointer identity
	actual, ok := val.(*AppContext)
	if !ok {
		t.Errorf("expected type *AppContext, but got %T", val)
	}

	if actual != expected {
		t.Errorf("pointer mismatch: expected %p, got %p", expected, actual)
	}

	if actual.Title != expected.Title {
		t.Errorf("data mismatch: expected title %s, got %s", expected.Title, actual.Title)
	}
}

func TestWithAppContext_Nil(t *testing.T) {
	// Ensure the function handles a nil AppContext pointer gracefully
	bgCtx := context.Background()
	ctx := WithAppContext(bgCtx, nil)

	val := ctx.Value(appContextKey)
	if val != nil && val.(*AppContext) != nil {
		t.Error("expected context value to represent a nil pointer")
	}
}
