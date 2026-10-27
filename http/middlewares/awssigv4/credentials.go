package awssigv4

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expires         time.Time
}

type CredentialsProvider interface {
	Retrieve(context.Context) (Credentials, error)
}

type Config struct {
	Region      string
	Credentials CredentialsProvider
}

type CredentialsProviderFunc func(context.Context) (Credentials, error)

func (f CredentialsProviderFunc) Retrieve(ctx context.Context) (Credentials, error) { return f(ctx) }

type StaticCredentialsProvider struct{ Credentials Credentials }

func NewStaticCredentialsProvider(credentials Credentials) StaticCredentialsProvider {
	return StaticCredentialsProvider{Credentials: credentials}
}

func (p StaticCredentialsProvider) Retrieve(ctx context.Context) (Credentials, error) {
	if err := ctx.Err(); err != nil {
		return Credentials{}, err
	}
	if err := p.Credentials.validate(time.Now()); err != nil {
		return Credentials{}, err
	}
	return p.Credentials, nil
}

type EnvironmentProvider struct{}

func (EnvironmentProvider) Retrieve(ctx context.Context) (Credentials, error) {
	return NewStaticCredentialsProvider(Credentials{
		AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), SessionToken: os.Getenv("AWS_SESSION_TOKEN"),
	}).Retrieve(ctx)
}

type CredentialsCache struct {
	provider    CredentialsProvider
	mu          sync.Mutex
	credentials Credentials
	refreshAt   time.Time
	now         func() time.Time
}

func NewCredentialsCache(provider CredentialsProvider) *CredentialsCache {
	if provider == nil {
		panic("AWS credentials cache requires a provider")
	}
	return &CredentialsCache{provider: provider, now: time.Now}
}

func (c *CredentialsCache) Retrieve(ctx context.Context) (Credentials, error) {
	if err := ctx.Err(); err != nil {
		return Credentials{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.credentials.AccessKeyID != "" && (c.refreshAt.IsZero() || now.Before(c.refreshAt)) {
		return c.credentials, nil
	}
	credentials, err := c.provider.Retrieve(ctx)
	if err != nil {
		return Credentials{}, fmt.Errorf("retrieve AWS credentials: %w", err)
	}
	if err := credentials.validate(now); err != nil {
		return Credentials{}, err
	}
	c.credentials = credentials
	c.refreshAt = time.Time{}
	if !credentials.Expires.IsZero() {
		window := min(time.Minute, credentials.Expires.Sub(now)/10)
		c.refreshAt = credentials.Expires.Add(-window)
	}
	return credentials, nil
}

func (c Credentials) validate(now time.Time) error {
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("AWS credentials require access key and secret key")
	}
	if !c.Expires.IsZero() && !c.Expires.After(now) {
		return fmt.Errorf("AWS credentials have expired")
	}
	return nil
}
