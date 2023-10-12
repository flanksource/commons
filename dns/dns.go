package dns

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/patrickmn/go-cache"
)

// dnsCache serves as the global cache for dns queries
var dnsCache = cache.New(time.Hour, time.Hour)

func CacheLookup(recordType, hostname string) ([]net.IP, error) {
	key := fmt.Sprintf("%s:%s", recordType, hostname)

	if val, ok := dnsCache.Get(key); ok {
		if ips, ok := val.([]net.IP); ok {
			return ips, nil
		}
	}

	ips, err := Lookup(recordType, hostname)
	if err != nil {
		return nil, err
	}

	dnsCache.SetDefault(key, ips)
	return ips, nil
}

// Lookup looks up the record using Go's default resolver
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
