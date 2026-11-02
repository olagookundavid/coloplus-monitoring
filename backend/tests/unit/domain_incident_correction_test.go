package unit

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func TestIncidentCorrection_CheckApply_OrderingRules(t *testing.T) {
	started := time.Date(2026, 12, 1, 6, 0, 0, 0, time.UTC)
	detected := started.Add(5 * time.Minute)
	restored := started.Add(2 * time.Hour)
	verified := restored.Add(time.Hour)

	base := domain.Incident{StartedAt: started, DetectedAt: detected, RestoredAt: &restored, VerifiedAt: &verified}

	cases := []struct {
		name string
		c    domain.IncidentCorrection
		i    domain.Incident
		ok   bool
	}{
		{
			name: "corrected started_at after it was detected",
			c:    domain.IncidentCorrection{Field: domain.FieldStartedAt, CorrectedValue: detected.Add(time.Minute)},
			i:    base,
		},
		{
			name: "corrected started_at after restoration",
			c:    domain.IncidentCorrection{Field: domain.FieldStartedAt, CorrectedValue: restored.Add(time.Minute)},
			i:    base,
		},
		{
			name: "corrected started_at within range",
			c:    domain.IncidentCorrection{Field: domain.FieldStartedAt, CorrectedValue: started.Add(time.Minute)},
			i:    base,
			ok:   true,
		},
		{
			name: "corrected restored_at before the fault started",
			c:    domain.IncidentCorrection{Field: domain.FieldRestoredAt, CorrectedValue: started.Add(-time.Minute)},
			i:    base,
		},
		{
			name: "corrected restored_at after verification",
			c:    domain.IncidentCorrection{Field: domain.FieldRestoredAt, CorrectedValue: verified.Add(time.Minute)},
			i:    base,
		},
		{
			name: "corrected restored_at within range",
			c:    domain.IncidentCorrection{Field: domain.FieldRestoredAt, CorrectedValue: restored.Add(-time.Minute)},
			i:    base,
			ok:   true,
		},
		{
			name: "restored_at has nothing to correct when the incident was never restored",
			c:    domain.IncidentCorrection{Field: domain.FieldRestoredAt, CorrectedValue: started.Add(time.Minute)},
			i:    domain.Incident{StartedAt: started, DetectedAt: detected},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := tc.c.CheckApply(tc.i)
			if tc.ok && len(problems) != 0 {
				t.Fatalf("expected no problems, got %v", problems)
			}
			if !tc.ok && len(problems) == 0 {
				t.Fatal("expected a problem, got none")
			}
		})
	}
}
