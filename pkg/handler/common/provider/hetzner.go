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

package provider

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

func GetHetznerServerType(ctx context.Context, token string, serverTypeName string) (*hcloud.ServerType, error) {
	if token == "" {
		return nil, fmt.Errorf("hetzner token cannot be empty")
	} else if serverTypeName == "" {
		return nil, fmt.Errorf("invalid hetzner serverTypeName:%v", serverTypeName)
	}
	hClient := hcloud.NewClient(hcloud.WithToken(token))
	serverType, _, err := hClient.ServerType.Get(ctx, serverTypeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get hetzner server type, error: %w", err)
	}
	return serverType, nil
}
