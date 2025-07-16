package traefik_real_ip

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
)

// Config the plugin configuration.
type Config struct {
	TrustedIPs []string `json:"trustedIPs,omitempty" toml:"trustedIPs,omitempty" yaml:"trustedIPs,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		TrustedIPs: []string{},
	}
}

// IPRewriter is a plugin that blocks incoming requests depending on their source IP.
type IPRewriter struct {
	next       http.Handler
	name       string
	trustedIPs []*net.IPNet
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	ipRewriter := &IPRewriter{
		next: next,
		name: name,
	}

	for _, v := range config.TrustedIPs {
		_, excludedNet, err := net.ParseCIDR(v)
		if err != nil {
			return nil, err
		}

		ipRewriter.trustedIPs = append(ipRewriter.trustedIPs, excludedNet)
	}

	return ipRewriter, nil
}

func (r *IPRewriter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Println("[real ip plugin] error while getting remote address:", err)
		rw.WriteHeader(500)
		fmt.Fprintf(rw, "500 internal error")
		return
	}

	clientIP = removeIPv6Zone(clientIP)

	if trusted, err := r.trustedIP(clientIP); err == nil && trusted {
		err := r.rewriteIP(clientIP, rw, req)
		if err != nil {
			log.Println("[real ip plugin] error while overwriting remote address:", err)
			rw.WriteHeader(500)
			fmt.Fprintf(rw, "500 internal error")
			return
		}
	} else {
		req.Header.Set(xRealIP, clientIP)
		req.Header.Del(xForwardedFor)
	}

	r.next.ServeHTTP(rw, req)
}

func (r *IPRewriter) rewriteIP(clientIP string, rw http.ResponseWriter, req *http.Request) error {
	var forwardedIPs []string
	if xForwardedFor := req.Header.Get(xForwardedFor); xForwardedFor != "" {
		// Strict adherence requires separation with ", ", but some proxies appear to strip whitespace
		forwardedIPs = strings.Split(xForwardedFor, ",")
	}

	// note that traefik appends the IP onto X-Forwarded-For only after all middleware is parsed, so we should not add the IP ourselves

	index := -1
	for i := len(forwardedIPs) - 1; i >= 0; i-- {
		forwardedIPs[i] = strings.TrimSpace(forwardedIPs[i])
		trusted, err := r.trustedIP(forwardedIPs[i])

		if err != nil {
			return err
		}

		index = i
		if !trusted {
			break
		}
	}

	if index == -1 {
		req.Header.Del(xForwardedFor)
		req.Header.Set(xRealIP, clientIP)
	} else {
		req.Header.Set(xForwardedFor, strings.Join(forwardedIPs[index:], ", "))
		req.Header.Set(xRealIP, forwardedIPs[index])
	}

	return nil
}

func (r *IPRewriter) trustedIP(s string) (bool, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return false, errors.New("no ip specified")
	}

	for _, network := range r.trustedIPs {
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}

// https://github.com/traefik/traefik/blob/b1b4e6b918e8eeaf9e24823baf24dbc77f7d373e/pkg/middlewares/forwardedheaders/forwarded_header.go#L86
// Licensed under MIT
func removeIPv6Zone(clientIP string) string {
	if idx := strings.Index(clientIP, "%"); idx != -1 {
		return clientIP[:idx]
	}
	return clientIP
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
