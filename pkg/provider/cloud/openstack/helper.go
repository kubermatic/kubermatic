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

package openstack

import (
	"errors"
	"strings"

	"github.com/gophercloud/gophercloud"
)

const (
	subnetCIDR         = "192.168.1.0/24"
	subnetFirstAddress = "192.168.1.2"
	subnetLastAddress  = "192.168.1.254"

	defaultIPv6SubnetCIDR = "fd00::/64"

	resourceNamePrefix = "kubernetes-"
)

func isNotFoundErr(err error) bool {
	var errNotFound gophercloud.ErrDefault404

	return errors.As(err, &errNotFound) || strings.Contains(err.Error(), "not found")
}

func isEndpointNotFoundErr(err error) bool {
	var endpointNotFoundErr *gophercloud.ErrEndpointNotFound
	// left side of the || to catch any error returned as pointer to struct (current case of gophercloud)
	// right side of the || to catch any error returned as struct (in case...)
	return errors.As(err, &endpointNotFoundErr) || errors.As(err, &gophercloud.ErrEndpointNotFound{})
}
