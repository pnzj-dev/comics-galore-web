package email

import (
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Service interface {
	Send(ctx context.Context, req Request) error
	SendWithTemplate(ctx context.Context, req TemplateRequest) error
}

type service struct {
	cfg       config.Service
	logger    *slog.Logger
	fromName  string
	fromEmail string
}

type Option func(*service)

func WithFromAddress(name, email string) Option {
	return func(s *service) {
		s.fromName = name
		s.fromEmail = email
	}
}

func NewService(cfg config.Service, opts ...Option) Service {
	s := &service{
		cfg:       cfg,
		logger:    cfg.GetLogger().With("component", "email_service"),
		fromName:  "System",
		fromEmail: "noreply@yourdomain.com",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *service) Send(ctx context.Context, req Request) error {
	l := s.logger.With("op", "Send", "to_email", req.ToEmail, "subject", req.Subject)

	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(req.ToName, req.ToEmail)
	message := mail.NewSingleEmail(from, req.Subject, to, req.PlainText, req.HTMLContent)

	return s.send(ctx, l, message)
}

func (s *service) SendWithTemplate(ctx context.Context, req TemplateRequest) error {
	l := s.logger.With("op", "SendWithTemplate", "to_email", req.ToEmail, "template_id", req.TemplateID)

	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(req.ToName, req.ToEmail)

	p := mail.NewPersonalization()
	p.AddTos(to)

	for k, v := range req.TemplateData {
		p.SetDynamicTemplateData(k, v)
	}

	m := mail.NewV3Mail()
	m.SetFrom(from)
	m.SetTemplateID(req.TemplateID)
	m.AddPersonalizations(p)

	return s.send(ctx, l, m)
}

func (s *service) send(ctx context.Context, l *slog.Logger, message *mail.SGMailV3) error {
	apiKey := s.cfg.Get().SendGrid.APIKey
	client := sendgrid.NewSendClient(apiKey)

	response, err := client.SendWithContext(ctx, message)
	if err != nil {
		l.Error("sendgrid network/client error", "error", err)
		return fmt.Errorf("sendgrid execution failed: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		l.Error("sendgrid api rejected email",
			"status_code", response.StatusCode,
			"response_body", response.Body)
		return fmt.Errorf("sendgrid api error: status %d", response.StatusCode)
	}

	l.Info("email sent successfully", "status_code", response.StatusCode)
	return nil
}
