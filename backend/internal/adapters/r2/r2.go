// Package r2 is private file storage on Cloudflare R2, behind ports.ObjectStore.
// R2 speaks the S3 protocol; the links are signed here with the standard
// library, so there is no SDK to carry. One http.Client with a timeout makes
// the few calls the server itself must make (look at an object, delete one).
package r2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// Config configures the store. Endpoint is optional: it defaults to the
// account's R2 address and is only set for a local S3-compatible server.
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	Timeout         time.Duration
}

type Store struct {
	cfg    Config
	host   string
	scheme string
	client *http.Client
	clock  ports.Clock
}

func New(cfg Config, clock ports.Clock) (*Store, error) {
	switch {
	case cfg.AccessKeyID == "" || cfg.SecretAccessKey == "":
		return nil, errors.New("r2: access key id and secret are required")
	case cfg.Bucket == "":
		return nil, errors.New("r2: bucket is required")
	case cfg.AccountID == "" && cfg.Endpoint == "":
		return nil, errors.New("r2: account id (or an endpoint) is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	scheme, host := "https", cfg.AccountID+".r2.cloudflarestorage.com"
	if cfg.Endpoint != "" {
		switch {
		case strings.HasPrefix(cfg.Endpoint, "https://"):
			host = strings.TrimPrefix(cfg.Endpoint, "https://")
		case strings.HasPrefix(cfg.Endpoint, "http://"):
			scheme, host = "http", strings.TrimPrefix(cfg.Endpoint, "http://")
		default:
			return nil, errors.New("r2: endpoint must start with https:// or http://")
		}
		host = strings.TrimRight(host, "/")
	}
	return &Store{cfg: cfg, host: host, scheme: scheme, client: &http.Client{Timeout: cfg.Timeout}, clock: clock}, nil
}

func (s *Store) objectPath(key string) string { return "/" + s.cfg.Bucket + "/" + key }

func (s *Store) sign(method, key string, ttl time.Duration, query, headers map[string]string) string {
	u := PresignURL(PresignParams{
		Method: method, Host: s.host, Path: s.objectPath(key), Query: query, Headers: headers,
		AccessKeyID: s.cfg.AccessKeyID, SecretAccessKey: s.cfg.SecretAccessKey,
		Now: s.clock.Now(), TTL: ttl,
	})
	if s.scheme == "http" {
		u = "http://" + strings.TrimPrefix(u, "https://")
	}
	return u
}

// PresignUpload signs a PUT for exactly this type, size and digest, so the
// store itself refuses anything else.
func (s *Store) PresignUpload(req ports.UploadRequest) (ports.PresignedUpload, error) {
	digest, err := hex.DecodeString(req.SHA256Hex)
	if err != nil || len(digest) != 32 {
		return ports.PresignedUpload{}, errors.New("r2: the digest must be 64 hex characters")
	}
	headers := map[string]string{
		"content-type":          req.ContentType,
		"content-length":        strconv.FormatInt(req.SizeBytes, 10),
		"x-amz-checksum-sha256": base64.StdEncoding.EncodeToString(digest),
	}
	now := s.clock.Now()
	return ports.PresignedUpload{
		Method: http.MethodPut, URL: s.sign(http.MethodPut, req.Key, req.TTL, nil, headers),
		Headers: map[string]string{
			"Content-Type":          headers["content-type"],
			"x-amz-checksum-sha256": headers["x-amz-checksum-sha256"],
		},
		ExpiresAt: now.Add(req.TTL),
	}, nil
}

// PresignDownload signs a GET whose response tells the browser to save the
// file under its own name rather than display it.
func (s *Store) PresignDownload(req ports.DownloadRequest) (string, time.Time, error) {
	q := map[string]string{
		"response-content-disposition": contentDisposition(req.Filename),
		"response-content-type":        req.ContentType,
	}
	return s.sign(http.MethodGet, req.Key, req.TTL, q, nil), s.clock.Now().Add(req.TTL), nil
}

// contentDisposition always asks for a download; the name is quoted for old
// clients and given in full as UTF-8 for new ones.
func contentDisposition(name string) string {
	ascii := make([]rune, 0, len(name))
	for _, r := range name {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			r = '_'
		}
		ascii = append(ascii, r)
	}
	return `attachment; filename="` + string(ascii) + `"; filename*=UTF-8''` + uriEncode(name)
}

// Stat looks at a stored object with a signed HEAD.
func (s *Store) Stat(ctx context.Context, key string) (ports.ObjectInfo, error) {
	headers := map[string]string{"x-amz-checksum-mode": "ENABLED"}
	resp, err := s.do(ctx, http.MethodHead, key, headers)
	if err != nil {
		return ports.ObjectInfo{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ports.ObjectInfo{}, apperrors.ErrNotFound
	case resp.StatusCode/100 != 2:
		return ports.ObjectInfo{}, fmt.Errorf("%w: storage answered %d", apperrors.ErrUnavailable, resp.StatusCode)
	}
	info := ports.ObjectInfo{SizeBytes: resp.ContentLength}
	if sum := resp.Header.Get("x-amz-checksum-sha256"); sum != "" {
		if raw, err := base64.StdEncoding.DecodeString(sum); err == nil {
			info.SHA256Hex = hex.EncodeToString(raw)
		}
	}
	return info, nil
}

// Delete removes an object; one that is already gone is fine.
func (s *Store) Delete(ctx context.Context, key string) error {
	resp, err := s.do(ctx, http.MethodDelete, key, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("%w: storage answered %d", apperrors.ErrUnavailable, resp.StatusCode)
	}
	return nil
}

func (s *Store) do(ctx context.Context, method, key string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.sign(method, key, time.Minute, nil, headers), nil)
	if err != nil {
		return nil, fmt.Errorf("r2: building request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: storage is not reachable: %v", apperrors.ErrUnavailable, err)
	}
	return resp, nil
}

// Put stores bytes the server holds, with a signed PUT of its own. The digest
// is signed in, so the store refuses anything but exactly these bytes.
func (s *Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	sum := sha256.Sum256(data)
	headers := map[string]string{
		"content-type":          contentType,
		"content-length":        strconv.Itoa(len(data)),
		"x-amz-checksum-sha256": base64.StdEncoding.EncodeToString(sum[:]),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.sign(http.MethodPut, key, time.Minute, nil, headers), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("r2: building request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-checksum-sha256", headers["x-amz-checksum-sha256"])
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: storage is not reachable: %v", apperrors.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: storage answered %d", apperrors.ErrUnavailable, resp.StatusCode)
	}
	return nil
}
