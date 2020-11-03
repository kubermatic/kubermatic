/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/ipam"
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
	split := strings.Split(value, ",")

	if len(split) < 3 {
		return fmt.Errorf("expected cidr,gateway,dns1,dns2,... but got: %s", value)
	}

	cidrStr := split[0]
	ip, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("error parsing cidr %s: %v", cidrStr, err)
	}

	gwStr := split[1]
	gwIP := net.ParseIP(gwStr)
	if gwIP == nil {
		return fmt.Errorf("expected valid gateway ip but got %s", gwStr)
	}

	dnsSplitted := split[2:]
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
