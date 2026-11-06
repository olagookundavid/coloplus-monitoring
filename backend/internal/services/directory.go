package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"coloplus-backend/internal/ports"
)

// DirectoryService names things for screens: it turns the ids a response
// carries into the user, site or record a person would recognise. Naming is a
// courtesy, never a gate: a failed lookup is logged and the response goes out
// with the names it could not find left blank.
type DirectoryService struct {
	dir ports.Directory
	log *slog.Logger
}

func NewDirectoryService(dir ports.Directory, log *slog.Logger) *DirectoryService {
	return &DirectoryService{dir: dir, log: log}
}

// Names maps an id to its name. The zero value and nil answer "" for everything.
type Names map[string]string

func (n Names) Get(id string) string {
	if n == nil {
		return ""
	}
	return n[id]
}

// Users names the given user ids (blanks and repeats are ignored).
func (s *DirectoryService) Users(ctx context.Context, ids ...string) Names {
	return s.lookup(ctx, "users", ids, s.dir.UserNames)
}

// Sites names the given site ids.
func (s *DirectoryService) Sites(ctx context.Context, ids ...string) Names {
	return s.lookup(ctx, "sites", ids, s.dir.SiteNames)
}

func (s *DirectoryService) lookup(ctx context.Context, what string, ids []string, fn func(context.Context, []string) (map[string]string, error)) Names {
	if s == nil {
		return nil
	}
	ids = distinct(ids)
	if len(ids) == 0 {
		return nil
	}
	out, err := fn(ctx, ids)
	if err != nil {
		s.log.Warn("could not look up names", "what", what, "error", err)
		return nil
	}
	return out
}

// Labels names records of one kind (an audit entity type such as "incident" or
// "daily_report") by their ids.
func (s *DirectoryService) Labels(ctx context.Context, entityType string, ids ...string) Names {
	return s.lookup(ctx, entityType, ids, func(ctx context.Context, ids []string) (map[string]string, error) {
		return s.dir.EntityLabels(ctx, entityType, ids)
	})
}

// EntityRef is one audit entity to name.
type EntityRef struct{ Type, ID string }

// EntityLabels names audit entities, one query per entity type. Months and
// export keys, which are not database ids, are worded here.
func (s *DirectoryService) EntityLabels(ctx context.Context, refs []EntityRef) map[EntityRef]string {
	out := map[EntityRef]string{}
	if s == nil {
		return out
	}
	byType := map[string][]string{}
	for _, r := range refs {
		if r.ID == "" {
			continue
		}
		switch r.Type {
		case "period":
			if t, err := time.Parse("2006-01", r.ID); err == nil {
				out[r] = t.Format("January 2006")
			}
		case "export":
			out[r] = exportLabel(r.ID)
		default:
			byType[r.Type] = append(byType[r.Type], r.ID)
		}
	}
	for typ, ids := range byType {
		found, err := s.dir.EntityLabels(ctx, typ, distinct(ids))
		if err != nil {
			s.log.Warn("could not name audit entities", "entity_type", typ, "error", err)
			continue
		}
		for id, label := range found {
			out[EntityRef{typ, id}] = label
		}
	}
	return out
}

// exportLabel words "daily_summaries:2026-11-01:2026-11-03".
func exportLabel(id string) string {
	parts := strings.Split(id, ":")
	name := strings.ReplaceAll(parts[0], "_", " ")
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	if len(parts) == 3 {
		return name + " · " + parts[1] + " to " + parts[2]
	}
	return name
}

func distinct(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
