package extip

import "net"

type AddressFamily int

const (
	Auto AddressFamily = iota
	IPv4
	IPv6
)

type IPSource func(af AddressFamily) (ip net.IP, err error)
