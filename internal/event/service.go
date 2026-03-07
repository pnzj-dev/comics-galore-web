package event

import (
	"bytes"
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"net/http"
)

type Service interface {
	TrackEvent(ctx context.Context, eventType string) error
}

type service struct {
	cfg config.Service
}

func (s service) TrackEvent(ctx context.Context, eventType string) error {
	// You make a light POST request to your Durable Object
	// This doesn't block the user's main request
	go func() {
		_, err := http.Post(
			fmt.Sprintf("%s/admin/api/track-event", s.cfg.Get().BetterAuth),
			"application/json",
			bytes.NewBufferString(`{"type":"`+eventType+`"}`),
		)
		if err != nil {
			s.cfg.GetLogger().Error("failed to sync metric to DO", "error", err)
		}
	}()
	return nil
}

func NewService(cfg config.Service) Service {
	return &service{
		cfg: cfg,
	}
}
