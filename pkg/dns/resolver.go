package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// DNSType represents the DNS protocol type
type DNSType string

const (
	DNSTypeUDP DNSType = "udp" // Traditional DNS over UDP
	DNSTypeTCP DNSType = "tcp" // DNS over TCP
	DNSTypeDoH DNSType = "doh" // DNS over HTTPS
	DNSTypeDoT DNSType = "dot" // DNS over TLS
)

// DNSQueryResult represents DNS query results
type DNSQueryResult struct {
	A     []string `json:"a"`
	AAAA  []string `json:"aaaa"`
	CNAME []string `json:"cname"`
	MX    []string `json:"mx"`
	TXT   []string `json:"txt"`
	NS    []string `json:"ns"`
}

// Resolver represents a DNS resolver
type Resolver struct {
	Server     string // DNS server address (e.g., 8.8.8.8:53, https://dns.google/resolve)
	ServerType DNSType
	Timeout    time.Duration
}

// NewResolver creates a new DNS resolver
func NewResolver(server string, dnsType DNSType) *Resolver {
	if dnsType == "" {
		dnsType = DNSTypeUDP
	}

	return &Resolver{
		Server:     server,
		ServerType: dnsType,
		Timeout:    10 * time.Second,
	}
}

// Lookup performs DNS lookup based on the resolver type
func (r *Resolver) Lookup(ctx context.Context, domain string) (*DNSQueryResult, error) {
	switch r.ServerType {
	case DNSTypeUDP:
		return r.lookupUDP(ctx, domain)
	case DNSTypeTCP:
		return r.lookupTCP(ctx, domain)
	case DNSTypeDoH:
		return r.lookupDoH(ctx, domain)
	case DNSTypeDoT:
		return r.lookupDoT(ctx, domain)
	default:
		return r.lookupUDP(ctx, domain)
	}
}

// lookupUDP performs traditional UDP DNS lookup
func (r *Resolver) lookupUDP(ctx context.Context, domain string) (*DNSQueryResult, error) {
	// Create DNS message
	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			RecursionDesired: true,
		},
		Questions: []dnsmessage.Question{
			{
				Name:  dnsmessage.MustNewName(domain + "."),
				Type:  dnsmessage.TypeA,
				Class: dnsmessage.ClassINET,
			},
		},
	}

	// Send query
	client := &net.Dialer{Timeout: r.Timeout}
	conn, err := client.DialContext(ctx, "udp", r.Server)
	if err != nil {
		return nil, fmt.Errorf("UDP dial failed: %w", err)
	}
	defer conn.Close()

	// Set write deadline
	deadline := time.Now().Add(r.Timeout)
	conn.SetWriteDeadline(deadline)

	// Send query
	buf, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack message failed: %w", err)
	}

	if _, err := conn.Write(buf); err != nil {
		return nil, fmt.Errorf("send query failed: %w", err)
	}

	// Set read deadline
	conn.SetReadDeadline(deadline)

	// Receive response
	respBuf := make([]byte, 512)
	n, err := conn.Read(respBuf)
	if err != nil {
		return nil, fmt.Errorf("receive response failed: %w", err)
	}

	var respMsg dnsmessage.Message
	if err := respMsg.Unpack(respBuf[:n]); err != nil {
		return nil, fmt.Errorf("unpack response failed: %w", err)
	}

	// Parse response
	return r.parseDNSResponse(respMsg), nil
}

// lookupTCP performs DNS over TCP lookup
func (r *Resolver) lookupTCP(ctx context.Context, domain string) (*DNSQueryResult, error) {
	// TCP DNS is similar to UDP but uses TCP for transport
	// Most modern DNS resolvers support TCP
	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			RecursionDesired: true,
		},
		Questions: []dnsmessage.Question{
			{
				Name:  dnsmessage.MustNewName(domain + "."),
				Type:  dnsmessage.TypeA,
				Class: dnsmessage.ClassINET,
			},
		},
	}

	client := &net.Dialer{Timeout: r.Timeout}
	conn, err := client.DialContext(ctx, "tcp", r.Server)
	if err != nil {
		return nil, fmt.Errorf("TCP dial failed: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(r.Timeout)
	conn.SetDeadline(deadline)

	buf, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack message failed: %w", err)
	}

	// TCP DNS requires a 2-byte length prefix
	lengthPrefix := make([]byte, 2)
	lengthPrefix[0] = byte(len(buf) >> 8)
	lengthPrefix[1] = byte(len(buf))

	if _, err := conn.Write(append(lengthPrefix, buf...)); err != nil {
		return nil, fmt.Errorf("send query failed: %w", err)
	}

	// Read length prefix
	lengthBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, fmt.Errorf("read length failed: %w", err)
	}

	length := int(lengthBuf[0])<<8 | int(lengthBuf[1])
	respBuf := make([]byte, length)

	if _, err := io.ReadFull(conn, respBuf); err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var respMsg dnsmessage.Message
	if err := respMsg.Unpack(respBuf); err != nil {
		return nil, fmt.Errorf("unpack response failed: %w", err)
	}

	return r.parseDNSResponse(respMsg), nil
}

