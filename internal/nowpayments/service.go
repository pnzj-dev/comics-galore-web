package nowpayments

import (
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v3/client"
)

type service struct {
	cfg    config.Service
	logger *slog.Logger
}

type Service interface {
	GetApiStatus(ctx context.Context) (*StatusResponse, error)
	GetAvailableCurrencies(ctx context.Context) (*CurrenciesResponse, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatus, error)
	CreateNowPayment(ctx context.Context, request Request) (*Response, error)
	GetEstimatedPrice(ctx context.Context, amount float32, currencyFrom, currencyTo string) (*EstimatedPrice, error)
}

func isSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func NewService(cfg config.Service) Service {
	return &service{
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "nowpayments_service"),
	}
}

func (s *service) newClient() *client.Client {
	return client.New().SetBaseURL(s.cfg.Get().NowPayments.Endpoint).AddHeaders(map[string][]string{
		"Accept":    {"application/json"},
		"x-api-key": {s.cfg.Get().NowPayments.APIKey},
	})
}

func (s *service) GetAvailableCurrencies(ctx context.Context) (*CurrenciesResponse, error) {
	l := s.logger.With("op", "GetAvailableCurrencies")
	result := &CurrenciesResponse{}

	resp, err := s.newClient().Get("/merchant/coins", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("network failure", "error", err)
		return nil, fmt.Errorf("network failure: %w", err)
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("provider rejected request",
			"status", resp.StatusCode(),
			"body", string(resp.Body()))
		return nil, fmt.Errorf("provider error (%d)", resp.StatusCode())
	}

	if err := resp.JSON(result); err != nil {
		l.Error("failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return result, nil
}

func (s *service) GetApiStatus(ctx context.Context) (*StatusResponse, error) {
	l := s.logger.With("op", "GetApiStatus")
	result := &StatusResponse{}

	resp, err := s.newClient().Get("/status", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Warn("health check failed", "status", resp.StatusCode())
		return nil, fmt.Errorf("provider status unavailable (%d)", resp.StatusCode())
	}

	if err := resp.JSON(result); err != nil {
		l.Error("json parse error", "error", err)
		return nil, err
	}

	return result, nil
}

func (s *service) CreateNowPayment(ctx context.Context, req Request) (*Response, error) {
	l := s.logger.With("op", "CreateNowPayment", "order_id", req.OrderID)
	result := &Response{}

	l.Debug("initiating payment request", "price", req.PriceAmount, "currency", req.PriceCurrency)

	resp, err := s.newClient().Post("/payment", client.Config{Ctx: ctx, Body: req})
	if err != nil {
		l.Error("transport error", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("payment creation rejected",
			"status", resp.StatusCode(),
			"response_body", string(resp.Body()))
		return nil, fmt.Errorf("provider error (%d)", resp.StatusCode())
	}

	if err := resp.JSON(result); err != nil {
		l.Error("failed to decode payment response", "error", err)
		return nil, err
	}

	l.Info("payment created", "payment_id", result.PaymentID)
	return result, nil
}

func (s *service) GetEstimatedPrice(ctx context.Context, amount float32, from, to string) (*EstimatedPrice, error) {
	l := s.logger.With("op", "GetEstimatedPrice", "from", from, "to", to)
	result := &EstimatedPrice{}

	queryParams := map[string][]string{
		"amount":        {fmt.Sprintf("%.4f", amount)},
		"currency_from": {from},
		"currency_to":   {to},
	}

	resp, err := s.newClient().AddParams(queryParams).Get("/estimate", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("estimation request failed", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Warn("estimation rejected", "status", resp.StatusCode(), "body", string(resp.Body()))
		return nil, fmt.Errorf("estimation failed (%d)", resp.StatusCode())
	}

	if err := resp.JSON(result); err != nil {
		l.Error("failed to parse estimation", "error", err)
		return nil, err
	}

	return result, nil
}

func (s *service) GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	l := s.logger.With("op", "GetPaymentStatus", "payment_id", paymentID)

	if paymentID == "" {
		l.Warn("attempted status check with empty payment_id")
		return nil, fmt.Errorf("paymentID is required")
	}

	result := &PaymentStatus{}

	resp, err := s.newClient().Get(fmt.Sprintf("/payment/%s", paymentID), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("network failure on status check", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("status check rejected", "status", resp.StatusCode())
		return nil, fmt.Errorf("failed to fetch status (%d)", resp.StatusCode())
	}

	if err := resp.JSON(result); err != nil {
		l.Error("status parse error", "error", err)
		return nil, err
	}

	return result, nil
}
