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

package dc

import (
	"fmt"
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/email"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// GetDatacenter a function that gives you a single apiv1.Datacenter object.
func GetDatacenter(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, datacenterToGet string) (apiv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return apiv1.Datacenter{}, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	// Get the DCs and immediately filter out the ones restricted by e-mail domain if user is not admin
	dcs := getAPIDCsFromSeedMap(seeds)
	if !userInfo.IsAdmin {
		dcs, err = filterDCsByEmail(userInfo, dcs)
		if err != nil {
			return apiv1.Datacenter{}, utilerrors.New(http.StatusInternalServerError,
				fmt.Sprintf("failed to filter datacenters by email: %v", err))
		}
	}

	return filterDCsByName(dcs, datacenterToGet)
}

func filterDCsByName(dcs []apiv1.Datacenter, dcName string) (apiv1.Datacenter, error) {
	var foundDCs []apiv1.Datacenter
	for _, unfilteredDC := range dcs {
		if unfilteredDC.Metadata.Name == dcName {
			foundDCs = append(foundDCs, unfilteredDC)
		}
	}

	if n := len(foundDCs); n > 1 {
		return apiv1.Datacenter{}, fmt.Errorf("did not find one but %d datacenters for name %q", n, dcName)
	}
	if len(foundDCs) == 0 {
		return apiv1.Datacenter{}, utilerrors.NewNotFound("datacenter", dcName)
	}

	return foundDCs[0], nil
}

func filterDCsByEmail(userInfo *provider.UserInfo, list []apiv1.Datacenter) ([]apiv1.Datacenter, error) {
	var result []apiv1.Datacenter

	for _, dc := range list {
		matches, err := email.MatchesRequirements(userInfo.Email, dc.Spec.RequiredEmails)
		if err != nil {
			return nil, err
		}

		if matches {
			result = append(result, dc)
		}
	}

	return result, nil
}

func getAPIDCsFromSeedMap(seeds map[string]*kubermaticv1.Seed) []apiv1.Datacenter {
	var foundDCs []apiv1.Datacenter
	for _, seed := range seeds {
		foundDCs = append(foundDCs, getAPIDCsFromSeed(seed)...)
	}
	return foundDCs
}

// TODO(lsviben) - check if a "seed" dc is needed + if whole metadata is needed for DC, maybe we only need the name.
func getAPIDCsFromSeed(seed *kubermaticv1.Seed) []apiv1.Datacenter {
	var foundDCs []apiv1.Datacenter
	for datacenterName, datacenter := range seed.Spec.Datacenters {
		spec, err := ConvertInternalDCToExternalSpec(datacenter.DeepCopy(), seed.Name)
		if err != nil {
			log.Logger.Errorf("api spec error in dc %q: %v", datacenterName, err)
			continue
		}
		foundDCs = append(foundDCs, apiv1.Datacenter{
			Metadata: apiv1.DatacenterMeta{
				Name: datacenterName,
			},
			Spec: *spec,
		})
	}
	return foundDCs
}

func ConvertInternalDCToExternalSpec(dc *kubermaticv1.Datacenter, seedName string) (*apiv1.DatacenterSpec, error) {
	p, err := kubermaticv1helper.DatacenterCloudProviderName(dc.Spec.DeepCopy())
	if err != nil {
		return nil, err
	}

	nodeSettings := kubermaticv1.NodeSettings{}
	if dc.Node != nil {
		nodeSettings = *dc.Node
	}

	return &apiv1.DatacenterSpec{
		Seed:                           seedName,
		Location:                       dc.Location,
		Country:                        dc.Country,
		Provider:                       p,
		Node:                           nodeSettings,
		Digitalocean:                   dc.Spec.Digitalocean,
		AWS:                            dc.Spec.AWS,
		BringYourOwn:                   dc.Spec.BringYourOwn,
		Openstack:                      dc.Spec.Openstack,
		Hetzner:                        dc.Spec.Hetzner,
		VSphere:                        dc.Spec.VSphere,
		Azure:                          dc.Spec.Azure,
		Packet:                         dc.Spec.Packet,
		GCP:                            dc.Spec.GCP,
		Kubevirt:                       dc.Spec.Kubevirt,
		Alibaba:                        dc.Spec.Alibaba,
		Anexia:                         dc.Spec.Anexia,
		Nutanix:                        dc.Spec.Nutanix,
		VMwareCloudDirector:            dc.Spec.VMwareCloudDirector,
		Fake:                           dc.Spec.Fake,
		RequiredEmails:                 dc.Spec.RequiredEmails,
		EnforceAuditLogging:            dc.Spec.EnforceAuditLogging,
		EnforcePodSecurityPolicy:       dc.Spec.EnforcePodSecurityPolicy,
		DefaultOperatingSystemProfiles: dc.Spec.DefaultOperatingSystemProfiles,
		IPv6Enabled:                    dc.IsIPv6Enabled(kubermaticv1.ProviderType(p)),
	}, nil
}
