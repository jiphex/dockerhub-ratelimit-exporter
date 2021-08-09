package dhubratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dafnifacility/dockerhub-ratelimit-exporter/pkg/extip"
)

const (
	repoURL  = "https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest"
	tokenURL = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull"
)

type Checker struct {
	ipsrc              extip.IPSource
	username, password string
	token              *authToken
	cacheip            *net.IP
}

type Option func(*Checker) error

func WithCredentials(username, password string) Option {
	return func(r *Checker) error {
		r.username = username
		r.password = password
		return nil
	}
}

func WithIPSource(label string) Option {
	return func(r *Checker) error {
		switch label {
		case "ipify":
			r.ipsrc = extip.IPify
		case "myipio":
			r.ipsrc = extip.MyIPIO
		case "icanhazip":
			fallthrough
		default:
			r.ipsrc = extip.ICanHazIP
		}
		return nil
	}
}

func (rlc *Checker) IPAddress(family extip.AddressFamily) net.IP {
	if rlc.cacheip != nil {
		return *rlc.cacheip
	}
	ip, err := rlc.ipsrc(family)
	if err != nil {
		return ip
	}
	rlc.cacheip = &ip
	return ip
}

func (rlc *Checker) HasCredentials() bool {
	return rlc.username != "" || rlc.password != ""
}

func (rlc *Checker) HasIdentity() bool {
	return (rlc.cacheip != nil && len(*rlc.cacheip) > 0) || rlc.HasCredentials()
}

func (rlc *Checker) IdentityString() string {
	if !rlc.HasIdentity() {
		return "unknown"
	} else {
		if rlc.HasCredentials() {
			return fmt.Sprintf("auth:%s", rlc.username)
		} else {
			return fmt.Sprintf("unauth:%s", rlc.IPAddress(extip.IPv4).String())
		}
	}
}

// func (rlc *Checker) registryImageLimitRequest() *http.Request {
// 	req, err := http.NewRequest(http.MethodHead, repoURL, nil)
// 	if err != nil {
// 		panic("failed to generate request from static data")
// 	}
// 	return req
// }

func (rlc *Checker) getAuthToken(ctx context.Context) (at authToken, err error) {
	if rlc.token != nil && !rlc.token.ExpiresSoon() {
		// If we already have an unexpired token, return that instead of hammering dockerhub
		return *rlc.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return
	}
	if rlc.HasCredentials() {
		log.Debug("using dockerhub auth credentials")
		req.SetBasicAuth(rlc.username, rlc.password)
	} else {
		log.Debug("using dockerhub without credentials")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	un := json.NewDecoder(res.Body)
	// Decode from the decoder (res.Body) into at, then return with either a token or error
	err = un.Decode(&at)
	if err == nil {
		log.WithField("expires", at.ExpiresAt()).Debug("got auth token from Docker Hub")
		rlc.token = &at
	}
	return
}

func (rlc *Checker) Check(ctx context.Context) (lim Result, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, repoURL, nil)
	if err != nil {
		return
	}
	tok, err := rlc.getAuthToken(ctx)
	if err != nil {
		return
	}
	tok.ApplyTo(req.Header)
	// FIXME: This is inconsistent on dualstack hosts, the IP address retrieved at the start might not necessarily be the same as the one we get back
	hres, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	res := InnerResult{
		CheckTime:     time.Now(),
		Authenticated: rlc.HasCredentials(),
	}
	var window int
	if lr := hres.Header.Get(headerLimitRemaining); lr != "" {
		res.PullRemaining, window = splitRatelimitHeader(lr)
	}
	if ll := hres.Header.Get(headerLimitLimit); ll != "" {
		res.PullLimit, window = splitRatelimitHeader(ll)
	}
	res.Window = time.Duration(window * int(time.Second))
	if rlc.HasCredentials() {
		return &AuthResult{
			InnerResult: res,
			Username:    rlc.username,
		}, nil
	} else {
		return &UnauthResult{
			InnerResult: res,
			ipAddress:   rlc.IPAddress(extip.IPv4),
		}, nil
	}
}

func NewChecker(opts ...Option) (*Checker, error) {
	rlc := &Checker{}
	for _, o := range opts {
		err := o(rlc)
		if err != nil {
			return nil, err
		}
	}
	if rlc == nil {
		log.Fatal("nil rlc?")
	}
	// If we don't have any credentials then we fallback to filling out unauthIP
	if !rlc.HasIdentity() {
		rlc.ipsrc = extip.ICanHazIP
		// rlc.ipsrc = extip.IPify
	}
	return rlc, nil
}
