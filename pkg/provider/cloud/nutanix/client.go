/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package nutanix

import (
	"errors"

	nutanixclient "github.com/terraform-providers/terraform-provider-nutanix/client"
	nutanixv3 "github.com/terraform-providers/terraform-provider-nutanix/client/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type ClientSet struct {
	Prism *nutanixv3.Client
}

func GetClientSet(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (*ClientSet, error) {
	return getClientSet(dc, cloud, secretKeyGetter)
}

func getCredentials(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (nutanixclient.Credentials, error) {
	username := cloud.Username
	password := cloud.Password

	var err error

	if username == "" {
		if cloud.CredentialsReference == nil {
			return nutanixclient.Credentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeyGetter(cloud.CredentialsReference, resources.NutanixUsername)
		if err != nil {
			return nutanixclient.Credentials{}, err
		}
	}

	if password == "" {
		if cloud.CredentialsReference == nil {
			return nutanixclient.Credentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeyGetter(cloud.CredentialsReference, resources.NutanixPassword)
		if err != nil {
			return nutanixclient.Credentials{}, err
		}
	}

	return nutanixclient.Credentials{
		URL:      dc.Endpoint,
		Insecure: dc.AllowInsecure,
		Username: username,
		Password: password,
	}, nil

}

func getClientSet(dc *kubermaticv1.DatacenterSpecNutanix, cloud *kubermaticv1.NutanixCloudSpec, secretKeyGetter provider.SecretKeySelectorValueFunc) (*ClientSet, error) {
	credentials, err := getCredentials(dc, cloud, secretKeyGetter)
	if err != nil {
		return nil, err
	}

	clientV3, err := nutanixv3.NewV3Client(credentials)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		Prism: clientV3,
	}, nil
}
