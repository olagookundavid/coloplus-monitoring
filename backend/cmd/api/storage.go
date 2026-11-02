package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"coloplus-backend/config"
	"coloplus-backend/internal/adapters/r2"
	"coloplus-backend/internal/platform"
	"coloplus-backend/internal/ports"
)

// newObjectStore builds the file store, or returns nil when storage is not
// configured; the evidence endpoints then answer "not set up yet".
func newObjectStore(cfg config.Config) (ports.ObjectStore, error) {
	if !cfg.Storage.Enabled() {
		return nil, nil
	}
	store, err := r2.New(r2.Config{
		AccountID: cfg.Storage.AccountID, AccessKeyID: cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey, Bucket: cfg.Storage.Bucket,
		Endpoint: cfg.Storage.Endpoint, Timeout: cfg.Storage.Timeout,
	}, platform.SystemClock{})
	if err != nil {
		return nil, err
	}
	return store, nil
}

// checkStorage is `api check-storage`: it proves the bucket and keys work by
// sending a tiny file the way a browser would, reading it back and deleting it.
// It touches only one object under healthchecks/ and needs no database.
func checkStorage(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	store, err := newObjectStore(cfg)
	if err != nil {
		return err
	}
	if store == nil {
		return errors.New("storage is not configured: set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_BUCKET")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	client := &http.Client{Timeout: 20 * time.Second}

	body := []byte("coloplus storage check " + time.Now().UTC().Format(time.RFC3339))
	sum := sha256.Sum256(body)
	key := "healthchecks/" + hex.EncodeToString(sum[:8])
	step := func(name string, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		log.Info("storage check ok", "step", name)
		return nil
	}

	up, err := store.PresignUpload(ports.UploadRequest{
		Key: key, ContentType: "text/plain", SizeBytes: int64(len(body)), SHA256Hex: hex.EncodeToString(sum[:]), TTL: time.Minute,
	})
	if err := step("sign upload", err); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, up.Method, up.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	for k, v := range up.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode/100 != 2 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			err = fmt.Errorf("storage answered %d: %s", resp.StatusCode, msg)
		}
	}
	if err := step("upload with the signed link", err); err != nil {
		return err
	}

	info, err := store.Stat(ctx, key)
	if err == nil && info.SizeBytes != int64(len(body)) {
		err = fmt.Errorf("stored %d bytes, sent %d", info.SizeBytes, len(body))
	}
	if err := step("read it back (size)", err); err != nil {
		return err
	}
	if info.SHA256Hex == "" {
		log.Warn("storage did not report a checksum; the upload is still checked by size, and the signed checksum header is enforced on upload")
	} else if err := step("checksum matches", errIf(info.SHA256Hex != hex.EncodeToString(sum[:]), "checksum differs")); err != nil {
		return err
	}

	url, _, err := store.PresignDownload(ports.DownloadRequest{Key: key, Filename: "check.txt", ContentType: "text/plain", TTL: time.Minute})
	if err := step("sign download", err); err != nil {
		return err
	}
	dreq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	dresp, err := client.Do(dreq)
	var got []byte
	if err == nil {
		defer func() { _ = dresp.Body.Close() }()
		got, err = io.ReadAll(io.LimitReader(dresp.Body, 1<<20))
		if err == nil && dresp.StatusCode/100 != 2 {
			err = fmt.Errorf("storage answered %d", dresp.StatusCode)
		}
		if err == nil && dresp.Header.Get("Content-Disposition") == "" {
			err = errors.New("the download did not ask the browser to save the file")
		}
	}
	if err := step("download with the signed link", err); err != nil {
		return err
	}
	if err := step("downloaded file is the one sent", errIf(!bytes.Equal(got, body), "contents differ")); err != nil {
		return err
	}

	if err := step("delete", store.Delete(ctx, key)); err != nil {
		return err
	}
	_, err = store.Stat(ctx, key)
	if err := step("gone after delete", errIf(err == nil, "the object is still there")); err != nil {
		return err
	}
	log.Info("storage works end to end", "bucket", cfg.Storage.Bucket)
	return nil
}

func errIf(cond bool, msg string) error {
	if cond {
		return errors.New(msg)
	}
	return nil
}
