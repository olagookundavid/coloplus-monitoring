// Package ports holds the interfaces services depend on. Adapters implement
// them; nothing here imports an adapter, a driver or a vendor SDK.
package ports

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
)

// Clock is injected wherever the current time matters, so rules can be tested.
type Clock interface {
	Now() time.Time
}

// TxManager runs fn inside one database transaction. The context handed to fn
// carries the transaction, so repositories called with it join the same one.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AuditStore persists audit events. Insert joins a transaction carried by the
// context; InsertBatch is the bulk path used by the batcher.
type AuditStore interface {
	Insert(ctx context.Context, event domain.AuditEvent) error
	InsertBatch(ctx context.Context, events []domain.AuditEvent) error
	// Search returns events newest first, with the total that match.
	Search(ctx context.Context, f AuditFilter) (events []domain.AuditEvent, total int, err error)
}

// AuditFilter narrows an audit search. When AllSites is false only events
// that carry one of SiteIDs in their metadata are returned, so a reader
// scoped to some sites never sees the rest.
type AuditFilter struct {
	AllSites   bool
	SiteIDs    []string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	From, To   *time.Time
	Limit      int
	Offset     int
}

// AuditQueue buffers audit events for batched writes. Add reports false when
// the event was not accepted.
type AuditQueue interface {
	Add(event domain.AuditEvent) bool
}

// WorkerPool runs background tasks (email sends, exports) off the request path.
type WorkerPool interface {
	Submit(task func()) error
}

// Pinger checks that a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PasswordHasher hashes and checks passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(hash, password string) bool
}

// TokenIssuer creates access tokens for signed-in users.
type TokenIssuer interface {
	Issue(userID string, ttl time.Duration) (token string, expiresAt time.Time, err error)
}

// TOTP generates and checks authenticator-app codes.
type TOTP interface {
	NewSecret() (string, error)
	Verify(secret, code string, now time.Time) bool
	URL(account, secret string) string
}

// SecretBox encrypts small secrets (such as TOTP seeds) for storage.
type SecretBox interface {
	Seal(plaintext string) (string, error)
	Open(sealed string) (string, error)
}

// UserRepository stores users, their permissions and their site assignments.
// Methods return apperrors.ErrNotFound or ErrConflict where they apply.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	// Update changes name, role, active flag and password hash.
	Update(ctx context.Context, u domain.User) error
	RecordLoginSuccess(ctx context.Context, id string, at time.Time) error
	RecordLoginFailure(ctx context.Context, id string, failed int, lockedUntil *time.Time) error
	SetTOTP(ctx context.Context, id, sealedSecret string, enabled bool) error
	// SetPassword stores a new hash, clears lockout, and ends older sessions.
	SetPassword(ctx context.Context, id, hash string, at time.Time) error
	// UsersWithPermission returns active users whose role holds the
	// permission and who can see the site (all sites, or assigned to it).
	UsersWithPermission(ctx context.Context, perm domain.Permission, siteID string) ([]domain.User, error)

	PermissionsForRole(ctx context.Context, role domain.Role) ([]domain.Permission, error)
	SiteIDsForUser(ctx context.Context, userID string) ([]string, error)
	SetUserSites(ctx context.Context, userID string, siteIDs []string) error
}

// SiteFilter narrows a site listing. SiteIDs applies only when AllSites is
// false, so a person never sees a site outside their assignment.
type SiteFilter struct {
	AllSites bool
	SiteIDs  []string
	Query    string
	State    string
	Region   string
	Status   domain.SiteStatus
	Limit    int
	Offset   int
}

type SiteRepository interface {
	Create(ctx context.Context, s domain.Site) (domain.Site, error)
	GetByID(ctx context.Context, id string) (domain.Site, error)
	Update(ctx context.Context, s domain.Site) error
	List(ctx context.Context, f SiteFilter) (sites []domain.Site, total int, err error)
	// GetMany returns the sites with the given ids, skipping unknown ones.
	GetMany(ctx context.Context, ids []string) ([]domain.Site, error)
	// CountExisting reports how many of the ids are real sites.
	CountExisting(ctx context.Context, ids []string) (int, error)
}

