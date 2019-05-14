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
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Interval and timeout for polling.
	pollInterval = 5 * time.Second
	pollTimeout  = 5 * time.Minute
)

// Google compute operation status.
const (
	statusDone = "DONE"
)

const (
	defaultNetwork = "global/networks/default"
)

// service wraps a GCE compute service for the extension with helper methods.
type service struct {
	*compute.Service
}

// connectComputeService establishes a service connection to the Compute Engine.
func connectComputeService(cfg *config) (*service, error) {
	svc, err := compute.New(cfg.jwtConfig.Client(oauth2.NoContext))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Google Cloud: %v", err)
	}
	return &service{svc}, nil
}

// networkInterfaces returns the configured network interfaces for an instance creation.
func (svc *service) networkInterfaces(cfg *config) ([]*compute.NetworkInterface, error) {
	network := cfg.network

	if cfg.network == "" && cfg.subnetwork == "" {
		network = defaultNetwork
	}

	ifc := &compute.NetworkInterface{
		Network:    network,
		Subnetwork: cfg.subnetwork,
	}

	if cfg.assignPublicIPAddress {
		ifc.AccessConfigs = []*compute.AccessConfig{{
			Name: "External NAT",
			Type: "ONE_TO_ONE_NAT",
		}}
	}

	return []*compute.NetworkInterface{ifc}, nil
}

// attachedDisks returns the configured attached disks for an instance creation.
func (svc *service) attachedDisks(cfg *config) ([]*compute.AttachedDisk, error) {
	sourceImage, err := cfg.sourceImageDescriptor()
	if err != nil {
		return nil, err
	}
	bootDisk := &compute.AttachedDisk{
		Boot:       true,
		AutoDelete: true,
		InitializeParams: &compute.AttachedDiskInitializeParams{
			DiskSizeGb:  cfg.diskSize,
			DiskType:    cfg.diskTypeDescriptor(),
			SourceImage: sourceImage,
		},
	}
	return []*compute.AttachedDisk{bootDisk}, nil
}

// waitZoneOperation waits for a GCE operation in a zone to be completed or timed out.
func (svc *service) waitZoneOperation(cfg *config, opName string) error {
	return svc.waitOperation(func() (*compute.Operation, error) {
		return svc.ZoneOperations.Get(cfg.projectID, cfg.zone, opName).Do()
	})
}

// waitOperation waits for a GCE operation to be completed or timed out.
func (svc *service) waitOperation(refreshOperation func() (*compute.Operation, error)) error {
	var op *compute.Operation
	var err error

	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		op, err = refreshOperation()
		if err != nil {
			return false, err
		}
		// Check if done (successfully).
		if op.Status == statusDone {
			if op.Error != nil {
				// Operation failed.
				return false, fmt.Errorf("GCE operation failed: %v", *op.Error.Errors[0])
			}
			return true, nil
		}
		// Not yet done.
		return false, nil
	})
}
