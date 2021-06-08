package dhubratelimit

import (
	"net/http"
	"strings"
	"time"
)

const (
	headerLimitRemaining = "RateLimit-Remaining"
	headerLimitLimit     = "RateLimit-Limit"
	headerLimitResets    = "RateLimit-Reset"
)

type authToken struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	IssuedAt    time.Time `json:"issued_at"`
	Token       string    `json:"token"`
}

func (at *authToken) ExpiresAt() time.Time {
	return at.IssuedAt.Add(time.Duration(time.Duration(at.ExpiresIn).Seconds()))
}

func (at *authToken) ExpiresSoon() bool {
	return time.Until(at.ExpiresAt()) < 30*time.Second
}

func (at *authToken) ApplyTo(hh http.Header) {
	hh.Set("Authorization", strings.Join([]string{"Bearer", at.Token}, " "))
}
