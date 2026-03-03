package nowpayments

import (
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"resty.dev/v3"
)

type Service interface {
	GetApiStatus(ctx context.Context) (*StatusResponse, error)
	GetAvailableCurrencies(ctx context.Context) (*CurrenciesResponse, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatus, error)
	CreateNowPayment(ctx context.Context, request Request) (*Response, error)
	GetEstimatedPrice(ctx context.Context, amount float32, currencyFrom, currencyTo string) (*EstimatedPrice, error)
}

type service struct {
	cfg        config.Service
	logger     *slog.Logger
	httpClient *http.Client // Shared transport for efficiency
}

func NewService(cfg config.Service) Service {
	return &service{
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "nowpayments_service"),
		// Standard library transport is thread-safe and reusable
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// newRequest creates a fresh resty request with the CURRENT config values.
// This handles hot-reloading seamlessly.
func (s *service) newRequest() *resty.Request {
	c := s.cfg.Get().NowPayments

	// We wrap the shared httpClient into a new resty instance for this call
	client := resty.NewWithClient(s.httpClient).
		SetBaseURL(c.Endpoint).
		SetHeader("x-api-key", c.APIKey).
		SetHeader("Accept", "application/json").
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)

	return client.R()
}

func (s *service) GetAvailableCurrencies(ctx context.Context) (*CurrenciesResponse, error) {
	l := s.logger.With("op", "GetAvailableCurrencies")
	result := &CurrenciesResponse{}

	resp, err := s.newRequest().
		SetContext(ctx).
		SetResult(result).
		Get("/merchant/coins")

	if err != nil {
		l.Error("network failure", "error", err)
		return nil, fmt.Errorf("provider connection failed: %w", err)
	}

	if resp.IsError() {
		l.Warn("api returned error", "status", resp.StatusCode(), "body", resp.String())
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	return result, nil
}

func (s *service) GetApiStatus(ctx context.Context) (*StatusResponse, error) {
	l := s.logger.With("op", "GetApiStatus")
	result := &StatusResponse{}

	resp, err := s.newRequest().
		SetContext(ctx).
		SetResult(result).
		Get("/status")

	if err != nil {
		l.Error("request failed", "error", err)
		return nil, err
	}

	if resp.IsError() {
		l.Warn("health check failed", "status", resp.StatusCode())
		return nil, fmt.Errorf("provider status unavailable (%d)", resp.StatusCode())
	}

	return result, nil
}

func (s *service) CreateNowPayment(ctx context.Context, req Request) (*Response, error) {
	l := s.logger.With("op", "CreateNowPayment", "order_id", req.OrderID)
	result := &Response{}

	resp, err := s.newRequest().
		SetContext(ctx).
		SetBody(req).
		SetResult(result).
		Post("/payment")

	if err != nil {
		l.Error("order creation failed", "error", err)
		return nil, err
	}

	if resp.IsError() {
		// We log the request (excluding potential secrets) for debugging
		l.Error("provider rejected payment",
			"status", resp.StatusCode(),
			"payload", req,
			"response", resp.String())
		return nil, fmt.Errorf("payment creation rejected (%d)", resp.StatusCode())
	}

	l.Info("payment created successfully", "payment_id", result.PaymentID)
	return result, nil
}

func (s *service) GetEstimatedPrice(ctx context.Context, amount float32, from, to string) (*EstimatedPrice, error) {
	l := s.logger.With("op", "GetEstimatedPrice", "from", from, "to", to, "amount", amount)
	result := &EstimatedPrice{}

	resp, err := s.newRequest().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"amount":        fmt.Sprintf("%.4f", amount),
			"currency_from": from,
			"currency_to":   to,
		}).
		SetResult(result).
		Get("/estimate")

	if err != nil {
		l.Error("estimation request error", "error", err)
		return nil, err
	}

	if resp.IsError() {
		l.Warn("estimation failed", "status", resp.StatusCode())
		return nil, fmt.Errorf("estimation failed (%d)", resp.StatusCode())
	}

	return result, nil
}

func (s *service) GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	l := s.logger.With("op", "GetPaymentStatus", "payment_id", paymentID)

	if paymentID == "" {
		l.Warn("missing payment id")
		return nil, fmt.Errorf("paymentID is required")
	}

	result := &PaymentStatus{}
	resp, err := s.newRequest().
		SetContext(ctx).
		SetResult(result).
		Get(fmt.Sprintf("/payment/%s", paymentID))

	if err != nil {
		l.Error("network error", "error", err)
		return nil, err
	}

	if resp.IsError() {
		l.Warn("payment status fetch failed", "status", resp.StatusCode())
		return nil, fmt.Errorf("failed to fetch status (%d)", resp.StatusCode())
	}

	return result, nil
}
