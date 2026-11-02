package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// AttachmentOwner is what a file is attached to.
type AttachmentOwner string

const (
	AttachTicket     AttachmentOwner = "ticket"
	AttachIncident   AttachmentOwner = "incident"
	AttachDispute    AttachmentOwner = "dispute"
	AttachAcceptance AttachmentOwner = "acceptance"
)

func ValidAttachmentOwner(o AttachmentOwner) bool {
	switch o {
	case AttachTicket, AttachIncident, AttachDispute, AttachAcceptance:
		return true
	}
	return false
}

type AttachmentStatus string

const (
	AttachmentPending  AttachmentStatus = "pending"
	AttachmentReady    AttachmentStatus = "ready"
	AttachmentRejected AttachmentStatus = "rejected"
	AttachmentExpired  AttachmentStatus = "expired"
)

// allowedAttachmentTypes is the short list of things evidence is allowed to
// be. Nothing that a browser would run or render as a page is on it, and a
// download is always sent as a file to save.
var allowedAttachmentTypes = map[string]string{
	"image/jpeg":      ".jpg .jpeg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
	"text/plain":      ".txt .log",
	"text/csv":        ".csv",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       ".xlsx",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
}

const (
	MaxAttachmentName      = 120
	MaxAttachmentsPerOwner = 25
	MaxPendingPerUser      = 20
)

type Attachment struct {
	ID          string
	SiteID      string
	OwnerType   AttachmentOwner
	OwnerID     string
	Filename    string
	ContentType string
	SizeBytes   int64
	// SHA256 is the lower-case hex digest the sender declared; the store
	// checks the upload against it.
	SHA256     string
	StorageKey string
	Status     AttachmentStatus
	// Internal files are Coloplus's own and are never shown to Airtel.
	Internal    bool
	Party       DisputeParty
	UploadedBy  string
	ScanStatus  string
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}

// NewStorageKey makes the random object key for a site's file. The file's name
// is never part of it.
func NewStorageKey(siteID string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating storage key: %w", err)
	}
	return "attachments/" + siteID + "/" + hex.EncodeToString(b[:]), nil
}

// SafeFilename keeps only the file's own name: no folders, no control
// characters, and a bounded length. It never returns an empty name.
func SafeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if unicode.IsControl(r) || r == '"' || r == '<' || r == '>' || r == '|' || r == ':' || r == '*' || r == '?' {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Trim(b.String(), " .")
	if out == "" || out == "/" {
		return "file"
	}
	if utf8.RuneCountInString(out) > MaxAttachmentName {
		ext := path.Ext(out)
		if utf8.RuneCountInString(ext) > 10 {
			ext = ""
		}
		keep := []rune(strings.TrimSuffix(out, ext))
		if len(keep) > MaxAttachmentName-utf8.RuneCountInString(ext) {
			keep = keep[:MaxAttachmentName-utf8.RuneCountInString(ext)]
		}
		out = string(keep) + ext
	}
	return out
}

// ValidateNew lists what is wrong with a file about to be sent.
func (a Attachment) ValidateNew(maxBytes int64) []string {
	var problems []string
	if !ValidAttachmentOwner(a.OwnerType) {
		problems = append(problems, "choose what the file belongs to: a ticket, incident, dispute or acceptance")
	}
	if a.OwnerID == "" {
		problems = append(problems, "say which one it belongs to")
	}
	exts, ok := allowedAttachmentTypes[a.ContentType]
	switch {
	case !ok:
		problems = append(problems, "that kind of file is not accepted; send a photo (JPEG, PNG, WebP), a PDF, a text or CSV file, or a Word or Excel file")
	case !hasExt(a.Filename, exts):
		problems = append(problems, "the file name does not match its type")
	}
	switch {
	case a.SizeBytes <= 0:
		problems = append(problems, "the file is empty")
	case a.SizeBytes > maxBytes:
		problems = append(problems, fmt.Sprintf("the file is too large; the limit is %d MB", maxBytes/(1<<20)))
	}
	if len(a.SHA256) != 64 || strings.Trim(a.SHA256, "0123456789abcdef") != "" {
		problems = append(problems, "give the file's SHA-256 as 64 lower-case hex characters")
	}
	return problems
}

func hasExt(name, exts string) bool {
	ext := strings.ToLower(path.Ext(name))
	for _, e := range strings.Fields(exts) {
		if ext == e {
			return true
		}
	}
	return false
}
