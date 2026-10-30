package mail

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"coloplus-backend/internal/domain"
)

// Log writes each email to the log instead of sending it. It is for local
// development, where it is the only way to see an invitation link. The config
// refuses it in production.
type Log struct {
	log *slog.Logger
	n   atomic.Int64
}

func NewLog(log *slog.Logger) *Log { return &Log{log: log} }

func (l *Log) Send(_ context.Context, mail domain.Email, _ string) (string, error) {
	id := l.n.Add(1)
	l.log.Info("email (not sent: MAIL_PROVIDER=log)", "to", mail.To, "subject", mail.Subject, "text", mail.Text)
	return fmt.Sprintf("log-%d", id), nil
}
