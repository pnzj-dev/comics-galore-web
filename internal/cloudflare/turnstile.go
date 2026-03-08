package cloudflare

import (
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/errors"
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3/client"
)

type turnstile struct {
	cfg    config.Service
	client *client.Client
}

type Turnstile interface {
	Verify(ctx context.Context, token, secretKey, remoteIP string) (*TurnstileResponse, error)
}

func NewTurnstile(cfg config.Service) Turnstile {
	cc := client.New().
		SetTimeout(10 * time.Second).
		SetUserAgent("comics-galore/1.0") // optional but nice

	return &turnstile{cfg: cfg, client: cc}
}

func (t *turnstile) Verify(ctx context.Context, token, secretKey, remoteIP string) (*TurnstileResponse, error) {
	logger := t.cfg.GetLogger().With(
		"op", "Turnstile.Verify",
		"token_prefix", safeTokenPrefix(token), // helper to avoid logging full token
		"remote_ip", remoteIP,
		"turnstile_url", t.cfg.Get().Cloudflare.TurnstileURL,
	)

	if token == "" {
		logger.Warn("turnstile token is empty → verification rejected early")
		return nil, errors.New("TURNSTILE_INVALID_TOKEN", "turnstile token is required", 400)
	}

	if secretKey == "" {
		logger.Error("turnstile secret key is empty → cannot perform verification")
		return nil, errors.New("CONFIGURATION_ERROR", "turnstile secret key is missing", 500)
	}

	form := map[string]string{
		"secret":   secretKey,
		"response": token,
	}

	if remoteIP != "" {
		form["remoteip"] = remoteIP
	}

	logger.Debug("remoteip_included", fmt.Sprintf("%t", remoteIP != ""), "message", "sending turnstile verification request")

	start := time.Now() // ← Measure duration manually

	resp, err := t.client.Post(t.cfg.Get().Cloudflare.TurnstileURL, client.Config{
		Ctx:      ctx,
		FormData: form,
	})

	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		logger.Error("error", err, "message", "turnstile HTTP request failed (network / timeout / dns / etc)")
		return nil, errors.New("TURNSTILE_NETWORK_ERROR", err.Error(), 500)
	}

	log := logger.With(
		"status_code", resp.StatusCode(),
		"response_duration_ms", durationMs,
	)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		bodyPreview := safeBodyPreview(resp.Body())
		log.Error("response_body_preview", bodyPreview, "message", "turnstile returned non-success status → verification failed")
		return nil, errors.New("TURNSTILE_HTTP_ERROR", resp.Status(), resp.StatusCode())
	}

	var tsResp TurnstileResponse
	if err := resp.JSON(&tsResp); err != nil {
		bodyPreview := safeBodyPreview(resp.Body())
		log.Error("error", err.Error(), "response_body_preview", bodyPreview, "message", "failed to parse turnstile JSON response")
		return nil, errors.New("TURNSTILE_INVALID_FORMAT", err.Error(), 500)
	}

	if !tsResp.Success {
		log.Warn("error_codes", tsResp.ErrorCodes, "hostname", tsResp.Hostname, "challenge_ts", tsResp.ChallengeTS, "message", "turnstile verification failed - token rejected by Cloudflare")

		code := "turnstile_verification_failed"
		msg := "Cloudflare Turnstile rejected the token"
		if len(tsResp.ErrorCodes) > 0 {
			msg += fmt.Sprintf(" (codes: %v)", tsResp.ErrorCodes)
		}
		return nil, errors.New(code, msg, 403)
	}

	// Success path — include useful context
	log.Debug("hostname", tsResp.Hostname, "challenge_ts", tsResp.ChallengeTS, "error_codes", tsResp.ErrorCodes, "message", "turnstile verification succeeded")

	return &tsResp, nil
}

// ────────────────────────────────────────────────
// Small helpers — put them in the same file or utils
// ────────────────────────────────────────────────

func safeTokenPrefix(token string) string {
	if len(token) <= 12 {
		return "[short]"
	}
	return token[:12] + "..."
}

func safeBodyPreview(body []byte) string {
	const maxPreview = 300
	s := string(body)
	if len(s) > maxPreview {
		return s[:maxPreview] + "... [truncated]"
	}
	return s
}
