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
	// ResetTOTP turns two-step verification off and ends older sessions.
	ResetTOTP(ctx context.Context, id string, at time.Time) error
	// RoleRequiresMFA says whether the role must sign in with two steps.
	RoleRequiresMFA(ctx context.Context, role domain.Role) (bool, error)
	// SetPassword stores a new hash, clears lockout, and ends older sessions.
	SetPassword(ctx context.Context, id, hash string, at time.Time) error
	// UsersWithPermission returns active users whose role holds the
	// permission and who can see the site (all sites, or assigned to it).
	UsersWithPermission(ctx context.Context, perm domain.Permission, siteID string) ([]domain.User, error)

	PermissionsForRole(ctx context.Context, role domain.Role) ([]domain.Permission, error)
	SiteIDsForUser(ctx context.Context, userID string) ([]string, error)
	// SiteIDsForUsers batch-loads site assignments for every id given, keyed
	// by user id -- the List endpoint's answer to the N+1 that fetching each
	// user's sites one at a time would be.
	SiteIDsForUsers(ctx context.Context, userIDs []string) (map[string][]string, error)
	SetUserSites(ctx context.Context, userID string, siteIDs []string) error
}

// RecoveryCodeRepository keeps the hashes of a person's one-time recovery codes.
type RecoveryCodeRepository interface {
	// Replace removes all the person's codes and stores the new hashes.
	Replace(ctx context.Context, userID string, hashes []string) error
	// Consume uses a code up. It reports false when there is no unused match,
	// and never succeeds twice for one code, even under concurrent requests.
	Consume(ctx context.Context, userID, hash string, at time.Time) (bool, error)
	// Remaining counts the unused codes.
	Remaining(ctx context.Context, userID string) (int, error)
	DeleteAll(ctx context.Context, userID string) error
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
	// AuthorID narrows to one author's own reports (e.g. a reporter's "my
	// reports"). Empty means no author narrowing.
	AuthorID string
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
	// ApprovedBetween returns one site's approved reports, with intervals,
	// for report dates from..to inclusive, newest first.
	ApprovedBetween(ctx context.Context, siteID string, from, to time.Time) ([]domain.DailyReport, error)
	// ApprovedInRange returns the sites' approved reports, with intervals, for
	// report dates from..to inclusive, ordered by site and date.
	ApprovedInRange(ctx context.Context, siteIDs []string, from, to time.Time) ([]domain.DailyReport, error)
	// LatestApproved returns, per site, the approved report with the latest
	// report date, with intervals.
	LatestApproved(ctx context.Context, siteIDs []string) (map[string]domain.DailyReport, error)
	// StatusForSitesOnDates returns, for each site and date, the report that
	// best represents that day's live status: the in-progress one (draft,
	// submitted or rejected) when there is one, otherwise the approved one.
	// Keyed by DayKey(siteID, date). Intervals are not attached; this is for
	// status display, not detail.
	StatusForSitesOnDates(ctx context.Context, siteIDs []string, dates []time.Time) (map[string]domain.DailyReport, error)
	// OutageDayCount counts the distinct site/day pairs in from..to
	// (inclusive) whose current submitted or approved report carries at
	// least one outage interval. An approximation, not a full weekly
	// aggregation.
	OutageDayCount(ctx context.Context, siteIDs []string, from, to time.Time) (int, error)
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
	// StartedFrom and StartedBefore keep incidents that started in
	// [StartedFrom, StartedBefore).
	StartedFrom, StartedBefore *time.Time
	Limit                      int
	Offset                     int
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

// IncidentCorrectionFilter narrows an incident-correction listing. Scope
// works as in IncidentFilter.
type IncidentCorrectionFilter struct {
	AllSites   bool
	SiteIDs    []string
	SiteID     string
	IncidentID string
	Statuses   []domain.CorrectionStatus
	Limit      int
	Offset     int
}

type IncidentCorrectionRepository interface {
	// Create stores a new correction request. ErrConflict when the same
	// incident and field already has one awaiting review.
	Create(ctx context.Context, c domain.IncidentCorrection) (domain.IncidentCorrection, error)
	// GetByID loads a correction. forUpdate locks it for the surrounding
	// transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.IncidentCorrection, error)
	// Save writes the mutable fields: status, reviewer, review note and
	// decided-at. The original/corrected values and reason are write-once,
	// set only at Create.
	Save(ctx context.Context, c domain.IncidentCorrection) error
	// List returns corrections, newest first.
	List(ctx context.Context, f IncidentCorrectionFilter) (items []domain.IncidentCorrection, total int, err error)
}

// DisputeDay is the dispute state of one site and reporting day.
type DisputeDay struct{ Open, Total int }

// DayKey identifies one site's reporting day in maps keyed by day.
func DayKey(siteID string, date time.Time) string { return siteID + "|" + date.Format("2006-01-02") }

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
	// ByDay counts the disputes that touch each site and reporting day in
	// from..to: against any revision of that day's report, or covering the day
	// by period. Keyed by DayKey.
	ByDay(ctx context.Context, siteIDs []string, from, to time.Time) (map[string]DisputeDay, error)
}

