package awssigv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SignOptions struct {
	Credentials                  Credentials
	Region, Service, PayloadHash string
	Time                         time.Time
	DisableURIPathEscaping       bool
	DisablePathNormalization     bool
	DisableSessionToken          bool
}

type Signature struct {
	CanonicalRequest string
	StringToSign     string
	Authorization    string
}

func Sign(request *http.Request, options SignOptions) (Signature, error) {
	if request == nil || request.URL == nil || request.URL.Host == "" || request.Method == "" {
		return Signature{}, fmt.Errorf("SigV4 requires an absolute request URL")
	}
	if options.Region == "" || options.Service == "" || options.Time.IsZero() || options.PayloadHash == "" {
		return Signature{}, fmt.Errorf("SigV4 requires region, service, signing time and payload hash")
	}
	if err := options.Credentials.validate(options.Time); err != nil {
		return Signature{}, err
	}
	query, err := canonicalQuery(request.URL.RawQuery)
	if err != nil {
		return Signature{}, err
	}
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	timestamp := options.Time.UTC().Format("20060102T150405Z")
	request.Header.Set("X-Amz-Date", timestamp)
	if !options.DisableSessionToken && options.Credentials.SessionToken != "" {
		request.Header.Set("X-Amz-Security-Token", options.Credentials.SessionToken)
	}
	if options.Service == "s3" {
		request.Header.Set("X-Amz-Content-Sha256", options.PayloadHash)
	}
	headers, signedHeaders := canonicalHeaders(request)
	canonical := strings.Join([]string{request.Method, canonicalPath(request.URL, options), query, headers, signedHeaders, options.PayloadHash}, "\n")
	scope := timestamp[:8] + "/" + options.Region + "/" + options.Service + "/aws4_request"
	toSign := strings.Join([]string{"AWS4-HMAC-SHA256", timestamp, scope, hashPayload([]byte(canonical))}, "\n")
	key := hmacSHA256([]byte("AWS4"+options.Credentials.SecretAccessKey), timestamp[:8])
	key = hmacSHA256(key, options.Region)
	key = hmacSHA256(key, options.Service)
	key = hmacSHA256(key, "aws4_request")
	authorization := "AWS4-HMAC-SHA256 Credential=" + options.Credentials.AccessKeyID + "/" + scope + ", SignedHeaders=" + signedHeaders + ", Signature=" + hex.EncodeToString(hmacSHA256(key, toSign))
	request.Header.Set("Authorization", authorization)
	request.URL.RawQuery = query
	return Signature{CanonicalRequest: canonical, StringToSign: toSign, Authorization: authorization}, nil
}

func hashPayload(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key []byte, value string) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(value))
	return hash.Sum(nil)
}
