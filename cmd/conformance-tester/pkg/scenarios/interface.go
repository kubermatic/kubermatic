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
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type baseScenario struct {
	version          *semver.Semver
	containerRuntime string
	datacenter       *kubermaticv1.Datacenter
	osSpec           apimodels.OperatingSystemSpec
}

func (s *baseScenario) DeepCopy() *baseScenario {
	version := s.version.DeepCopy()

	return &baseScenario{
		version:          &version,
		containerRuntime: s.containerRuntime,
		datacenter:       s.datacenter.DeepCopy(),
		osSpec:           s.osSpec,
	}
}

func (s *baseScenario) Version() *semver.Semver {
	return s.version
}

func (s *baseScenario) Datacenter() *kubermaticv1.Datacenter {
	return s.datacenter
}

func (s *baseScenario) ContainerRuntime() string {
	return s.containerRuntime
}

func (s *baseScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}

type Scenario interface {
	Name() string
	OS() apimodels.OperatingSystemSpec
	Version() *semver.Semver
	Datacenter() *kubermaticv1.Datacenter
	ContainerRuntime() string
	Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec
	APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec
	MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error)
	NodeDeployments(ctx context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error)
}

//gocyclo:ignore
func GetScenarios(ctx context.Context, opts *types.Options, log *zap.SugaredLogger) ([]Scenario, error) {
	scenarioOptions := strings.Split(opts.ScenarioOptions, ",")

	var scenarios []Scenario
	if opts.Providers.Has("aws") {
		log.Info("Adding AWS scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.AWS.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetAWSScenarios(opts.Versions, opts.KubermaticClient, opts.KubermaticAuthenticator, dc)...)
	}

	if opts.Providers.Has("digitalocean") {
		log.Info("Adding Digitalocean scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Digitalocean.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetDigitaloceanScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("hetzner") {
		log.Info("Adding Hetzner scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Hetzner.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetHetznerScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("openstack") {
		log.Info("Adding OpenStack scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.OpenStack.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetOpenStackScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("vsphere") {
		log.Info("Adding vSphere scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.VSphere.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetVSphereScenarios(scenarioOptions, opts.Versions, dc)...)
	}

	if opts.Providers.Has("azure") {
		log.Info("Adding Azure scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Azure.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetAzureScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("packet") {
		log.Info("Adding Packet scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Packet.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetPacketScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("gcp") {
		log.Info("Adding GCP scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.GCP.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetGCPScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("kubevirt") {
		log.Info("Adding Kubevirt scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Kubevirt.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetKubevirtScenarios(opts.Versions, log, dc)...)
	}

	if opts.Providers.Has("alibaba") {
		log.Info("Adding Alibaba scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Alibaba.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetAlibabaScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("anexia") {
		log.Info("Adding Anexia scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Anexia.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetAnexiaScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("nutanix") {
		log.Info("Adding Nutanix scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.Nutanix.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetNutanixScenarios(opts.Versions, dc)...)
	}

	if opts.Providers.Has("vmwareclouddirector") {
		log.Info("Adding VMware Cloud Director scenarios")
		dc, err := getDatacenter(ctx, opts.SeedClusterClient, opts.Secrets.VMwareCloudDirector.KKPDatacenter)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, GetVMwareCloudDirectorScenarios(opts.Versions, dc)...)
	}

	var filteredScenarios []Scenario
	for _, scenario := range scenarios {
		if isValidScenario(opts, log, scenario) {
			filteredScenarios = append(filteredScenarios, scenario)
		}
	}

	// Shuffle scenarios - avoids timeouts caused by quota issues
	return shuffle(filteredScenarios), nil
}

func isValidScenario(opts *types.Options, log *zap.SugaredLogger, scenario Scenario) bool {
	// check if the CRI is enabled by the user
	cri := scenario.ContainerRuntime()
	if !opts.ContainerRuntimes.Has(scenario.ContainerRuntime()) {
		log.Debugw("Skipping scenario because this CRI is not enabled.", "cri", cri, "scenario", scenario.Name())
		return false
	}

	// check if the OS is enabled by the user
	os := getOSNameFromSpec(scenario.OS())
	if !opts.Distributions.Has(string(os)) {
		log.Debugw("Skipping scenario because this OS is not enabled.", "os", os, "scenario", scenario.Name())
		return false
	}

	// apply static filters
	clusterVersion := scenario.Version()
	dockerSupported := clusterVersion.LessThan(semver.NewSemverOrDie("1.24"))
	if !dockerSupported && cri == resources.ContainerRuntimeDocker {
		log.Infow("Skipping scenario because CRI is not supported in this Kubernetes version", "cri", cri, "version", clusterVersion, "scenario", scenario.Name())
		return false
	}

	return true
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

func createMachineDeployment(replicas int, version *semver.Semver, os providerconfig.OperatingSystem, osSpec interface{}, provider providerconfig.CloudProvider, providerSpec interface{}) (clusterv1alpha1.MachineDeployment, error) {
	replicas32 := int32(replicas)

	encodedOSSpec, err := json.Marshal(osSpec)
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	encodedCloudProviderSpec, err := json.Marshal(providerSpec)
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	cfg := providerconfig.Config{
		CloudProvider: provider,
		CloudProviderSpec: runtime.RawExtension{
			Raw: encodedCloudProviderSpec,
		},
		OperatingSystem: os,
		OperatingSystemSpec: runtime.RawExtension{
			Raw: encodedOSSpec,
		},
	}

	encodedConfig, err := json.Marshal(cfg)
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	machineLabels := map[string]string{
		"machine": "md-" + utilrand.String(5),
	}

	return clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-" + utilrand.String(5),
			Namespace: metav1.NamespaceSystem,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Replicas: &replicas32,
			Selector: metav1.LabelSelector{
				MatchLabels: machineLabels,
			},
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: machineLabels,
				},
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: version.String(),
					},
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: encodedConfig,
						},
					},
				},
			},
		},
	}, nil
}

func getDatacenter(ctx context.Context, client ctrlruntimeclient.Client, datacenter string) (*kubermaticv1.Datacenter, error) {
	seeds := &kubermaticv1.SeedList{}
	if err := client.List(ctx, seeds); err != nil {
		return nil, fmt.Errorf("failed to list seeds: %w", err)
	}

	for _, seed := range seeds.Items {
		for name, dc := range seed.Spec.Datacenters {
			if name == datacenter {
				return &dc, nil
			}
		}
	}

	return nil, fmt.Errorf("no Seed contains datacenter %q", datacenter)
}
