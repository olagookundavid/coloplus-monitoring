package r2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PresignParams describes one signed link. It is the AWS Signature Version 4
// "query string" form, which R2 accepts with the region "auto".
type PresignParams struct {
	Method string
	// Host and Path are the request's host and (already path-style) URI path,
	// for example "acct.r2.cloudflarestorage.com" and "/bucket/key".
	Host string
	Path string
	// Query holds extra signed query parameters, for example the response
	// headers a download should carry.
	Query map[string]string
	// Headers are signed too: the caller of the link must send each one with
	// exactly this value. Host is added automatically.
	Headers map[string]string

	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Service         string
	Now             time.Time
	TTL             time.Duration
}

// PresignURL returns the full https URL, signature included. Only the standard
// library is used; the output for AWS's published example is checked by a test.
func PresignURL(p PresignParams) string {
	if p.Region == "" {
		p.Region = "auto"
	}
	if p.Service == "" {
		p.Service = "s3"
	}
	now := p.Now.UTC()
	date := now.Format("20060102")
	stamp := now.Format("20060102T150405Z")
	scope := date + "/" + p.Region + "/" + p.Service + "/aws4_request"

	headers := map[string]string{"host": p.Host}
	for k, v := range p.Headers {
		headers[strings.ToLower(k)] = strings.TrimSpace(v)
	}
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	signed := strings.Join(names, ";")

	query := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    p.AccessKeyID + "/" + scope,
		"X-Amz-Date":          stamp,
		"X-Amz-Expires":       strconv.Itoa(int(p.TTL / time.Second)),
		"X-Amz-SignedHeaders": signed,
	}
	for k, v := range p.Query {
		query[k] = v
	}
	canonicalQuery := encodeQuery(query)

	var canonicalHeaders strings.Builder
	for _, n := range names {
		canonicalHeaders.WriteString(n + ":" + headers[n] + "\n")
	}
	canonicalURI := encodePath(p.Path)
	canonical := strings.Join([]string{
		p.Method, canonicalURI, canonicalQuery, canonicalHeaders.String(), signed, "UNSIGNED-PAYLOAD",
	}, "\n")

	sum := sha256.Sum256([]byte(canonical))
	toSign := strings.Join([]string{"AWS4-HMAC-SHA256", stamp, scope, hex.EncodeToString(sum[:])}, "\n")

	key := hmacSHA256([]byte("AWS4"+p.SecretAccessKey), date)
	key = hmacSHA256(key, p.Region)
	key = hmacSHA256(key, p.Service)
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, toSign))

	return "https://" + p.Host + canonicalURI + "?" + canonicalQuery + "&X-Amz-Signature=" + signature
}

func hmacSHA256(key []byte, msg string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

func encodeQuery(q map[string]string) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = uriEncode(k) + "=" + uriEncode(q[k])
	}
	return strings.Join(parts, "&")
}

// encodePath encodes each path segment and keeps the slashes.
func encodePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = uriEncode(s)
	}
	return strings.Join(segs, "/")
}

// uriEncode is the AWS flavour of percent-encoding: only A-Z a-z 0-9 - _ . ~
// stay as they are, everything else (a slash included) becomes %XX.
func uriEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		default:
			b.WriteString("%" + strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}
