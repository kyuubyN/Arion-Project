// SPDX-License-Identifier: GPL-3.0-only

package security

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidScheme   = errors.New("security: invalid URL scheme (only http and https allowed)")
	ErrForbiddenIP     = errors.New("security: destination IP is restricted (loopback, private, link-local, or multicast)")
	ErrRedirectBlocked = errors.New("security: redirect destination blocked by SSRF policy")
	ErrPayloadTooLarge = errors.New("security: response size exceeds maximum allowed limit")
)

type SSRFGuard struct {
	blockedCount uint64
	auditLogger  *AuditLogger
}

func NewSSRFGuard(logger *AuditLogger) *SSRFGuard {
	return &SSRFGuard{
		auditLogger: logger,
	}
}

func (s *SSRFGuard) BlockedCount() uint64 {
	return atomic.LoadUint64(&s.blockedCount)
}

// ValidateURL parses rawURL and checks scheme and resolved IP addresses against restricted networks.
func (s *SSRFGuard) ValidateURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		s.recordBlocked("invalid_url_syntax", rawURL)
		return nil, fmt.Errorf("invalid URL syntax: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		s.recordBlocked("forbidden_scheme", rawURL)
		return nil, ErrInvalidScheme
	}

	host := parsedURL.Hostname()
	if host == "" {
		s.recordBlocked("empty_host", rawURL)
		return nil, errors.New("security: empty hostname in URL")
	}

	// Check if host is direct IP string
	if ip := net.ParseIP(host); ip != nil {
		if s.IsRestrictedIP(ip) {
			s.recordBlocked("restricted_direct_ip", rawURL)
			return nil, ErrForbiddenIP
		}
		return parsedURL, nil
	}

	// Perform DNS resolution and check all resolved IPs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		s.recordBlocked("dns_lookup_failed", rawURL)
		return nil, fmt.Errorf("security: DNS resolution failed for host %s: %v", host, err)
	}

	for _, ip := range ips {
		if s.IsRestrictedIP(ip) {
			s.recordBlocked("restricted_resolved_ip", rawURL)
			return nil, fmt.Errorf("%w: host %s resolved to restricted IP %s", ErrForbiddenIP, host, ip.String())
		}
	}

	return parsedURL, nil
}

// IsRestrictedIP checks if an IP belongs to loopback, private (RFC1918), link-local, or multicast ranges.
func (s *SSRFGuard) IsRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Standard net.IP methods
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	// IPv4 specific restricted ranges
	ip4 := ip.To4()
	if ip4 != nil {
		// 127.0.0.0/8 Loopback
		if ip4[0] == 127 {
			return true
		}
		// 10.0.0.0/8 Private
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12 Private
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16 Private
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 Link-local / IMDS
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 Carrier-grade NAT
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}

	return false
}

func (s *SSRFGuard) recordBlocked(reason, targetURL string) {
	atomic.AddUint64(&s.blockedCount, 1)
	if s.auditLogger != nil {
		s.auditLogger.LogEvent(EventSSRFBlocked, SeverityHigh, "SSRFGuard", reason, RedactURL(targetURL))
	}
}

// NewSecureHTTPClient returns a hardened http.Client configured with timeouts, redirect validation, and SSRFGuard checks.
func (s *SSRFGuard) NewSecureHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Validate target IP again during dial to defeat DNS rebinding
			ip := net.ParseIP(host)
			if ip != nil {
				if s.IsRestrictedIP(ip) {
					return nil, ErrForbiddenIP
				}
			} else {
				ips, lookupErr := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if lookupErr != nil || len(ips) == 0 {
					return nil, fmt.Errorf("security: dial DNS resolution failed: %v", lookupErr)
				}
				for _, resolvedIP := range ips {
					if s.IsRestrictedIP(resolvedIP) {
						return nil, ErrForbiddenIP
					}
				}
				// Dial the first safe resolved IP directly
				addr = net.JoinHostPort(ips[0].String(), port)
			}

			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   5,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("security: maximum redirect limit (5) exceeded")
			}
			// Validate redirect destination URL through SSRFGuard
			if _, err := s.ValidateURL(req.URL.String()); err != nil {
				s.recordBlocked("redirect_ssrf_detected", req.URL.String())
				return fmt.Errorf("%w: %v", ErrRedirectBlocked, err)
			}
			return nil
		},
	}
}

// BoundedRead reads at most maxBytes from reader using io.LimitReader to prevent Denial of Service via huge response payloads.
func BoundedRead(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 10 * 1024 * 1024 // 10MB default limit
	}
	limitedReader := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrPayloadTooLarge
	}
	return data, nil
}
