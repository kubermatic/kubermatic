/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package scenarios

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

type Scenario interface {
	Name() string
	Cluster(secrets types.Secrets) *apimodels.CreateClusterSpec
	NodeDeployments(ctx context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error)
	OS() apimodels.OperatingSystemSpec
}

func GetScenarios(opts *types.Options, log *zap.SugaredLogger) []Scenario {
	scenarioOptions := strings.Split(opts.ScenarioOptions, ",")

	var scenarios []Scenario
	if opts.Providers.Has("aws") {
		log.Info("Adding AWS scenarios")
		scenarios = append(scenarios, GetAWSScenarios(opts.Versions, opts.KubermaticClient, opts.KubermaticAuthenticator)...)
	}
	if opts.Providers.Has("digitalocean") {
		log.Info("Adding Digitalocean scenarios")
		scenarios = append(scenarios, GetDigitaloceanScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("hetzner") {
		log.Info("Adding Hetzner scenarios")
		scenarios = append(scenarios, GetHetznerScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("openstack") {
		log.Info("Adding OpenStack scenarios")
		scenarios = append(scenarios, GetOpenStackScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("vsphere") {
		log.Info("Adding vSphere scenarios")
		scenarios = append(scenarios, GetVSphereScenarios(scenarioOptions, opts.Versions)...)
	}
	if opts.Providers.Has("azure") {
		log.Info("Adding Azure scenarios")
		scenarios = append(scenarios, GetAzureScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("packet") {
		log.Info("Adding Packet scenarios")
		scenarios = append(scenarios, GetPacketScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("gcp") {
		log.Info("Adding GCP scenarios")
		scenarios = append(scenarios, GetGCPScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("kubevirt") {
		log.Info("Adding Kubevirt scenarios")
		scenarios = append(scenarios, GetKubevirtScenarios(opts.Versions, log)...)
	}
	if opts.Providers.Has("alibaba") {
		log.Info("Adding Alibaba scenarios")
		scenarios = append(scenarios, GetAlibabaScenarios(opts.Versions)...)
	}
	if opts.Providers.Has("nutanix") {
		log.Info("Adding Nutanix scenarios")
		scenarios = append(scenarios, GetNutanixScenarios(opts.Versions)...)
	}

	hasDistribution := func(distribution providerconfig.OperatingSystem) bool {
		return opts.Distributions.Has(string(distribution))
	}

	var filteredScenarios []Scenario
	for _, scenario := range scenarios {
		osspec := scenario.OS()
		if osspec.Ubuntu != nil && hasDistribution(providerconfig.OperatingSystemUbuntu) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Flatcar != nil && hasDistribution(providerconfig.OperatingSystemFlatcar) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Centos != nil && hasDistribution(providerconfig.OperatingSystemCentOS) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Sles != nil && hasDistribution(providerconfig.OperatingSystemSLES) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
		if osspec.Rhel != nil && hasDistribution(providerconfig.OperatingSystemRHEL) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
	}

	// Shuffle scenarios - avoids timeouts caused by quota issues
	return shuffle(filteredScenarios)
}

func shuffle(vals []Scenario) []Scenario {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]Scenario, len(vals))
	n := len(vals)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(vals))
		ret[i] = vals[randIndex]
		vals = append(vals[:randIndex], vals[randIndex+1:]...)
	}
	return ret
}
