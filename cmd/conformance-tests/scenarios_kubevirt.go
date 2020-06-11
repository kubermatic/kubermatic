package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"

	"go.uber.org/zap"
	utilpointer "k8s.io/utils/pointer"
)

// Returns a matrix of (version x operating system)
func getKubevirtScenarios(versions []*semver.Semver, log *zap.SugaredLogger) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
			logger: log,
		})
		// CentOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
			logger: log,
		})
	}

	return scenarios
}

type kubevirtScenario struct {
	version    *semver.Semver
	nodeOsSpec *apimodels.OperatingSystemSpec
	logger     *zap.SugaredLogger
}

func (s *kubevirtScenario) Name() string {
	return fmt.Sprintf("kubevirt-%s-%s", getOSNameFromSpec(*s.nodeOsSpec), s.version.String())
}

func (s *kubevirtScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					Kubevirt: &apimodels.KubevirtCloudSpec{
						Kubeconfig: secrets.Kubevirt.Kubeconfig,
					},
					DatacenterName: "kubevirt-europe-west3-c",
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *kubevirtScenario) NodeDeployments(num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	var sourceURL string

	switch {
	case s.nodeOsSpec.Ubuntu != nil:
		sourceURL = "http://10.109.79.210/ubuntu.img"
	case s.nodeOsSpec.Centos != nil:
		sourceURL = "http://10.109.79.210/centos.img"
	default:
		s.logger.Error("coreos operating system is not supported")
	}

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: utilpointer.Int32Ptr(int32(num)),
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Kubevirt: &apimodels.KubevirtNodeSpec{
							Memory:           utilpointer.StringPtr("1024M"),
							Namespace:        utilpointer.StringPtr("kube-system"),
							SourceURL:        utilpointer.StringPtr(sourceURL),
							StorageClassName: utilpointer.StringPtr("kubermatic-fast"),
							PVCSize:          utilpointer.StringPtr("10Gi"),
							CPUs:             utilpointer.StringPtr("1"),
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *kubevirtScenario) OS() apimodels.OperatingSystemSpec {
	return *s.nodeOsSpec
}
