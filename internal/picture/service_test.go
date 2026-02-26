package picture

import (
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/server"
	"errors"
	"gotest.tools/v3/assert"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

func TestImageHandler_S3Error(t *testing.T) {
	// 1. Create the mock with a specific failure
	mockPic := &MockService{
		ProcessAndCacheFunc: func(key string, width, quality int) (io.ReadCloser, error) {
			return nil, errors.New("s3 connection timeout")
		},
	}

	// 2. Inject the mock into your server deps
	deps := server.Deps{
		Config:  config.NewMockService(&config.Env{}),
		Logger:  slog.Default(),
		Picture: mockPic,
	}
	srv := deps.New()
	srv.RegisterFiberRoutes()

	// 3. Perform the request
	req := httptest.NewRequest("GET", "/api/v1/image/test.jpg", nil)
	resp, _ := srv.App.Test(req)

	// 4. Assert that the server returns a 500 status code correctly
	assert.Equal(t, 500, resp.StatusCode)
}
