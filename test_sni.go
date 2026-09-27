package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func main() {
	var ip string
	ips, _ := net.LookupIP("api.telegram.org")
	for _, ipAddr := range ips {
		if ipAddr.To4() != nil {
			ip = ipAddr.String()
			break
		}
	}
	fmt.Printf("Dialing %s...\n", ip)

	conn, err := net.DialTimeout("tcp4", ip+":443", 5*time.Second)
	if err != nil {
		fmt.Println("TCP Dial Error:", err)
		return
	}
	defer conn.Close()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,                // we will verify manually if needed
		ServerName:         "www.microsoft.com", // Dummy SNI!
	}
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		fmt.Println("TLS Handshake Error:", err)
		return
	}

	fmt.Println("TLS Handshake SUCCESS with dummy SNI!")
	certs := tlsConn.ConnectionState().PeerCertificates
	fmt.Println("Cert Subject:", certs[0].Subject.String())
}
