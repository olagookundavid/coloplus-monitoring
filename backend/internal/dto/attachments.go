package dto

import "time"

type RequestUploadRequest struct {
	// OwnerType is ticket, incident, dispute or acceptance.
	OwnerType string `json:"owner_type" example:"ticket"`
	// OwnerID is the id of that ticket, incident or dispute. For acceptance it
	// is the site's id: a site has one acceptance.
	OwnerID string `json:"owner_id"`
	// Filename is shown to people; it is never used as a path.
	Filename    string `json:"filename" example:"generator-panel.jpg"`
	ContentType string `json:"content_type" example:"image/jpeg"`
	SizeBytes   int64  `json:"size_bytes" example:"482113"`
	// SHA256 is the file's SHA-256 as 64 lower-case hex characters. The store
	// refuses an upload that does not match it.
	SHA256 string `json:"sha256"`
	// Shared lets the customer see a file Coloplus sends. Left out, a Coloplus file
	// is internal. The customer's own files are always visible to Coloplus.
	Shared bool `json:"shared,omitempty"`
}

// AttachmentDTO is one file. UploadedBy, Internal and ScanStatus are only sent
// to Coloplus staff.
type AttachmentDTO struct {
	ID          string `json:"id"`
	OwnerType   string `json:"owner_type"`
	OwnerID     string `json:"owner_id"`
	SiteID      string `json:"site_id"`
	SiteName    string `json:"site_name,omitempty" names:"SiteID:site"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	// Status is pending until the upload is confirmed, then ready.
	Status string `json:"status"`
	// Party is who sent it: customer or coloplus.
	Party          string     `json:"party"`
	CreatedAt      time.Time  `json:"created_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	Internal       bool       `json:"internal,omitempty"`
	UploadedBy     string     `json:"uploaded_by,omitempty"`
	UploadedByName string     `json:"uploaded_by_name,omitempty" names:"UploadedBy:user"`
	ScanStatus     string     `json:"scan_status,omitempty"`
}

// UploadTarget is where to send the file: a PUT with exactly these headers.
type UploadTarget struct {
	Method    string            `json:"method" example:"PUT"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type RequestUploadResponse struct {
	Attachment AttachmentDTO `json:"attachment"`
	Upload     UploadTarget  `json:"upload"`
}

type AttachmentListResponse struct {
	Attachments []AttachmentDTO `json:"attachments"`
}

type DownloadResponse struct {
	// URL works for a couple of minutes and makes the browser save the file.
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	Filename  string    `json:"filename"`
}
