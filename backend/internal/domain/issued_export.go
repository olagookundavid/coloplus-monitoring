package domain

import "time"

// IssuedExport is the record of one export that was handed out.
type IssuedExport struct {
	ID       string
	Kind     string
	Filename string
	// Params are the request that produced it (period, region, site).
	Params    map[string]any
	RowCount  int
	SizeBytes int64
	// SHA256 is the digest of the file as sent, in lower-case hex.
	SHA256 string
	// StorageKey is where the exact file is kept; empty when it was not stored.
	StorageKey string
	IssuedBy   string
	CreatedAt  time.Time
}

// Stored reports whether the exact file was kept.
func (e IssuedExport) Stored() bool { return e.StorageKey != "" }
