package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

type Role string

const (
	RoleAdmin        Role = "admin"
	RoleReporter     Role = "reporter"
	RoleSupervisor   Role = "supervisor"
	RoleAirtelViewer Role = "airtel_viewer"
)

// ValidRole reports whether r is a role the system knows.
func ValidRole(r Role) bool {
	switch r {
	case RoleAdmin, RoleReporter, RoleSupervisor, RoleAirtelViewer:
		return true
	}
	return false
}

// Permission is what the code checks. Roles map to permission sets in the
// database; code never branches on a role name.
type Permission string

const (
	PermUsersManage   Permission = "users.manage"
	PermSitesAll      Permission = "sites.all"
	PermSitesRead     Permission = "sites.read"
	PermSitesManage   Permission = "sites.manage"
	PermReportsCreate Permission = "reports.create"
	PermReportsRead   Permission = "reports.read"
	PermReportsReview Permission = "reports.review"
	PermPublishedRead Permission = "published.read"
	PermAuditRead     Permission = "audit.read"

	PermIncidentsCreate Permission = "incidents.create"
	PermIncidentsRead   Permission = "incidents.read"
	PermIncidentsManage Permission = "incidents.manage"
)

const (
	MaxFailedLogins = 5
	LockDuration    = 15 * time.Minute
	minPasswordLen  = 12
	// bcrypt ignores everything past 72 bytes, so longer passwords are refused
	// rather than silently truncated.
	maxPasswordLen = 72
)

type User struct {
	ID           string
	Email        string
	FullName     string
	PasswordHash string
	Role         Role
	Active       bool

	// TOTPSecret is held sealed (encrypted) and only opened to verify a code.
	TOTPSecret  string
	TOTPEnabled bool

	FailedLogins int
	LockedUntil  *time.Time
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsLocked reports whether sign-in is blocked after repeated failures.
func (u User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// AfterFailedLogin returns the new failure count and, when the limit is
// reached, the time the account unlocks.
func (u User) AfterFailedLogin(now time.Time) (int, *time.Time) {
	failed := u.FailedLogins + 1
	if failed < MaxFailedLogins {
		return failed, nil
	}
	until := now.Add(LockDuration)
	return 0, &until
}

// Actor is the signed-in person as the services see them: who they are, what
// they may do, and which sites they may touch.
type Actor struct {
	UserID      string
	Role        Role
	Permissions map[Permission]struct{}
	AllSites    bool
	SiteIDs     map[string]struct{}
}

func (a Actor) Can(p Permission) bool {
	_, ok := a.Permissions[p]
	return ok
}

// CanAccessSite is the site-scope check applied on every read and write.
func (a Actor) CanAccessSite(siteID string) bool {
	if a.AllSites {
		return true
	}
	_, ok := a.SiteIDs[siteID]
	return ok
}

// SiteIDList returns the assigned site ids; meaningless when AllSites is set.
func (a Actor) SiteIDList() []string {
	ids := make([]string, 0, len(a.SiteIDs))
	for id := range a.SiteIDs {
		ids = append(ids, id)
	}
	return ids
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return errors.New("enter a valid email address")
	}
	return nil
}

// ValidatePassword enforces the password policy.
func ValidatePassword(pw string) error {
	if len(pw) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	if len(pw) > maxPasswordLen {
		return fmt.Errorf("password must be at most %d characters", maxPasswordLen)
	}
	var letter, digit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return errors.New("password must contain at least one letter and one number")
	}
	return nil
}