// ReportFilter narrows a report listing. Scope works as in SiteFilter.
type ReportFilter struct {
	AllSites bool
	SiteIDs  []string
	SiteID   string
	From     *time.Time
	To       *time.Time
	Statuses []domain.ReportStatus
	Limit    int
	Offset   int
}

type ReportRepository interface {
	// Create stores a new report with its intervals. ErrConflict when one is
	// already in progress or approved for that site and day.
	Create(ctx context.Context, r domain.DailyReport) (domain.DailyReport, error)
	// GetByID loads a report with its intervals. forUpdate locks the row for
	// the surrounding transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.DailyReport, error)
	// SaveDraft replaces notes and intervals and sets the status back to
	// draft. ErrConflict when rowVersion is stale.
	SaveDraft(ctx context.Context, r domain.DailyReport, rowVersion int) (domain.DailyReport, error)
	MarkSubmitted(ctx context.Context, id string, at time.Time) error
	// MarkReviewed records approval or return, with who and why.
	MarkReviewed(ctx context.Context, id string, status domain.ReportStatus, reviewerID, comment string, at time.Time) error
	MarkSuperseded(ctx context.Context, id string) error
	// FindApproved returns the live approved report for a site and day.
	// ErrNotFound when there is none.
	FindApproved(ctx context.Context, siteID string, date time.Time) (domain.DailyReport, error)
	NextRevision(ctx context.Context, siteID string, date time.Time) (int, error)
	// List returns reports without intervals, newest first.
	List(ctx context.Context, f ReportFilter) (reports []domain.DailyReport, total int, err error)
	// ApprovedOnDate returns approved reports with intervals for the sites.
	ApprovedOnDate(ctx context.Context, siteIDs []string, date time.Time) ([]domain.DailyReport, error)
	// LatestApproved returns, per site, the approved report with the latest
	// report date, with intervals.
	LatestApproved(ctx context.Context, siteIDs []string) (map[string]domain.DailyReport, error)
}

// IncidentFilter narrows an incident listing. Scope works as in SiteFilter.
type IncidentFilter struct {
	AllSites bool
	SiteIDs  []string
	SiteID   string
	OwnerID  string
	Statuses []domain.IncidentStatus
	Severity domain.Severity
	// OpenOnly leaves out closed incidents. PublishedOnly leaves out anything
	// not yet shared with the customer.
	OpenOnly      bool
	PublishedOnly bool
	// OverdueAt, when set, keeps only open incidents whose next update was
	// due before that time.
	OverdueAt *time.Time
	Limit     int
	Offset    int
}

// IncidentCounts is the headline numbers for a scope.
type IncidentCounts struct {
	Open, Active, Critical, Overdue int
}

type IncidentRepository interface {
	Create(ctx context.Context, i domain.Incident) (domain.Incident, error)
	// GetByID loads an incident without its history. forUpdate locks the row
	// for the surrounding transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.Incident, error)
	// Save writes the mutable fields: status and its stamps, owner, plan,
	// root cause and publication.
	Save(ctx context.Context, i domain.Incident) error
	AddUpdate(ctx context.Context, u domain.IncidentUpdate) (domain.IncidentUpdate, error)
	// Updates returns the history oldest first; customerOnly keeps just the
	// updates marked for the customer.
	Updates(ctx context.Context, incidentID string, customerOnly bool) ([]domain.IncidentUpdate, error)
	// List returns incidents, newest first.
	List(ctx context.Context, f IncidentFilter) (incidents []domain.Incident, total int, err error)
	Counts(ctx context.Context, f IncidentFilter, now time.Time) (IncidentCounts, error)
}

