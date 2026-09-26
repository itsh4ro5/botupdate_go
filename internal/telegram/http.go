package telegram

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

// NewHTTPClient creates a robust production-grade HTTP client specifically tailored
// for Hugging Face Spaces and unstable network environments.
// It enforces strict timeouts on DNS, TLS Handshakes, and Keep-Alives.
func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			// Force IPv4 to prevent IPv6/MTU blackholes on Hugging Face Spaces
			return dialer.DialContext(ctx, "tcp4", addr)
		},

		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},

		TLSHandshakeTimeout: 15 * time.Second,

		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,

		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // Absolute maximum for the entire request
	}
}

// TestConnectivity safely tests connectivity to a given URL
func TestConnectivity(targetURL string) {
	log.Printf("Running diagnostic check for: %s", targetURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		log.Printf("Diagnostic FAIL: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Diagnostic SUCCESS: HTTP %d", resp.StatusCode)
}
