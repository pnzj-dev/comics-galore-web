package picture

import (
	"io"
	"strings"
	"time"
)

// MockService implements the Service interface for testing.
type MockService struct {
	// We use function fields so we can change behavior per test case.
	ProcessAndCacheFunc func(key string, width, quality int) (io.ReadCloser, error)
	ShutdownFunc        func() error
}

func (m *MockService) ProcessAndCacheFromS3(key string, width, quality int) (io.ReadCloser, error) {
	if m.ProcessAndCacheFunc != nil {
		return m.ProcessAndCacheFunc(key, width, quality)
	}
	// Default behavior: return a dummy reader
	return io.NopCloser(strings.NewReader("fake-image-data")), nil
}

func (m *MockService) Shutdown(timeout time.Duration) error {
	if m.ShutdownFunc != nil {
		return m.ShutdownFunc()
	}
	return nil
}
