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

package vsphere

import (
	"context"
	"fmt"
	"net/url"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

type Session struct {
	Client     *govmomi.Client
	Finder     *find.Finder
	Datacenter *object.Datacenter
}

// NewSession creates a vCenter client with initialized finder
func NewSession(ctx context.Context, config *Config) (*Session, error) {
	clientURL, err := url.Parse(fmt.Sprintf("%s/sdk", config.VSphereURL))
	if err != nil {
		return nil, err
	}
	clientURL.User = url.UserPassword(config.Username, config.Password)

	client, err := govmomi.NewClient(ctx, clientURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to build client: %v", err)
	}

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(ctx, config.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere datacenter: %v", err)
	}
	finder.SetDatacenter(dc)

	return &Session{
		Client:     client,
		Finder:     finder,
		Datacenter: dc,
	}, nil
}

// Logout closes the idling vCenter connections
func (s *Session) Logout() {
	if err := s.Client.Logout(context.Background()); err != nil {
		utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", err))
	}
}
