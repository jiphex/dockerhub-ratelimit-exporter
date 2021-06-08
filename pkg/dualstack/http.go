package dualstack

import (
	"net"
	"net/http"
	"time"
)

type AddressFamily int

const (
	IPv4 AddressFamily = iota
	IPv6
)

func FamilyTransport(family AddressFamily) http.RoundTripper {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:       30 * time.Second,
			KeepAlive:     30 * time.Second,
			DualStack:     false,
			FallbackDelay: -1,
			Resolver: &net.Resolver{
				
			},
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return t
}
