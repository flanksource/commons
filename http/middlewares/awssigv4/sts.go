package awssigv4

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/commons/http/internal/httppolicy"
)

const (
	stsVersion      = "2011-06-15"
	maxSTSBodyBytes = 1 << 20
)

// STSError is the error returned for a non-2xx STS Query API response.
type STSError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *STSError) Error() string {
	return fmt.Sprintf("STS error (HTTP %d): %s: %s (request id %s)", e.StatusCode, e.Code, e.Message, e.RequestID)
}

type stsCredentialsDoc struct {
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

type stsResponseXML struct {
	AssumeRole   *stsCredentialsDoc `xml:"AssumeRoleResult>Credentials"`
	WebIdentity  *stsCredentialsDoc `xml:"AssumeRoleWithWebIdentityResult>Credentials"`
	ErrorCode    string             `xml:"Error>Code"`
	ErrorMessage string             `xml:"Error>Message"`
	RequestID    string             `xml:"RequestId"`
}

// stsCall is one STS Query API invocation. A nil Signer sends the request unsigned.
type stsCall struct {
	Action   string
	Region   string
	Endpoint string
	Client   *http.Client
	Signer   CredentialsProvider
	Params   url.Values
}

func (c stsCall) url() (string, error) {
	if c.Region == "" {
		return "", fmt.Errorf("STS %s requires a region", c.Action)
	}
	if c.Endpoint != "" {
		return c.Endpoint, nil
	}
	suffix := "amazonaws.com"
	if strings.HasPrefix(c.Region, "cn-") {
		suffix = "amazonaws.com.cn"
	}
	return "https://sts." + c.Region + "." + suffix + "/", nil
}

func (c stsCall) do(ctx context.Context) (Credentials, error) {
	endpoint, err := c.url()
	if err != nil {
		return Credentials{}, err
	}
	form := url.Values{"Action": {c.Action}, "Version": {stsVersion}}
	for key, values := range c.Params {
		form[key] = values
	}
	body := form.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return Credentials{}, fmt.Errorf("build STS %s request: %w", c.Action, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.Signer != nil {
		if err := c.sign(ctx, request, body); err != nil {
			return Credentials{}, err
		}
	}
	return c.send(request)
}

func (c stsCall) sign(ctx context.Context, request *http.Request, body string) error {
	base, err := c.Signer.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("retrieve base credentials for STS %s: %w", c.Action, err)
	}
	_, err = Sign(request, SignOptions{Credentials: base, Region: c.Region, Service: "sts", Time: time.Now(), PayloadHash: hashPayload([]byte(body))})
	if err != nil {
		return fmt.Errorf("sign STS %s request: %w", c.Action, err)
	}
	return nil
}

func (c stsCall) send(request *http.Request) (Credentials, error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	noRedirect := httppolicy.WithoutRedirects(client)
	response, err := noRedirect.Do(request)
	if err != nil {
		return Credentials{}, fmt.Errorf("STS %s request: %w", c.Action, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxSTSBodyBytes))
	if err != nil {
		return Credentials{}, fmt.Errorf("read STS %s response: %w", c.Action, err)
	}
	var parsed stsResponseXML
	xmlErr := xml.Unmarshal(payload, &parsed)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return Credentials{}, stsFailure(response.StatusCode, parsed, xmlErr, payload)
	}
	if xmlErr != nil {
		return Credentials{}, fmt.Errorf("parse STS %s response: %w", c.Action, xmlErr)
	}
	return c.credentials(parsed)
}

func stsFailure(status int, parsed stsResponseXML, xmlErr error, payload []byte) error {
	if xmlErr != nil || parsed.ErrorCode == "" {
		return fmt.Errorf("STS returned HTTP %d with unrecognized body: %.200s", status, string(payload))
	}
	return &STSError{StatusCode: status, Code: parsed.ErrorCode, Message: parsed.ErrorMessage, RequestID: parsed.RequestID}
}

func (c stsCall) credentials(parsed stsResponseXML) (Credentials, error) {
	raw := parsed.AssumeRole
	if c.Action == "AssumeRoleWithWebIdentity" {
		raw = parsed.WebIdentity
	}
	if raw == nil {
		return Credentials{}, fmt.Errorf("STS %s response contains no credentials", c.Action)
	}
	expires, err := time.Parse(time.RFC3339, strings.TrimSpace(raw.Expiration))
	if err != nil {
		return Credentials{}, fmt.Errorf("STS %s response has invalid expiration %q: %w", c.Action, raw.Expiration, err)
	}
	credentials := Credentials{AccessKeyID: raw.AccessKeyID, SecretAccessKey: raw.SecretAccessKey, SessionToken: raw.SessionToken, Expires: expires}
	if err := credentials.validate(time.Now()); err != nil {
		return Credentials{}, fmt.Errorf("STS %s returned unusable credentials: %w", c.Action, err)
	}
	return credentials, nil
}

func durationSeconds(params url.Values, duration time.Duration) {
	if duration > 0 {
		params.Set("DurationSeconds", fmt.Sprint(int(duration.Seconds())))
	}
}
