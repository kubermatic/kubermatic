/*
Copyright 2019 The Machine Controller Authors.

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

//
// Google Cloud Provider for the Machine Controller
//

package gce

import (
	"strconv"

	"google.golang.org/api/compute/v1"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
)

// Possible instance statuses.
const (
	statusInstanceProvisioning = "PROVISIONING"
	statusInstanceRunning      = "RUNNING"
	statusInstanceStaging      = "STAGING"
	statusInstanceStopped      = "STOPPED"
	statusInstanceStopping     = "STOPPING"
	statusInstanceSuspended    = "SUSPENDED"
	statusInstanceSuspending   = "SUSPENDING"
	statusInstanceTerminated   = "TERMINATED"
)

// googleInstance implements instance.Instance for the Google compute instance.
type googleInstance struct {
	ci *compute.Instance
}

// Name implements instance.Instance.
func (gi *googleInstance) Name() string {
	return gi.ci.Name
}

// ID implements instance.Instance.
func (gi *googleInstance) ID() string {
	return strconv.FormatUint(gi.ci.Id, 10)
}

// Addresses implements instance.Instance.
func (gi *googleInstance) Addresses() []string {
	var addrs []string
	for _, ifc := range gi.ci.NetworkInterfaces {
		addrs = append(addrs, ifc.NetworkIP)
	}
	return addrs
}

// Status implements instance.Instance.
// TODO Check status mapping for staging, delet(ed|ing), suspend(ed|ing).
func (gi *googleInstance) Status() instance.Status {
	switch gi.ci.Status {
	case statusInstanceProvisioning:
		return instance.StatusCreating
	case statusInstanceRunning:
		return instance.StatusRunning
	case statusInstanceStaging:
		return instance.StatusCreating
	case statusInstanceStopped:
		return instance.StatusDeleted
	case statusInstanceStopping:
		return instance.StatusDeleting
	case statusInstanceSuspended:
		return instance.StatusDeleted
	case statusInstanceSuspending:
		return instance.StatusDeleting
	case statusInstanceTerminated:
		return instance.StatusDeleted
	}
	// Must not happen.
	return instance.StatusUnknown
}
