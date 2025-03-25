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

package digitalocean

import (
	"context"
	"errors"
	"fmt"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type digitalocean struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &digitalocean{
		secretKeySelector: secretKeyGetter,
	}
}

var _ provider.CloudProvider = &digitalocean{}

func (do *digitalocean) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	return nil
}

func getClient(ctx context.Context, token string) *godo.Client {
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := godo.NewClient(oauth2.NewClient(ctx, static))

	return client
}

func ValidateCredentials(ctx context.Context, token string) error {
	_, _, err := getClient(ctx, token).Regions.List(ctx, nil)
	return err
}

func (do *digitalocean) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	token, err := GetCredentialsForCluster(spec, do.secretKeySelector)
	if err != nil {
		return err
	}

	return ValidateCredentials(ctx, token)
}

func (do *digitalocean) InitializeCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (do *digitalocean) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (do *digitalocean) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessToken string, err error) {
	accessToken = cloud.Digitalocean.Token

	if accessToken == "" {
		if cloud.Digitalocean.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		accessToken, err = secretKeySelector(cloud.Digitalocean.CredentialsReference, resources.DigitaloceanToken)
		if err != nil {
			return "", err
		}
	}

	return accessToken, nil
}

func DescribeDropletSize(ctx context.Context, token, sizeName string) (*provider.NodeCapacity, error) {
	sizes, _, err := getClient(ctx, token).Sizes.List(ctx, &godo.ListOptions{PerPage: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to list droplet sizes: %w", err)
	}

	for _, godosize := range sizes {
		if godosize.Slug == sizeName {
			capacity := provider.NewNodeCapacity()
			capacity.WithCPUCount(godosize.Vcpus)

			if err := capacity.WithMemory(godosize.Memory, "M"); err != nil {
				return nil, fmt.Errorf("failed to parse memory size: %w", err)
			}

			if err := capacity.WithStorage(godosize.Disk, "G"); err != nil {
				return nil, fmt.Errorf("failed to parse disk size: %w", err)
			}

			return capacity, nil
		}
	}

	return nil, fmt.Errorf("droplet size %q not found", sizeName)
}
