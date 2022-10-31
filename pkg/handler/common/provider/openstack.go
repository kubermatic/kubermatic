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
	"crypto/x509"
	"fmt"
	"strings"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func GetOpenStackFlavorSize(credentials *resources.OpenstackCredentials, authURL, region string,
	caBundle *x509.CertPool, flavorName string) (*apiv1.OpenstackSize, error) {
	flavors, err := openstack.GetFlavors(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, err
	}

	for _, flavor := range flavors {
		if strings.EqualFold(flavor.Name, flavorName) {
			return &apiv1.OpenstackSize{
				Slug:     flavor.Name,
				Memory:   flavor.RAM,
				VCPUs:    flavor.VCPUs,
				Disk:     flavor.Disk,
				Swap:     flavor.Swap,
				Region:   region,
				IsPublic: flavor.IsPublic,
			}, nil
		}
	}

	return nil, fmt.Errorf("cannot find openstack flavor %q size", flavorName)
}
