/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package ipam

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddressRange(t *testing.T) {
	testCases := []struct {
		subnet        string
		expectedFirst string
		expectedLast  string
	}{
		{
			subnet:        "192.168.0.0/16",
			expectedFirst: "192.168.0.0",
			expectedLast:  "192.168.255.255",
		},
		{
			subnet:        "192.168.0.0/17",
			expectedFirst: "192.168.0.0",
			expectedLast:  "192.168.127.255",
		},
		{
			subnet:        "fe80::/64",
			expectedFirst: "fe80::",
			expectedLast:  "fe80::ffff:ffff:ffff:ffff",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.subnet, func(t *testing.T) {
			_, subnet, _ := net.ParseCIDR(tc.subnet)
			firstIP, lastIP := addressRange(subnet)
			assert.Equal(t, tc.expectedFirst, firstIP.String())
			assert.Equal(t, tc.expectedLast, lastIP.String())
		})
	}
}

func TestNextSubnet(t *testing.T) {
	testCases := []struct {
		subnet             string
		expectedNextSubnet string
		expectToRollover   bool
	}{
		{
			subnet:             "9.255.255.0/24",
			expectedNextSubnet: "10.0.0.0/24",
			expectToRollover:   false,
		},
		{
			subnet:             "99.255.255.192/26",
			expectedNextSubnet: "100.0.0.0/26",
			expectToRollover:   false,
		},
		{
			subnet:             "255.255.255.192/26",
			expectedNextSubnet: "0.0.0.0/26",
			expectToRollover:   true,
		},
		{
			subnet:             "2001:db8:d000::/36",
			expectedNextSubnet: "2001:db8:e000::/36",
			expectToRollover:   false,
		},
		{
			subnet:             "ffff:ffff:ffff:ffff::/64",
			expectedNextSubnet: "::/64",
			expectToRollover:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.subnet, func(t *testing.T) {
			_, subnet, _ := net.ParseCIDR(tc.subnet)
			_, expectedNextSubnet, _ := net.ParseCIDR(tc.expectedNextSubnet)
			mask, _ := subnet.Mask.Size()
			nextSubnet, rollover := nextSubnet(subnet, mask)
			assert.Equal(t, expectedNextSubnet, nextSubnet)
			assert.Equal(t, tc.expectToRollover, rollover)
		})
	}
}

func TestIncIP(t *testing.T) {
	testCases := []struct {
		ip             string
		expectedNextIP string
	}{
		{ip: "0.0.0.0", expectedNextIP: "0.0.0.1"},
		{ip: "10.0.0.0", expectedNextIP: "10.0.0.1"},
		{ip: "9.255.255.255", expectedNextIP: "10.0.0.0"},
		{ip: "255.255.255.255", expectedNextIP: "0.0.0.0"},
		{ip: "::", expectedNextIP: "::1"},
		{ip: "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", expectedNextIP: "::"},
		{ip: "2001:db8:c001:ba00::", expectedNextIP: "2001:db8:c001:ba00::1"},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			expectedNextIP := net.ParseIP(tc.expectedNextIP)
			nextIP := incIP(ip)
			assert.Equal(t, expectedNextIP.String(), nextIP.String())
		})
	}
}
