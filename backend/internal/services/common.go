package services

import (
	"fmt"
	"strings"

	apperrors "coloplus-backend/internal/errors"
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
