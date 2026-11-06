package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"coloplus-backend/internal/dto"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// searchAudit godoc
//
//	@Summary		Search the audit log
//	@Description	Newest first. Needs audit access, which is separate from editing rights. A reader scoped to some sites sees only events about those sites.
//	@Tags			audit
//	@Produce		json
//	@Param			actor_id	query		string	false	"Who acted"
//	@Param			action		query		string	false	"Exact action, such as report.approved"
//	@Param			entity_type	query		string	false	"daily_report, incident, dispute, user, site..."
//	@Param			entity_id	query		string	false	"One record's id"
//	@Param			from		query		string	false	"From this time (RFC 3339), inclusive"
//	@Param			to			query		string	false	"Until this time (RFC 3339), exclusive"
//	@Param			limit		query		int		false	"Page size (default 50, max 200)"
//	@Param			offset		query		int		false	"Offset"
//	@Success		200			{object}	dto.AuditListResponse
//	@Failure		403			{object}	dto.ErrorResponse
//	@Router			/api/v1/audit [get]
func (a *api) searchAudit(c *gin.Context) {
	in := services.SearchAuditInput{
		ActorID: c.Query("actor_id"), Action: c.Query("action"), EntityType: c.Query("entity_type"),
		EntityID: c.Query("entity_id"), Limit: intQuery(c, "limit"), Offset: intQuery(c, "offset"),
	}
	var err error
	if in.From, err = optionalTime(c.Query("from")); err != nil {
		a.fail(c, err)
		return
	}
	if in.To, err = optionalTime(c.Query("to")); err != nil {
		a.fail(c, err)
		return
	}
	events, total, err := a.audit.Search(c.Request.Context(), actorFrom(c), in)
	if err != nil {
		a.fail(c, err)
		return
	}
	limit, offset := services.Page(in.Limit, in.Offset)
	resp := dto.AuditListResponse{Events: make([]dto.AuditEventDTO, 0, len(events)), Total: total, Limit: limit, Offset: offset}
	// Name the people and records an event mentions, one query per kind.
	var actorIDs, siteIDs []string
	refs := make([]services.EntityRef, 0, len(events))
	for _, e := range events {
		actorIDs = append(actorIDs, e.ActorID)
		if id, ok := e.Metadata["site_id"].(string); ok {
			siteIDs = append(siteIDs, id)
		}
		refs = append(refs, services.EntityRef{Type: e.EntityType, ID: e.EntityID})
		for key, typ := range auditMetadataRefs {
			if id, ok := e.Metadata[key].(string); ok {
				refs = append(refs, services.EntityRef{Type: typ, ID: id})
			}
		}
	}
	ctx := c.Request.Context()
	actors := a.directory.Users(ctx, actorIDs...)
	sites := a.directory.Sites(ctx, siteIDs...)
	labels := a.directory.EntityLabels(ctx, refs)
	for _, e := range events {
		meta := e.Metadata
		if meta == nil {
			meta = map[string]any{}
		}
		// Add names beside the ids in the metadata, on a copy of it.
		if named := metadataWithNames(meta, sites, labels); named != nil {
			meta = named
		}
		resp.Events = append(resp.Events, dto.AuditEventDTO{
			ID: e.ID, OccurredAt: e.OccurredAt, ActorID: e.ActorID, ActorName: actors.Get(e.ActorID),
			Action: e.Action, EntityType: e.EntityType, EntityID: e.EntityID,
			EntityName: labels[services.EntityRef{Type: e.EntityType, ID: e.EntityID}], Metadata: meta,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func optionalTime(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("%w: times must look like 2026-12-01T06:00:00+01:00", apperrors.ErrInvalidInput)
	}
	return &t, nil
}

// auditMetadataRefs are metadata keys that hold the id of another record, and
// what kind of record it is. Each gets a "<key without _id>_label" beside it.
var auditMetadataRefs = map[string]string{
	"run_id":        "calculation_run",
	"supersedes_id": "daily_report",
	"acceptance_id": "acceptance",
}

// metadataWithNames returns a copy of meta with the site's name and the
// labels of the records it refers to added, or nil when there is nothing to add.
func metadataWithNames(meta map[string]any, sites services.Names, labels map[services.EntityRef]string) map[string]any {
	var out map[string]any
	add := func(k string, v any) {
		if out == nil {
			out = make(map[string]any, len(meta)+2)
			for key, val := range meta {
				out[key] = val
			}
		}
		out[k] = v
	}
	if id, ok := meta["site_id"].(string); ok && sites.Get(id) != "" {
		add("site_name", sites.Get(id))
	}
	for key, typ := range auditMetadataRefs {
		if id, ok := meta[key].(string); ok {
			if label := labels[services.EntityRef{Type: typ, ID: id}]; label != "" {
				add(strings.TrimSuffix(key, "_id")+"_label", label)
			}
		}
	}
	return out
}
