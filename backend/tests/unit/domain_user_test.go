package unit

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func TestUser_LockoutAfterRepeatedFailures(t *testing.T) {
	now := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	u := domain.User{}

	for i := 1; i < domain.MaxFailedLogins; i++ {
		failed, locked := u.AfterFailedLogin(now)
		if locked != nil || failed != i {
			t.Fatalf("attempt %d: failed=%d locked=%v", i, failed, locked)
		}
		u.FailedLogins = failed
	}
	failed, locked := u.AfterFailedLogin(now)
	if locked == nil || failed != 0 {
		t.Fatalf("attempt %d must lock the account: failed=%d locked=%v", domain.MaxFailedLogins, failed, locked)
	}
	u.LockedUntil = locked
	if !u.IsLocked(now.Add(domain.LockDuration-time.Second)) || u.IsLocked(now.Add(domain.LockDuration+time.Second)) {
		t.Fatal("the lock must hold for exactly the lock duration")
	}
}