// AcceptanceFilter narrows the commissioning backlog. Scope works as in SiteFilter.
type AcceptanceFilter struct {
	AllSites bool
	SiteIDs  []string
	Stages   []domain.AcceptanceStage
	// OpenOnly leaves out accepted sites.
	OpenOnly bool
	// OverdueAt, when set, keeps only those whose due time passed before it
	// and that are neither on hold nor accepted.
	OverdueAt *time.Time
	Limit     int
	Offset    int
}

type AcceptanceRepository interface {
	// Create starts the process for a site. ErrConflict when it already has one.
	Create(ctx context.Context, a domain.Acceptance) (domain.Acceptance, error)
	// GetBySite loads a site's acceptance; forUpdate locks it for the transaction.
	GetBySite(ctx context.Context, siteID string, forUpdate bool) (domain.Acceptance, error)
	Save(ctx context.Context, a domain.Acceptance) error
	AddEvent(ctx context.Context, e domain.AcceptanceEvent) (domain.AcceptanceEvent, error)
	Events(ctx context.Context, acceptanceID string) ([]domain.AcceptanceEvent, error)
	// List returns the backlog, longest-waiting first.
	List(ctx context.Context, f AcceptanceFilter) (items []domain.Acceptance, total int, err error)
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

// TicketFilter narrows a ticket listing. Scope works as in SiteFilter.
type TicketFilter struct {
	AllSites   bool
	SiteIDs    []string
	SiteID     string
	Statuses   []domain.TicketStatus
	Category   domain.TicketCategory
	Priority   domain.TicketPriority
	IncidentID string
	// OpenOnly leaves out resolved tickets.
	OpenOnly bool
	Limit    int
	Offset   int
}

// TicketCounts is the headline numbers for a scope. CriticalOpen are
// unresolved tickets whose requester asked for critical priority.
type TicketCounts struct {
	Open, InProgress, Resolved, CriticalOpen int
}

type TicketRepository interface {
	Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error)
	// ByClientKey finds the ticket this person already made with the key;
	// ErrNotFound when none.
	ByClientKey(ctx context.Context, raisedBy, key string) (domain.Ticket, error)
	// GetByID loads a ticket. forUpdate locks the row for the surrounding
	// transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.Ticket, error)
	// Save writes the mutable fields: status, assignee, resolution, incident link.
	Save(ctx context.Context, t domain.Ticket) error
	AddEvent(ctx context.Context, e domain.TicketEvent) (domain.TicketEvent, error)
	// Events returns the history oldest first; publicOnly leaves out
	// Coloplus's internal notes.
	Events(ctx context.Context, ticketID string, publicOnly bool) ([]domain.TicketEvent, error)
	// List returns tickets, newest first.
	List(ctx context.Context, f TicketFilter) (tickets []domain.Ticket, total int, err error)
	Counts(ctx context.Context, f TicketFilter) (TicketCounts, error)
}

// UploadRequest asks the store for a short-lived link a browser can PUT one
// file to. The link only works for exactly this key, type, size and digest.
type UploadRequest struct {
	Key         string
	ContentType string
	SizeBytes   int64
	// SHA256Hex is the lower-case hex digest the upload must match.
	SHA256Hex string
	TTL       time.Duration
}

type PresignedUpload struct {
	Method string
	URL    string
	// Headers must be sent with the PUT exactly as given (the browser adds
	// Content-Length itself).
	Headers   map[string]string
	ExpiresAt time.Time
}

// DownloadRequest asks for a short-lived link to read one file. The link
// makes the browser save it rather than display it.
type DownloadRequest struct {
	Key         string
	Filename    string
	ContentType string
	TTL         time.Duration
}

// ObjectInfo is what the store reports about a stored object.
type ObjectInfo struct {
	SizeBytes int64
	// SHA256Hex is the digest the store recorded; empty when it did not say.
	SHA256Hex string
}

// ObjectStore is private file storage. Nothing here is public: every read is
// a link issued after a permission check, and every link expires.
type ObjectStore interface {
	PresignUpload(req UploadRequest) (PresignedUpload, error)
	PresignDownload(req DownloadRequest) (url string, expiresAt time.Time, err error)
	// Stat reports a stored object; apperrors.ErrNotFound when it is not there.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	// Delete removes an object. Deleting one that is not there is not an error.
	Delete(ctx context.Context, key string) error
}

type AttachmentRepository interface {
	// Create stores a pending attachment.
	Create(ctx context.Context, a domain.Attachment) (domain.Attachment, error)
	// GetByID loads one; forUpdate locks it for the surrounding transaction.
	GetByID(ctx context.Context, id string, forUpdate bool) (domain.Attachment, error)
	// SetStatus moves a pending attachment to ready, rejected or expired. It
	// changes nothing and returns false when the attachment was not pending.
	SetStatus(ctx context.Context, id string, status domain.AttachmentStatus, at time.Time) (bool, error)
	// ListReady returns an owner's received files, oldest first; withInternal
	// includes Coloplus-only ones.
	ListReady(ctx context.Context, owner domain.AttachmentOwner, ownerID string, withInternal bool) ([]domain.Attachment, error)
	// CountForOwner counts the pending and ready files on an owner.
	CountForOwner(ctx context.Context, owner domain.AttachmentOwner, ownerID string) (int, error)
	// CountPendingBy counts a person's uploads not yet confirmed.
	CountPendingBy(ctx context.Context, userID string) (int, error)
	// StalePending returns uploads started before the given time and never confirmed.
	StalePending(ctx context.Context, before time.Time, limit int) ([]domain.Attachment, error)
}
