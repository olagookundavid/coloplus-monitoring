package integration

import (
	"context"
	"testing"

	"coloplus-backend/internal/domain"
)

// Old emails lose their address and text but keep their row and dedupe key, so
// a job that runs again can never email the same reason twice; a dry run
// changes nothing.
func TestRetention_BlanksOldEmailsButKeepsTheDedupeKey(t *testing.T) {
	s := newStack(t)
	key := uniq("retention-key")
	s.exec(`INSERT INTO notifications (kind, recipient, subject, body, status, dedupe_key, created_at)
	        VALUES ('test', 'someone@example.com', 'Subject', '', 'sent', $1, now() - interval '200 days')`, key)
	ctx := context.Background()

	n, err := s.notifications.ApplyRetention(ctx, 90, true)
	if err != nil || n < 1 {
		t.Fatalf("a dry run should count the old email: n=%d err=%v", n, err)
	}
	var recipient string
	read := func() (string, string) {
		var r, subj string
		if err := s.pool.QueryRow(ctx, `SELECT recipient, subject FROM notifications WHERE dedupe_key = $1`, key).Scan(&r, &subj); err != nil {
			t.Fatal(err)
		}
		return r, subj
	}
	if recipient, _ = read(); recipient == "" {
		t.Fatal("a dry run must not change anything")
	}

	if _, err := s.notifications.ApplyRetention(ctx, 90, false); err != nil {
		t.Fatal(err)
	}
	if recipient, subj := read(); recipient != "" || subj != "" {
		t.Fatalf("the address and subject should be blanked: %q %q", recipient, subj)
	}
	// The same reason is still known, so it is not queued again.
	queued, err := s.notifications.Enqueue(ctx, domainNotification(key))
	if err != nil || queued {
		t.Fatalf("the dedupe key must outlive the text: queued=%v err=%v", queued, err)
	}
}

func domainNotification(key string) domain.Notification {
	return domain.Notification{Kind: "test", Recipient: "someone@example.com", Subject: "Subject", DedupeKey: key}
}
