package dhubratelimit

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"
)

type InnerResult struct {
	PullLimit     int           `json:"pull_limit"`
	PullRemaining int           `json:"pull_remaining"`
	CheckTime     time.Time     `json:"checked_at"`
	Window        time.Duration `json:"-"`
	Authenticated bool          `json:"authenticated"`
}

type Result interface {
	GetLimit() int
	GetRemaining() int
	GetWindow() int
	GetIdentity() string
	MarshalJSON() ([]byte, error)
}

func (r *InnerResult) GetLimit() int {
	return r.PullLimit
}

func (r *InnerResult) GetRemaining() int {
	return r.PullLimit
}

func (r *InnerResult) GetWindow() int {
	return int(r.Window)
}

// This is to ensure the interface is implemented
var _ Result = &AuthResult{}

type AuthResult struct {
	InnerResult
	Username string
}

func (ar *AuthResult) GetIdentity() string {
	return ar.Username
}

func (ur *AuthResult) MarshalJSON() ([]byte, error) {
	type out struct {
		InnerResult
		Username string `json:"username"`
	}
	x := out{
		InnerResult: ur.InnerResult,
		Username:    ur.Username,
	}
	return json.Marshal(x)
}

// This is to ensure the interface is implemented
var _ Result = &UnauthResult{}

type UnauthResult struct {
	InnerResult
	ipAddress net.IP
}

func (ur *UnauthResult) MarshalJSON() ([]byte, error) {
	type out struct {
		InnerResult
		IPAddress string `json:"ip_address"`
		Family    string `json:"ip_family"`
	}
	fam := "unknown"
	if len(ur.ipAddress) == 16 {
		fam = "inet"
	} else if len(ur.ipAddress) == 128 {
		fam = "inet6"
	}
	return json.Marshal(out{
		InnerResult: ur.InnerResult,
		IPAddress:   ur.ipAddress.String(),
		Family:      fam,
	})
}

func (ur *UnauthResult) GetIdentity() string {
	return ur.ipAddress.String()
}

func splitRatelimitHeader(in string) (int, int) {
	p := strings.SplitN(in, ";w=", 2)
	lim, _ := strconv.Atoi(p[0])
	win, _ := strconv.Atoi(p[1])
	return lim, win
}
