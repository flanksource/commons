package dns

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	bigcache "github.com/allegro/bigcache/v3"
	"github.com/eko/gocache/lib/v4/marshaler"
	bcstore "github.com/eko/gocache/store/bigcache/v4"
)

type Cache struct {
	*marshaler.Marshaler
}

var cache *Cache

func newCache() (*Cache, error) {
	bigcacheClient, _ := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))
	bigcacheStore := bcstore.NewBigcache(bigcacheClient)
	return &Cache{marshaler.New(bigcacheStore)}, nil
}

func init() {
	cache, _ = newCache()
}

type IPs []net.IP

func CacheLookup(ctx context.Context, recordType, hostname string) ([]net.IP, error) {
	var ips IPs
	key := fmt.Sprintf("%s:%s", recordType, hostname)

	if _, err := cache.Get(ctx, key, &ips); err == nil {
		return ips, nil
	}

	ips, err := Lookup(recordType, hostname)
	if err != nil {
		return nil, err
	}

	err = cache.Set(ctx, key, ips, nil)
	return ips, err
}

// Lookup looksup the record using Go's default resolver
func Lookup(recordType, hostname string) ([]net.IP, error) {
	host := hostname
	if url, err := url.Parse(hostname); err != nil {
		return nil, fmt.Errorf("invalid IP/URL: %s: %w", hostname, err)
	} else if url.Hostname() != "" {
		host = url.Hostname()
	}

	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("lookup of %s failed: %w", host, err)
	}

	var ipv4 []net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = append(ipv4, ip)
		}
	}

	return ipv4, nil
}
