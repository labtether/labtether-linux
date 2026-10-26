package securityruntime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

const defaultOutboundTimeout = 30 * time.Second

// Canonical HTTP policy shared by ordinary requests and context-owned streams.
func secureOutboundHTTPClient(base *http.Client, scheme string) *http.Client {
	if base == nil {
		base = &http.Client{Timeout: defaultOutboundTimeout}
	}
	client := *base
	if client.Timeout <= 0 {
		client.Timeout = defaultOutboundTimeout
	}
	client.Transport = secureOutboundRoundTripper(base.Transport, scheme)
	priorRedirect := base.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req == nil || req.URL == nil {
			return fmt.Errorf("redirect target is required")
		}
		if _, err := ValidateOutboundURL(req.URL.String()); err != nil {
			return err
		}
		if priorRedirect != nil {
			return priorRedirect(req, via)
		}
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return &client
}

func secureOutboundRoundTripper(base http.RoundTripper, scheme string) http.RoundTripper {
	var transport *http.Transport
	if t, ok := base.(*http.Transport); ok && t != nil {
		transport = t.Clone()
	} else if base == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	} else {
		return outboundValidatingRoundTripper{base: base}
	}
	transport.Proxy = nil
	transport.DialTLSContext = nil
	allowPrivate := resolvedOutboundAllowPrivate(scheme)
	allowLoopback := parseBoolEnv(envOutboundAllowLoopback, false)
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialOutboundValidated(ctx, network, address, allowLoopback, allowPrivate)
	}
	return transport
}

type outboundValidatingRoundTripper struct {
	base http.RoundTripper
}

func (rt outboundValidatingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("request is required")
	}
	if _, err := ValidateOutboundURL(req.URL.String()); err != nil {
		return nil, err
	}
	return rt.base.RoundTrip(req)
}

func dialOutboundValidated(ctx context.Context, network, address string, allowLoopback, allowPrivate bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	host = normalizeHostname(host)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	dialer := &net.Dialer{Timeout: defaultOutboundTimeout, KeepAlive: 30 * time.Second}
	if ip := net.ParseIP(host); ip != nil {
		if err := validateResolvedOutboundIP(host, ip, allowLoopback, allowPrivate); err != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("host %q did not resolve", host)
	}
	for _, addr := range addrs {
		if err := validateResolvedOutboundIP(host, addr.IP, allowLoopback, allowPrivate); err != nil {
			return nil, err
		}
	}
	var lastErr error
	for _, addr := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("host %q did not resolve to a dialable address", host)
}
