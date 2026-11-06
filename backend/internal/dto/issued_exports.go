package dto

import "time"

// IssuedExportDTO is the record of one export that was handed out. The file's
// SHA-256 lets anyone check a copy against what was issued.
type IssuedExportDTO struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind" example:"monthly_performance"`
	Filename string         `json:"filename"`
	Params   map[string]any `json:"params"`
	RowCount int            `json:"row_count"`
	// SizeBytes and SHA256 describe the file as sent.
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
	// Stored: the exact file was kept and can be fetched again.
	Stored   bool   `json:"stored"`
	IssuedBy string `json:"issued_by"`
	// SiteID and SiteName repeat the site the export was limited to, taken from params.
	SiteID       string    `json:"site_id,omitempty"`
	SiteName     string    `json:"site_name,omitempty" names:"SiteID:site"`
	IssuedByName string    `json:"issued_by_name,omitempty" names:"IssuedBy:user"`
	CreatedAt    time.Time `json:"created_at"`
}

type IssuedExportListResponse struct {
	Exports []IssuedExportDTO `json:"exports"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}
