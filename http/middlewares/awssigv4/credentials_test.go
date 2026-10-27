package awssigv4

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("credential providers", func() {
	It("caches concurrent requests and refreshes before expiry without hiding refresh failures", func() {
		now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		var calls atomic.Int64
		failure := errors.New("provider unavailable")
		provider := CredentialsProviderFunc(func(context.Context) (Credentials, error) {
			if calls.Add(1) == 3 {
				return Credentials{}, failure
			}
			return Credentials{AccessKeyID: fmt.Sprint(calls.Load()), SecretAccessKey: "example-secret", Expires: now.Add(time.Hour)}, nil
		})
		cache := NewCredentialsCache(provider)
		cache.now = func() time.Time { return now }
		var group sync.WaitGroup
		for range 8 {
			group.Go(func() {
				defer GinkgoRecover()
				credentials, err := cache.Retrieve(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(credentials.AccessKeyID).To(Equal("1"))
			})
		}
		group.Wait()
		Expect(calls.Load()).To(Equal(int64(1)))
		now = now.Add(59 * time.Minute)
		credentials, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(credentials.AccessKeyID).To(Equal("2"))
		Expect(calls.Load()).To(Equal(int64(2)))
		now = now.Add(59 * time.Minute)
		_, err = cache.Retrieve(context.Background())
		Expect(errors.Is(err, failure)).To(BeTrue())
	})

	It("rejects missing keys, expired credentials and canceled cached requests", func() {
		now := time.Now()
		for _, credentials := range []Credentials{{}, {AccessKeyID: "example", SecretAccessKey: "example-secret", Expires: now.Add(-time.Second)}} {
			cache := NewCredentialsCache(CredentialsProviderFunc(func(context.Context) (Credentials, error) { return credentials, nil }))
			_, err := cache.Retrieve(context.Background())
			Expect(err).To(HaveOccurred())
		}
		cache := NewCredentialsCache(NewStaticCredentialsProvider(Credentials{AccessKeyID: "example", SecretAccessKey: "example-secret"}))
		_, err := cache.Retrieve(context.Background())
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = cache.Retrieve(ctx)
		Expect(err).To(MatchError(context.Canceled))
	})
})
