package services

import (
	"context"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// invalid builds an ErrInvalidInput that carries a specific, worded reason.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, fmt.Sprintf(format, args...))
}

func invalidProblems(problems []string) error {
	return invalid("%s", strings.Join(problems, "; "))
}

// Page clamps a requested page to sane bounds.
func Page(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func conflict(format string, args ...any) error {
	return fmt.Errorf("%w: %s", apperrors.ErrConflict, fmt.Sprintf(format, args...))
}

func forbidden(format string, args ...any) error {
	return fmt.Errorf("%w: %s", apperrors.ErrForbidden, fmt.Sprintf(format, args...))
}

// unavailableErr is for a feature whose provider is not set up or not
// reachable; the caller is told to try again later, never why.
func unavailableErr(format string, args ...any) error {
	return fmt.Errorf("%w: %s", apperrors.ErrUnavailable, fmt.Sprintf(format, args...))
}

// listAllActiveSites reads every active site, a page at a time, for jobs that
// must look at all of them. A job that stopped at the first page would skip the
// rest of a large portfolio without a word.
func listAllActiveSites(ctx context.Context, repo ports.SiteRepository) ([]domain.Site, error) {
	const page = 500
	var all []domain.Site
	for offset := 0; ; offset += page {
		sites, _, err := repo.List(ctx, ports.SiteFilter{AllSites: true, Status: domain.SiteActive, Limit: page, Offset: offset})
		if err != nil {
			return nil, err
		}
		all = append(all, sites...)
		if len(sites) < page {
			return all, nil
		}
	}
}
