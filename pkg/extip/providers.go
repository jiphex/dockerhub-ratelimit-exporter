package extip

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func ICanHazIP(af AddressFamily) (ip net.IP, err error) {
	return GenericIPService(af, "icanhazip.com", map[AddressFamily]string{
		Auto: "",
		IPv6: "ipv6.",
		IPv4: "ipv4.",
	}, "")
}

func IPify(af AddressFamily) (ip net.IP, err error) {
	return GenericIPService(af, "ipify.org", map[AddressFamily]string{
		IPv4: "api.",
		IPv6: "api6.",
		Auto: "api64.",
	}, "")
}

func MyIPIO(af AddressFamily) (ip net.IP, err error) {
	return GenericIPService(af, "my-ip.io", map[AddressFamily]string{
		IPv4: "api.",
		IPv6: "api6.",
		Auto: "api4.",
	}, "ip")
}

func GenericIPService(af AddressFamily, domain string, afSubdomains map[AddressFamily]string, path string) (ip net.IP, err error) {
	url := fmt.Sprintf("https://%s%s/%s", afSubdomains[af], domain, path)
	res, err := http.Get(url)
	if err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		return ip, fmt.Errorf("bad HTTP response status: %s", res.Status)
	}
	bout, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	sip := strings.TrimSpace(string(bout))
	ip = net.ParseIP(sip)
	if ip == nil {
		return ip, fmt.Errorf("unable to parse IP address: %s - len=%d", bout, len(ip))
	} else {
		return ip, nil
	}
}
