package qrcode

import (
	"bytes"
	"comics-galore-web/internal/config"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// bufferCloser wraps bytes.Buffer to satisfy io.WriteCloser without side effects on Close
type bufferCloser struct {
	*bytes.Buffer
}

func (b bufferCloser) Close() error { return nil }

type Service interface {
	GeneratePNG(dataType string, params map[string]string) ([]byte, error)
	IsValidIBAN(iban string) bool
}

type service struct {
	ibanRegex *regexp.Regexp
	logger    *slog.Logger
}

func NewService(cfg config.Service) Service {
	return &service{
		ibanRegex: regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{4,30}$`),
		logger:    cfg.GetLogger().With("component", "qrcode_service"),
	}
}

func (s *service) IsValidIBAN(iban string) bool {
	iban = strings.ReplaceAll(iban, " ", "")
	if !s.ibanRegex.MatchString(iban) {
		return false
	}

	// 1. Move first 4 chars to the end
	rearranged := iban[4:] + iban[:4]

	// 2. Convert letters to numbers
	var sb strings.Builder
	for _, char := range rearranged {
		if char >= 'A' && char <= 'Z' {
			sb.WriteString(fmt.Sprintf("%d", char-'A'+10))
		} else {
			sb.WriteRune(char)
		}
	}

	// 3. Modulo 97 check
	ibanInt, ok := new(big.Int).SetString(sb.String(), 10)
	if !ok {
		return false
	}

	return ibanInt.Mod(ibanInt, big.NewInt(97)).Int64() == 1
}

func (s *service) validate(dataType string, params map[string]string) error {
	switch dataType {
	case "epc_payment":
		if !s.IsValidIBAN(params["iban"]) {
			return fmt.Errorf("invalid IBAN format")
		}
		if params["bic"] == "" || params["name"] == "" {
			return fmt.Errorf("missing mandatory SEPA fields (BIC/Name)")
		}
	case "bitcoin", "ethereum", "solana", "ripple", "flare":
		if params["address"] == "" {
			return fmt.Errorf("crypto address is required for %s", dataType)
		}
	}
	return nil
}

func (s *service) GeneratePNG(dataType string, params map[string]string) ([]byte, error) {
	l := s.logger.With("op", "GeneratePNG", "data_type", dataType)

	if err := s.validate(dataType, params); err != nil {
		l.Warn("validation failed", "error", err)
		return nil, err
	}

	input := s.formatInput(dataType, params)

	qrc, err := qrcode.New(input)
	if err != nil {
		l.Error("failed to create QR matrix", "error", err)
		return nil, fmt.Errorf("matrix creation failed: %w", err)
	}

	buf := new(bytes.Buffer)
	// Wrap buffer to satisfy the writer interface requirement of the library
	wr := bufferCloser{Buffer: buf}

	// standard.NewWithWriter expects an io.WriteCloser
	w := standard.NewWithWriter(wr,
		standard.WithQRWidth(20),
		standard.WithFgColorRGBHex("#000000"),
		standard.WithBgColorRGBHex("#ffffff"),
	)

	if err = qrc.Save(w); err != nil {
		l.Error("failed to render QR to buffer", "error", err)
		return nil, fmt.Errorf("render failed: %w", err)
	}

	l.Debug("QR code generated", "input_len", len(input), "output_size", buf.Len())
	return buf.Bytes(), nil
}

func (s *service) formatInput(dataType string, params map[string]string) string {
	switch dataType {
	case "bitcoin", "solana", "flare":
		u := url.URL{
			Scheme: dataType,
			Path:   params["address"],
		}
		q := u.Query()
		if amt, ok := params["amount"]; ok {
			q.Set("amount", amt)
		}
		u.RawQuery = q.Encode()
		return u.String()

	case "ethereum":
		u := url.URL{Scheme: "ethereum", Path: params["address"]}
		q := u.Query()
		if val, ok := params["amount"]; ok {
			q.Set("value", val)
		}
		u.RawQuery = q.Encode()
		return u.String()

	case "ripple", "xrp":
		u := url.URL{Scheme: "ripple", Path: params["address"]}
		q := u.Query()
		if amt, ok := params["amount"]; ok {
			q.Set("amount", amt)
		}
		if tag, ok := params["tag"]; ok {
			q.Set("dt", tag)
		}
		u.RawQuery = q.Encode()
		return u.String()

	case "epc_payment":
		// Format: BCD\n{version}\n{encoding}\n{identification}\n{bic}\n{name}\n{iban}\n{amount}\n{reason}
		return fmt.Sprintf("BCD\n002\n1\nSCT\n%s\n%s\n%s\nEUR%s\n\n%s",
			params["bic"], params["name"], params["iban"], params["amount"], params["reason"])

	case "wifi":
		return fmt.Sprintf("WIFI:S:%s;T:%s;P:%s;;", params["ssid"], params["auth"], params["pass"])

	case "vcard":
		return fmt.Sprintf("BEGIN:VCARD\nVERSION:3.0\nFN:%s\nTEL:%s\nEMAIL:%s\nEND:VCARD",
			params["name"], params["tel"], params["email"])
	case "geo":
		return fmt.Sprintf("geo:%s,%s", params["lat"], params["lng"])
	case "sms":
		return fmt.Sprintf("SMSTO:%s:%s", params["number"], params["msg"])
	default:
		return params["data"]
	}
}
