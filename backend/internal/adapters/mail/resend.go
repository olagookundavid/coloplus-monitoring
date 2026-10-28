// Package mail holds the email providers behind ports.Mailer.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
)

const defaultResendURL = "https://api.resend.com"

// ResendConfig configures the Resend adapter.
type ResendConfig struct {
	APIKey string
	// From is the sender, e.g. "Coloplus <alerts@example.com>". The sending
	// domain must be verified in Resend.
	From    string
	BaseURL string
	// RatePerSecond paces sends below Resend's account limit.
	RatePerSecond int
	Timeout       time.Duration
}

// Resend sends through the Resend HTTP API. It owns one HTTP client and
// paces calls, and it makes a single attempt per call: retrying belongs to
// the outbox, which can wait minutes and survives a restart.
type Resend struct {
	cfg      ResendConfig
	client   *http.Client
	interval time.Duration

	mu   sync.Mutex
	last time.Time
}

func NewResend(cfg ResendConfig) (*Resend, error) {
	switch {
	case cfg.APIKey == "":
		return nil, errors.New("resend: API key is required")
	case cfg.From == "":
		return nil, errors.New("resend: sender address is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultResendURL
	}
	if cfg.RatePerSecond < 1 {
		cfg.RatePerSecond = 8
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Resend{
		cfg:      cfg,
		client:   &http.Client{Timeout: cfg.Timeout},
		interval: time.Second / time.Duration(cfg.RatePerSecond),
	}, nil
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

// pace waits until the next send is allowed.
func (r *Resend) pace(ctx context.Context) error {
	r.mu.Lock()
	wait := time.Until(r.last.Add(r.interval))
	if wait < 0 {
		wait = 0
	}
	r.last = time.Now().Add(wait)
	r.mu.Unlock()

	if wait == 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Resend) Send(ctx context.Context, mail domain.Email, idempotencyKey string) (string, error) {
	if err := r.pace(ctx); err != nil {
		return "", fmt.Errorf("%w: waiting to send: %v", apperrors.ErrUnavailable, err)
	}
	body, err := json.Marshal(resendRequest{From: r.cfg.From, To: []string{mail.To}, Subject: mail.Subject, Text: mail.Text})
	if err != nil {
		return "", fmt.Errorf("encoding email: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.cfg.BaseURL, "/")+"/emails", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: calling Resend: %v", apperrors.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		var ok struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &ok)
		return ok.ID, nil
	case resp.StatusCode == http.StatusTooManyRequests, resp.StatusCode == http.StatusConflict, resp.StatusCode >= 500:
		return "", fmt.Errorf("%w: Resend answered %d: %s", apperrors.ErrUnavailable, resp.StatusCode, snippet(raw))
	default:
		// 400, 401, 403, 422: the message or our credentials are wrong.
		// Retrying cannot help until someone fixes it.
		return "", fmt.Errorf("%w: Resend answered %d: %s", apperrors.ErrMailRejected, resp.StatusCode, snippet(raw))
	}
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