// DisputeFilter narrows a dispute listing.
type DisputeFilter struct {
	AllSites bool
	SiteIDs  []string
	SiteID   string
	ReportID string
	Statuses []domain.DisputeStatus
	// OpenOnly leaves out closed disputes.
	OpenOnly bool
	Limit    int
	Offset   int
}

// DisputeCounts is the headline numbers for a scope. WithColoplus are waiting
// on Coloplus (open, under review, correction agreed); WithAirtel are waiting
// on Airtel (evidence requested, or a resolution to acknowledge).
type DisputeCounts struct {
	Open, WithColoplus, WithAirtel int
}

type DisputeRepository interface {
	// Create stores a new dispute. ErrConflict when the same thing is already
	// being disputed and has not closed.
	Create(ctx context.Context, d domain.Dispute) (domain.Dispute, error)
	// GetByID loads a dispute without its history. forUpdate locks the row for
	// the surrounding transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.Dispute, error)
	// Save writes the mutable fields: status, owner, resolution and
	// acknowledgement.
	Save(ctx context.Context, d domain.Dispute) error
	AddEvent(ctx context.Context, e domain.DisputeEvent) (domain.DisputeEvent, error)
	// Events returns the history oldest first; publicOnly leaves out
	// Coloplus's internal notes.
	Events(ctx context.Context, disputeID string, publicOnly bool) ([]domain.DisputeEvent, error)
	// List returns disputes, newest first.
	List(ctx context.Context, f DisputeFilter) (disputes []domain.Dispute, total int, err error)
	Counts(ctx context.Context, f DisputeFilter) (DisputeCounts, error)
	// OpenByReport counts, per report, the disputes that have not closed.
	OpenByReport(ctx context.Context, reportIDs []string) (map[string]int, error)
}

// Mailer sends one email. A transient failure wraps apperrors.ErrUnavailable;
// a refusal that retrying cannot fix wraps apperrors.ErrMailRejected. The
// idempotency key lets the provider drop a repeat of the same send.
type Mailer interface {
	Send(ctx context.Context, mail domain.Email, idempotencyKey string) (providerID string, err error)
}

type NotificationFilter struct {
	Status domain.NotificationStatus
	Limit  int
	Offset int
}

// NotificationStore is the email outbox.
type NotificationStore interface {
	// Enqueue stores the notification unless one with the same dedupe key
	// exists. It reports whether this call queued it.
	Enqueue(ctx context.Context, n domain.Notification) (bool, error)
	// ClaimDue takes up to limit pending notifications that are due, counts an
	// attempt and pushes their next attempt out by lease, so a crash or a
	// second instance never sends the same one twice at once.
	ClaimDue(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]domain.Notification, error)
	MarkSent(ctx context.Context, id, providerID string, at time.Time) error
	// MarkRetry schedules another attempt. MarkFailed gives up.
	MarkRetry(ctx context.Context, id, reason string, next time.Time) error
	MarkFailed(ctx context.Context, id, reason string) error
	// Requeue puts a failed notification back with a fresh set of attempts.
	// ErrNotFound when it is not failed.
	Requeue(ctx context.Context, id string, at time.Time) error
	GetByID(ctx context.Context, id string) (domain.Notification, error)
	List(ctx context.Context, f NotificationFilter) (items []domain.Notification, total int, err error)
}

// UserTokenRepository stores one-time link tokens by hash.
type UserTokenRepository interface {
	Create(ctx context.Context, t domain.UserToken, hash string) (domain.UserToken, error)
	// GetByHash locks the row for the surrounding transaction.
	GetByHash(ctx context.Context, hash string) (domain.UserToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
	// RevokeOutstanding uses up every unused token of a purpose for a user.
	RevokeOutstanding(ctx context.Context, userID string, purpose domain.TokenPurpose, at time.Time) error
	// CountSince counts tokens of a purpose made for a user since a time.
	CountSince(ctx context.Context, userID string, purpose domain.TokenPurpose, since time.Time) (int, error)
}
