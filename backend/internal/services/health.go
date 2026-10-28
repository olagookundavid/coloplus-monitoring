package services

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/ports"
)

// HealthService answers whether the service can do its job.
type HealthService struct {
	db      ports.Pinger
	timeout time.Duration
}

func NewHealthService(db ports.Pinger) *HealthService {
	return &HealthService{db: db, timeout: 2 * time.Second}
}

// Check pings the database with a short deadline.
func (s *HealthService) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}
