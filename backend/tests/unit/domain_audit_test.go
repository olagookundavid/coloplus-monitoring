package unit

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func TestAuditEvent_Validate(t *testing.T) {
	now := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		event   domain.AuditEvent
		wantErr bool
	}{
		{"complete", domain.AuditEvent{OccurredAt: now, Action: "report.approved", EntityType: "daily_report"}, false},
		{"no action", domain.AuditEvent{OccurredAt: now, EntityType: "daily_report"}, true},
		{"blank action", domain.AuditEvent{OccurredAt: now, Action: "  ", EntityType: "daily_report"}, true},
		{"no entity type", domain.AuditEvent{OccurredAt: now, Action: "report.approved"}, true},
		{"no time", domain.AuditEvent{Action: "report.approved", EntityType: "daily_report"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.event.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
