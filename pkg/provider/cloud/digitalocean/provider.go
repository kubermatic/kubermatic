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

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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

func (do *digitalocean) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (do *digitalocean) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	token, err := GetCredentialsForCluster(spec, do.secretKeySelector)
	if err != nil {
		return err
	}

	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := godo.NewClient(oauth2.NewClient(context.Background(), static))

	_, _, err = client.Regions.List(context.Background(), nil)
	return err
}

func (do *digitalocean) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (do *digitalocean) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (do *digitalocean) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
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
