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

package anexia

import (
	"context"
	"errors"

	"go.anx.io/go-anxcloud/pkg/client"
	"go.anx.io/go-anxcloud/pkg/vsphere/provisioning/templates"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type Anexia struct {
	dc                *kubermaticv1.DatacenterSpecAnexia
	secretKeySelector provider.SecretKeySelectorValueFunc
}

var _ provider.CloudProvider = &Anexia{}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Anexia, error) {
	if dc.Spec.Anexia == nil {
		return nil, errors.New("datacenter is not an Anexia datacenter")
	}
	return &Anexia{
		dc:                dc.Spec.Anexia,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func (a *Anexia) DefaultCloudSpec(_ context.Context, _ *kubermaticv1.ClusterSpec) error {
	return nil
}

func (a *Anexia) ValidateCloudSpec(_ context.Context, spec kubermaticv1.CloudSpec) error {
	_, err := GetCredentialsForCluster(spec, a.secretKeySelector)

	return err
}

func (a *Anexia) InitializeCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (a *Anexia) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (a *Anexia) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (token string, err error) {
	accessToken := cloud.Anexia.Token

	if accessToken == "" {
		if cloud.Anexia.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		accessToken, err = secretKeySelector(cloud.Anexia.CredentialsReference, resources.AnexiaToken)
		if err != nil {
			return "", err
		}
	}

	return accessToken, nil
}

func ValidateCredentials(ctx context.Context, accessToken, locationID string) error {
	cli, err := getClient(accessToken)
	if err != nil {
		return err
	}
	t := templates.NewAPI(cli)
	_, err = t.List(ctx, locationID, "templates", 1, 1)
	return err
}

func getClient(token string) (client.Client, error) {
	tokenOpt := client.TokenFromString(token)
	return client.New(tokenOpt)
}
