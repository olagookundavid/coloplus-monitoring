package services

import (
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
)

// The wording of every email lives here, so it can be reviewed in one place.
// Emails are plain text and never carry internal incident detail.

func inviteEmail(name, link string, ttl time.Duration) (subject, body string) {
	return "You have been invited to the Coloplus portal",
		fmt.Sprintf("Hello %s,\n\nYou have been given access to the Coloplus Remote Monitoring Portal.\n\n"+
			"Choose your password here:\n%s\n\nThis link works once and expires in %s. "+
			"If you were not expecting this, you can ignore this email.\n",
			name, link, humanDuration(ttl))
}

func resetEmail(name, link string, ttl time.Duration) (subject, body string) {
	return "Reset your Coloplus password",
		fmt.Sprintf("Hello %s,\n\nSomeone asked to reset the password for your Coloplus account. "+
			"If that was you, choose a new password here:\n%s\n\nThis link works once and expires in %s. "+
			"If it was not you, ignore this email; your password has not changed.\n",
			name, link, humanDuration(ttl))
}

func overdueIncidentEmail(i domain.Incident, site domain.Site, due time.Time, loc *time.Location, link string) (subject, body string) {
	return fmt.Sprintf("Update overdue: %s %s", incidentRef(i.Number), i.Title),
		fmt.Sprintf("The update promised for %s %s at %s was due %s and has not been given.\n\n"+
			"Stage: %s\nSeverity: %s\nNext action: %s\n\nOpen it: %s\n",
			incidentRef(i.Number), i.Title, siteLabel(site), due.In(loc).Format(timeLayout),
			statusWord(i.Status), i.Severity, orDash(i.NextAction), link)
}

// criticalIncidentEmail goes to the customer, so it carries only what the
// published view carries.
func criticalIncidentEmail(i domain.Incident, site domain.Site, loc *time.Location, link string) (subject, body string) {
	return fmt.Sprintf("Critical incident at %s: %s", siteLabel(site), i.Title),
		fmt.Sprintf("A critical incident has been reported at %s.\n\n%s\nStarted: %s\nStage: %s\n\nDetails and updates: %s\n",
			siteLabel(site), i.Title, i.StartedAt.In(loc).Format(timeLayout), statusWord(i.Status), link)
}

const timeLayout = "2 Jan 2006 15:04 MST"

func incidentRef(n int64) string { return fmt.Sprintf("INC-%06d", n) }

func siteLabel(s domain.Site) string {
	if s.Name == "" {
		return s.AirtelCode
	}
	return fmt.Sprintf("%s (%s)", s.Name, s.AirtelCode)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func humanDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
	if d >= time.Hour {
		return fmt.Sprintf("%d hour(s)", int(d.Hours()))
	}
	return fmt.Sprintf("%d minutes", int(d.Minutes()))
}
