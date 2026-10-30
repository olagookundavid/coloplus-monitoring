package services

import (
	"context"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// SiteService manages the site register and enforces site scope on reads.
type SiteService struct {
	sites ports.SiteRepository
	tx    ports.TxManager
	audit *AuditService
}

func NewSiteService(sites ports.SiteRepository, tx ports.TxManager, audit *AuditService) *SiteService {
	return &SiteService{sites: sites, tx: tx, audit: audit}
}

const (
	defaultFreshnessHours = 30
	fullDayEndMinute      = 24 * 60
)

// withDefaults fills the fields a site form may leave blank.
func withDefaults(s domain.Site) domain.Site {
	if s.Status == "" {
		s.Status = domain.SitePendingAcceptance
	}
	if s.ServiceStartMinute == 0 && s.ServiceEndMinute == 0 {
		s.ServiceEndMinute = fullDayEndMinute
	}
	if s.FreshnessHours == 0 {
		s.FreshnessHours = defaultFreshnessHours
	}
	return s
}

func (s *SiteService) Create(ctx context.Context, actor domain.Actor, site domain.Site) (domain.Site, error) {
	if !actor.Can(domain.PermSitesManage) {
		return domain.Site{}, apperrors.ErrForbidden
	}
	site = withDefaults(site)
	if err := site.Validate(); err != nil {
		return domain.Site{}, invalid("%s", err.Error())
	}

	var created domain.Site
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if created, err = s.sites.Create(ctx, site); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "site.created", EntityType: "site", EntityID: created.ID,
			Metadata: map[string]any{"airtel_code": created.AirtelCode},
		})
	})
	return created, err
}

// Get returns a site, answering "not found" for sites outside the caller's
// scope so a guessed id reveals nothing.
func (s *SiteService) Get(ctx context.Context, actor domain.Actor, id string) (domain.Site, error) {
	if !actor.Can(domain.PermSitesRead) || !actor.CanAccessSite(id) {
		return domain.Site{}, apperrors.ErrNotFound
	}
	return s.sites.GetByID(ctx, id)
}

type ListSitesInput struct {
	Query, State, Region string
	Status               domain.SiteStatus
	Limit, Offset        int
}

func (s *SiteService) List(ctx context.Context, actor domain.Actor, in ListSitesInput) ([]domain.Site, int, error) {
	if !actor.Can(domain.PermSitesRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	if in.Status != "" && !domain.ValidSiteStatus(in.Status) {
		return nil, 0, invalid("%q is not a known site status", in.Status)
	}
	limit, offset := Page(in.Limit, in.Offset)
	return s.sites.List(ctx, ports.SiteFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(),
		Query: in.Query, State: in.State, Region: in.Region, Status: in.Status,
		Limit: limit, Offset: offset,
	})
}

// Update replaces a site's editable fields.
func (s *SiteService) Update(ctx context.Context, actor domain.Actor, id string, site domain.Site) (domain.Site, error) {
	if !actor.Can(domain.PermSitesManage) {
		return domain.Site{}, apperrors.ErrForbidden
	}
	site = withDefaults(site)
	site.ID = id
	if err := site.Validate(); err != nil {
		return domain.Site{}, invalid("%s", err.Error())
	}

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.sites.GetByID(ctx, id); err != nil {
			return err
		}
		if err := s.sites.Update(ctx, site); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "site.updated", EntityType: "site", EntityID: id,
			Metadata: map[string]any{"status": string(site.Status)},
		})
	})
	if err != nil {
		return domain.Site{}, err
	}
	return s.sites.GetByID(ctx, id)
}
