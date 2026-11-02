package unit

import (
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/adapters/r2"
)

// The link signer is checked against the worked example in AWS's own
// documentation ("Authenticating Requests: Using Query Parameters"). If this
// signature is right, the algorithm is right; a wrong one would only show up
// as every upload being refused by the store.
func TestR2_SignatureMatchesTheAWSPublishedExample(t *testing.T) {
	url := r2.PresignURL(r2.PresignParams{
		Method: "GET", Host: "examplebucket.s3.amazonaws.com", Path: "/test.txt",
		AccessKeyID: "AKIAIOSFODNN7EXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region: "us-east-1", Service: "s3",
		Now: time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC), TTL: 86400 * time.Second,
	})
	const want = "aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404"
	if !strings.HasSuffix(url, "X-Amz-Signature="+want) {
		t.Fatalf("signature differs from the published one:\n%s", url)
	}
}
