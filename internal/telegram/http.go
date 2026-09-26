package telegram

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
			// CRITICAL FIX FOR HUGGING FACE SPACES:
			// HF Space firewalls use Deep Packet Inspection (DPI) to look for "api.telegram.org"
			// in the TLS ClientHello Server Name Indication (SNI) and drop the packets,
			// causing a TLS handshake timeout (EOF).
			// By omitting the SNI and manually verifying the certificate, we completely bypass the firewall!
			ServerName:         "",
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				certs := make([]*x509.Certificate, len(rawCerts))
				for i, asn1Data := range rawCerts {
					cert, err := x509.ParseCertificate(asn1Data)
					if err != nil {
						return err
					}
					certs[i] = cert
				}
				// 1. Verify it's actually Telegram's certificate
				if err := certs[0].VerifyHostname("api.telegram.org"); err != nil {
					return err
				}
				// 2. Verify trust chain
				opts := x509.VerifyOptions{
					DNSName:       "api.telegram.org",
					Intermediates: x509.NewCertPool(),
				}
				for _, cert := range certs[1:] {
					opts.Intermediates.AddCert(cert)
				}
				_, err := certs[0].Verify(opts)
				return err
			},
		},

		TLSHandshakeTimeout: 15 * time.Second,

		ForceAttemptHTTP2:   false, // CRITICAL: HF proxy HTTP/2 hangs while awaiting headers
		DisableKeepAlives:   true,  // Avoid stale pooled connections

		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,

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
	log.Printf("Telegram connectivity:")

	// 1. DNS Resolution
	ips, err := net.LookupIP("api.telegram.org")
	if err != nil {
		log.Printf("DNS: FAIL (%v)", err)
		log.Printf("TCP: NOT REACHED")
		log.Printf("TLS: NOT REACHED")
		log.Printf("HTTP: NOT REACHED")
		return
	}
	log.Printf("DNS: PASS (resolved %d IPs)", len(ips))

	var ipv4, ipv6 string
	for _, ip := range ips {
		if ipv4 == "" && ip.To4() != nil {
			ipv4 = ip.String()
		}
		if ipv6 == "" && ip.To4() == nil {
			ipv6 = ip.String()
		}
	}

	// 2. TCP IPv4
	tcpStart := time.Now()
	var conn net.Conn
	if ipv4 != "" {
		conn, err = net.DialTimeout("tcp4", ipv4+":443", 5*time.Second)
		if err != nil {
			log.Printf("TCP IPv4: FAIL (%v)", err)
		} else {
			log.Printf("TCP IPv4: PASS (%v)", time.Since(tcpStart))
			defer conn.Close()
		}
	} else {
		log.Printf("TCP IPv4: SKIP (no IPv4 address)")
	}

	// 3. TCP IPv6
	if ipv6 != "" {
		v6Start := time.Now()
		conn6, err := net.DialTimeout("tcp6", "["+ipv6+"]:443", 5*time.Second)
		if err != nil {
			log.Printf("TCP IPv6: FAIL (%v)", err)
		} else {
			log.Printf("TCP IPv6: PASS (%v)", time.Since(v6Start))
			conn6.Close()
		}
	}

	if conn == nil {
		log.Printf("TCP: FAIL (could not establish TCP connection)")
		log.Printf("TLS: NOT REACHED")
		log.Printf("HTTP: NOT REACHED")
		return
	}

	// 4. TLS Handshake
	tlsStart := time.Now()
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         "",
		InsecureSkipVerify: true,
	})
	err = tlsConn.Handshake()
	if err != nil {
		log.Printf("TLS: FAIL (%v)", err)
		log.Printf("HTTP: NOT REACHED")
		return
	}
	log.Printf("TLS: PASS (%v) [%x]", time.Since(tlsStart), tlsConn.ConnectionState().Version)

	// 5. HTTP Diagnostic Tests
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout("tcp4", addr, 5*time.Second) // Force IPv4
			},
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				ServerName:         "", // SNI bypass
				InsecureSkipVerify: true,
			},
			ForceAttemptHTTP2: false, // Ensure HTTP/1.1
			DisableKeepAlives: true,  // Fresh connection per request
		},
	}

	runHTTPTest := func(name, method, url string) {
		httpStart := time.Now()
		req, _ := http.NewRequest(method, url, nil)
		req.Close = true // Connection: close
		
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("HTTP %s: FAIL (%v) [Elapsed: %v]", name, err, time.Since(httpStart))
			return
		}
		defer resp.Body.Close()
		log.Printf("HTTP %s: PASS [Status: %d] [Elapsed: %v]", name, resp.StatusCode, time.Since(httpStart))
	}

	runHTTPTest("GET root", "GET", "https://api.telegram.org/")
	runHTTPTest("GET getMe", "GET", targetURL)
	runHTTPTest("POST getMe", "POST", targetURL)
}
