package main

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/ipam"
)

type networkFlags []ipam.Network

func (nf *networkFlags) String() string {
	var buf bytes.Buffer

	for i, n := range *nf {
		buf.WriteString(n.IP.String())
		buf.WriteString(",")
		buf.WriteString(n.Gateway.String())
		buf.WriteString(",")

		for iD, dns := range n.DNSServers {
			buf.WriteString(dns.String())

			if iD < len(n.DNSServers)-1 {
				buf.WriteString(",")
			}
		}

		if i < len(*nf)-1 {
			buf.WriteString(";")
		}
	}

	return buf.String()
}

func (nf *networkFlags) Set(value string) error {
	splitted := strings.Split(value, ",")

	if len(splitted) < 3 {
		return fmt.Errorf("expected cidr,gateway,dns1,dns2,... but got: %s", value)
	}

	cidrStr := splitted[0]
	ip, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("error parsing cidr %s: %v", cidrStr, err)
	}

	gwStr := splitted[1]
	gwIP := net.ParseIP(gwStr)
	if gwIP == nil {
		return fmt.Errorf("expected valid gateway ip but got %s", gwStr)
	}

	dnsSplitted := splitted[2:]
	dnsServers := make([]net.IP, len(dnsSplitted))
	for i, d := range dnsSplitted {
		dnsIP := net.ParseIP(d)
		if dnsIP == nil {
			return fmt.Errorf("expected valid dns ip but got %s", d)
		}

		dnsServers[i] = dnsIP
	}

	val := ipam.Network{
		IP:         ip,
		IPNet:      *ipnet,
		Gateway:    gwIP,
		DNSServers: dnsServers,
	}

	*nf = append(*nf, val)
	return nil
}
