package services

import (
	"context"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// AlertService is the in-app alert centre. A person sees only their own alerts
// and can acknowledge them; nobody else can see or clear someone's alerts.
type AlertService struct {
	alerts ports.AlertRepository
	clock  ports.Clock
}

func NewAlertService(alerts ports.AlertRepository, clock ports.Clock) *AlertService {
	return &AlertService{alerts: alerts, clock: clock}
}

type AlertList struct {
	Alerts []domain.Alert
	Total  int
	Unread int
}

// List returns the caller's alerts, newest first.
func (s *AlertService) List(ctx context.Context, actor domain.Actor, unreadOnly bool, limit, offset int) (AlertList, error) {
	if !actor.Can(domain.PermAlertsRead) {
		return AlertList{}, apperrors.ErrForbidden
	}
	limit, offset = Page(limit, offset)
	alerts, total, unread, err := s.alerts.List(ctx, actor.UserID, unreadOnly, limit, offset)
	return AlertList{Alerts: alerts, Total: total, Unread: unread}, err
}

// Acknowledge clears one of the caller's own alerts. Another person's alert,
// or one that does not exist, is not found.
func (s *AlertService) Acknowledge(ctx context.Context, actor domain.Actor, id string) error {
	if !actor.Can(domain.PermAlertsRead) {
		return apperrors.ErrForbidden
	}
	ok, err := s.alerts.Acknowledge(ctx, actor.UserID, id, s.clock.Now())
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.ErrNotFound
	}
	return nil
}

// AcknowledgeAll clears every open alert of the caller and says how many.
func (s *AlertService) AcknowledgeAll(ctx context.Context, actor domain.Actor) (int, error) {
	if !actor.Can(domain.PermAlertsRead) {
		return 0, apperrors.ErrForbidden
	}
	return s.alerts.AcknowledgeAll(ctx, actor.UserID, s.clock.Now())
}
