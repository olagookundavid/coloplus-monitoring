package services

import (
	"context"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// PeriodService closes and reopens reporting months. A closed month takes no
// new report, correction or approval for any of its days; the report workflow
// asks this service's repository before it accepts one.
type PeriodService struct {
	closures ports.PeriodClosureRepository
	exports  *ExportService
	tx       ports.TxManager
	audit    *AuditService
	clock    ports.Clock
	loc      *time.Location
	// graceDays are the days after a month ends before it may be closed, so
	// late reports can still come in.
	graceDays int
}

func NewPeriodService(
	closures ports.PeriodClosureRepository, exports *ExportService, tx ports.TxManager, audit *AuditService,
	clock ports.Clock, loc *time.Location, graceDays int,
) *PeriodService {
	return &PeriodService{closures: closures, exports: exports, tx: tx, audit: audit, clock: clock, loc: loc, graceDays: graceDays}
}

// ParseMonth reads 2026-10 as the first day of that month.
func ParseMonth(v string) (time.Time, error) {
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return time.Time{}, invalid("the month must look like 2026-10")
	}
	return t, nil
}

// List returns the closures, newest first, reopened ones included.
func (s *PeriodService) List(ctx context.Context, actor domain.Actor) ([]domain.PeriodClosure, error) {
	if !actor.Can(domain.PermPeriodsManage) && !actor.Can(domain.PermAuditRead) {
		return nil, apperrors.ErrForbidden
	}
	return s.closures.List(ctx, 200)
}

// Close locks a finished month. It is allowed once the grace period after the
// month's end has passed, and it stores the month's calculation run first, so
// the figures that were closed are on record.
func (s *PeriodService) Close(ctx context.Context, actor domain.Actor, month time.Time) (domain.PeriodClosure, error) {
	if !actor.Can(domain.PermPeriodsManage) {
		return domain.PeriodClosure{}, apperrors.ErrForbidden
	}
	month = domain.MonthOf(month)
	y, m, d := s.clock.Now().In(s.loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if today.Before(month.AddDate(0, 1, s.graceDays)) {
		return domain.PeriodClosure{}, invalid("%s can be closed from %s, %d days after it ends, so late reports can still come in",
			month.Format("January 2006"), month.AddDate(0, 1, s.graceDays).Format("2 January 2006"), s.graceDays)
	}
	if closed, err := s.closures.IsClosed(ctx, month); err != nil {
		return domain.PeriodClosure{}, err
	} else if closed {
		return domain.PeriodClosure{}, conflict("%s is already closed", month.Format("January 2006"))
	}
	run, err := s.exports.StoreRun(ctx, actor, month, "", "Close "+month.Format("2006-01"))
	if err != nil {
		return domain.PeriodClosure{}, err
	}
	var out domain.PeriodClosure
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.closures.Close(ctx, domain.PeriodClosure{Month: month, ClosedBy: actor.UserID, ClosedAt: s.clock.Now(), RunID: run.ID})
		if err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "period.closed", EntityType: "period", EntityID: month.Format("2006-01"),
			Metadata: map[string]any{"run_id": run.ID, "sites": len(run.Rows)},
		})
	})
	return out, err
}

// Reopen lifts a month's closure. The reason is required and kept in the audit
// log; closing it again later stores a new run.
func (s *PeriodService) Reopen(ctx context.Context, actor domain.Actor, month time.Time, reason string) (domain.PeriodClosure, error) {
	if !actor.Can(domain.PermPeriodsManage) {
		return domain.PeriodClosure{}, apperrors.ErrForbidden
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 10 {
		return domain.PeriodClosure{}, invalid("say why the month is being reopened (at least 10 characters)")
	}
	month = domain.MonthOf(month)
	var out domain.PeriodClosure
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if out, err = s.closures.Reopen(ctx, month, actor.UserID, reason, s.clock.Now()); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "period.reopened", EntityType: "period", EntityID: month.Format("2006-01"),
			Metadata: map[string]any{"reason": reason},
		})
	})
	return out, err
}

// WithPeriods makes the report workflow refuse changes to a closed month.
func (s *ReportService) WithPeriods(p ports.PeriodClosureRepository) *ReportService {
	s.periods = p
	return s
}

// WithAdjustments makes a report's figures take planned maintenance and the
// midnight carry-over into account.
func (s *ReportService) WithAdjustments(a *Adjustments) *ReportService {
	s.adjust = a
	return s
}

// ensureOpen refuses work on a day whose month is closed.
func (s *ReportService) ensureOpen(ctx context.Context, date time.Time) error {
	if s.periods == nil {
		return nil
	}
	closed, err := s.periods.IsClosed(ctx, date)
	if err != nil {
		return err
	}
	if closed {
		return conflict("%s is closed, so reports for it can no longer be added, corrected or approved. An administrator can reopen the month and must say why", date.Format("January 2006"))
	}
	return nil
}
