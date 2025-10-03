// Package dns provides DNS lookup functionality with built-in caching support.
//
// The package offers a simple API for resolving hostnames to IP addresses
// with automatic caching to improve performance and reduce DNS query load.
// It handles both raw hostnames and URLs, automatically extracting the
// hostname portion when needed.
//
// Features:
//   - Automatic caching with 1-hour TTL
//   - Support for both hostnames and URLs
//   - IPv4 address filtering
//   - Direct IP address passthrough
//
// Basic Usage:
//
//	// Lookup with caching (recommended)
//	ips, err := dns.CacheLookup("A", "example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, ip := range ips {
//		fmt.Printf("IP: %s\n", ip)
//	}
//
//	// Direct lookup without caching
//	ips, err := dns.Lookup("A", "https://example.com")
//	// Automatically extracts hostname from URL
//
// The package prioritizes IPv4 addresses in results and handles IP addresses
// directly without performing DNS lookups.
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

// CacheLookup performs a DNS lookup with caching support.
// Results are cached for 1 hour to reduce DNS query load.
//
// Parameters:
//   - recordType: DNS record type (e.g., "A", "AAAA"). Currently not used but reserved for future use.
//   - hostname: The hostname, URL, or IP address to resolve
//
// Returns:
//   - []net.IP: Slice of resolved IPv4 addresses
//   - error: If the lookup fails or hostname is invalid
//
// Example:
//
//	ips, err := CacheLookup("A", "google.com")
//	if err != nil {
//		return err
//	}
//	fmt.Printf("Resolved IPs: %v\n", ips)
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

// Lookup performs a direct DNS lookup without caching.
// It uses Go's default resolver to query DNS servers.
//
// The function handles multiple input formats:
//   - Raw hostnames: "example.com"
//   - URLs: "https://example.com:8080/path" (extracts hostname)
//   - IP addresses: "192.168.1.1" (returns immediately without lookup)
//
// Only IPv4 addresses are returned in the result.
//
// Parameters:
//   - recordType: DNS record type (currently not used but reserved for future use)
//   - hostname: The hostname, URL, or IP address to resolve
//
// Returns:
//   - []net.IP: Slice of resolved IPv4 addresses
//   - error: If the lookup fails or hostname is invalid
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