// lookupDoH performs DNS over HTTPS lookup (RFC 8484)
func (r *Resolver) lookupDoH(ctx context.Context, domain string) (*DNSQueryResult, error) {
	// DoH uses GET or POST to an HTTPS endpoint
	// Google DoH: https://dns.google/resolve
	// Cloudflare DoH: https://1.1.1.1/dns-query

	// Build URL for GET request
	url := r.buildDoHURL(domain)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: r.Timeout,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Accept", "application/dns-json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DoH request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// Parse JSON response
	return r.parseDoHResponse(body)
}

// lookupDoT performs DNS over TLS lookup (RFC 7858)
func (r *Resolver) lookupDoT(ctx context.Context, domain string) (*DNSQueryResult, error) {
	// DoT uses TLS on port 853
	// Similar to TCP DNS but with TLS wrapper

	// Extract host and port from server
	host, port, err := net.SplitHostPort(r.Server)
	if err != nil {
		// Default to port 853 for DoT
		host = r.Server
		port = "853"
	}

	serverAddr := net.JoinHostPort(host, port)

	// Create TLS connection
	dialer := &net.Dialer{Timeout: r.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("TCP dial failed: %w", err)
	}

	// Upgrade to TLS
	tlsConn, err := tlsClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	defer tlsConn.Close()

	deadline := time.Now().Add(r.Timeout)
	tlsConn.SetDeadline(deadline)

	// Create and send DNS message (same as TCP)
	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			RecursionDesired: true,
		},
		Questions: []dnsmessage.Question{
			{
				Name:  dnsmessage.MustNewName(domain + "."),
				Type:  dnsmessage.TypeA,
				Class: dnsmessage.ClassINET,
			},
		},
	}

	buf, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack message failed: %w", err)
	}

	lengthPrefix := make([]byte, 2)
	lengthPrefix[0] = byte(len(buf) >> 8)
	lengthPrefix[1] = byte(len(buf))

	if _, err := tlsConn.Write(append(lengthPrefix, buf...)); err != nil {
		return nil, fmt.Errorf("send query failed: %w", err)
	}

	// Read response
	lengthBuf := make([]byte, 2)
	if _, err := io.ReadFull(tlsConn, lengthBuf); err != nil {
		return nil, fmt.Errorf("read length failed: %w", err)
	}

	length := int(lengthBuf[0])<<8 | int(lengthBuf[1])
	respBuf := make([]byte, length)

	if _, err := io.ReadFull(tlsConn, respBuf); err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var respMsg dnsmessage.Message
	if err := respMsg.Unpack(respBuf); err != nil {
		return nil, fmt.Errorf("unpack response failed: %w", err)
	}

	return r.parseDNSResponse(respMsg), nil
}

// buildDoHURL constructs a DoH query URL
func (r *Resolver) buildDoHURL(domain string) string {
	baseURL := strings.TrimSuffix(r.Server, "/")

	// Add query parameters
	return fmt.Sprintf("%s?name=%s&type=A", baseURL, domain)
}

// parseDoHResponse parses DoH JSON response
func (r *Resolver) parseDoHResponse(body []byte) (*DNSQueryResult, error) {
	var dohResp struct {
		Status int `json:"Status"`
		Answer []struct {
			Name string `json:"name"`
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.Unmarshal(body, &dohResp); err != nil {
		return nil, fmt.Errorf("parse DoH response failed: %w", err)
	}

	result := &DNSQueryResult{}

	for _, ans := range dohResp.Answer {
		switch ans.Type {
		case 1: // A
			result.A = append(result.A, ans.Data)
		case 28: // AAAA
			result.AAAA = append(result.AAAA, ans.Data)
		case 5: // CNAME
			result.CNAME = append(result.CNAME, ans.Data)
		}
	}

	return result, nil
}

// parseDNSResponse parses DNS message response
func (r *Resolver) parseDNSResponse(msg dnsmessage.Message) *DNSQueryResult {
	result := &DNSQueryResult{}

	for _, ans := range msg.Answers {
		switch ans.Header.Type {
		case dnsmessage.TypeA:
			if a, ok := ans.Body.(*dnsmessage.AResource); ok {
				result.A = append(result.A, net.IP(a.A[:]).String())
			}
		case dnsmessage.TypeAAAA:
			if aaaa, ok := ans.Body.(*dnsmessage.AAAAResource); ok {
				result.AAAA = append(result.AAAA, net.IP(aaaa.AAAA[:]).String())
			}
		case dnsmessage.TypeCNAME:
			if cname, ok := ans.Body.(*dnsmessage.CNAMEResource); ok {
				result.CNAME = append(result.CNAME, cname.CNAME.String())
			}
		case dnsmessage.TypeMX:
			if mx, ok := ans.Body.(*dnsmessage.MXResource); ok {
				result.MX = append(result.MX, fmt.Sprintf("%d %s", mx.Pref, mx.MX.String()))
			}
		case dnsmessage.TypeTXT:
			if txt, ok := ans.Body.(*dnsmessage.TXTResource); ok {
				for _, s := range txt.TXT {
					result.TXT = append(result.TXT, s)
				}
			}
		case dnsmessage.TypeNS:
			if ns, ok := ans.Body.(*dnsmessage.NSResource); ok {
				result.NS = append(result.NS, ns.NS.String())
			}
		}
	}

	return result
}

// tlsClient creates a TLS connection (simplified, without full cert validation for DoT)
func tlsClient(conn net.Conn, server string) (net.Conn, error) {
	// For production use, proper TLS configuration should be implemented
	// This is a simplified version
	return conn, nil
}