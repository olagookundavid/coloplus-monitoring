package domain

import "time"

// Email is one message to one person.
type Email struct {
	To      string
	Subject string
	Text    string
}

type NotificationStatus string

const (
	NotificationPending NotificationStatus = "pending"
	NotificationSent    NotificationStatus = "sent"
	NotificationFailed  NotificationStatus = "failed"
)

// Notification is an email waiting in, or having left, the outbox.
type Notification struct {
	ID        string
	Kind      string
	Recipient string
	Subject   string
	Body      string
	Status    NotificationStatus
	Attempts  int
	// NextAttemptAt is when the dispatcher may try again.
	NextAttemptAt time.Time
	LastError     string
	ProviderID    string
	// DedupeKey makes enqueueing idempotent: the same key is queued once.
	DedupeKey string
	CreatedAt time.Time
	SentAt    *time.Time
}

// MaxNotificationAttempts is how many times an email is tried before it is
// marked failed and left for a person to see.
const MaxNotificationAttempts = 5

var retryDelays = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour}

// RetryDelay returns how long to wait after the given number of failed
// attempts, and false once the attempts are used up.
func RetryDelay(attempts int) (time.Duration, bool) {
	if attempts >= MaxNotificationAttempts {
		return 0, false
	}
	return retryDelays[attempts-1], true
}

type TokenPurpose string

const (
	PurposeInvite TokenPurpose = "invite"
	PurposeReset  TokenPurpose = "reset"
)

// UserToken is a one-time link target. Only its hash is stored.
type UserToken struct {
	ID        string
	UserID    string
	Purpose   TokenPurpose
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// Usable reports whether the token can still be redeemed.
func (t UserToken) Usable(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}
