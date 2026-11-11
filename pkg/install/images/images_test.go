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

package images

import (
	"context"
	"fmt"
	"strings"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
)

// These tests do not just verify that the Docker images can be extracted
// properly, but also that all addons can be processed for each cluster
// combination (i.e. there are no broken Go templates). To save save, this
// test only tests each minor Kubernetes release, not every single configured
// patch version.
// When working on this test, remember that it also tests pkg/addon/'s code.

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.NewLogrus()

	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("failed to determine versions: %v", err)
	}

	kubermaticVersions := kubermatic.GetFakeVersions()
	clusterVersions := getLatestMinorVersions(config)

	caBundle, err := certificates.NewCABundleFromFile("../../../charts/kubermatic-operator/static/ca-bundle.pem")
	if err != nil {
		t.Fatalf("failed to load CA bundle: %v", err)
	}

	allAddons, err := addonutil.LoadAddonsFromDirectory("../../../addons")
	if err != nil {
		t.Fatalf("failed to load addons: %v", err)
	}

	cloudSpecs := GetCloudSpecs()
	cniPlugins := GetCNIPlugins()

	imageSet := sets.New[string]()
	for _, clusterVersion := range clusterVersions {
		vlog := log.WithField("version", clusterVersion.Version.String())

		for _, cloudSpec := range cloudSpecs {
			plog := vlog.WithField("provider", cloudSpec.ProviderName)

			for _, cniPlugin := range cniPlugins {
				cni := fmt.Sprintf("%s_%s", cniPlugin.Type, cniPlugin.Version)
				clog := plog.WithField("cni", cni)

				images, err := GetImagesForVersion(clog, clusterVersion, cloudSpec, cniPlugin, false, config, allAddons, kubermaticVersions, caBundle, "")
				if err != nil {
					t.Errorf("Error calling getImagesForVersion for %s / %s / %s: %v", cloudSpec.ProviderName, clusterVersion.Version.String(), cni, err)
				}
				imageSet.Insert(images...)
			}
		}
	}

	if _, _, err := CopyImages(context.Background(), log, true, true, sets.List(imageSet), "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}

// TestGetImagesForVersionIncludesRuntimeImages checks images that KKP only uses in
// Jobs or provider specific workloads, which are easy to miss in the image list.
func TestGetImagesForVersionIncludesRuntimeImages(t *testing.T) {
	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("Failed to default KubermaticConfiguration: %v", err)
	}

	clusterVersion := &version.Version{Version: config.Spec.Versions.Default.Semver()}
	clusterSemver := *config.Spec.Versions.Default

	caBundle, err := certificates.NewCABundleFromFile("../../../charts/kubermatic-operator/static/ca-bundle.pem")
	if err != nil {
		t.Fatalf("Failed to load CA bundle: %v", err)
	}

	cniPlugin := &kubermaticv1.CNIPluginSettings{
		Type:    kubermaticv1.CNIPluginTypeCilium,
		Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium),
	}

	allAddons, err := addonutil.LoadAddonsFromDirectory("../../../addons")
	if err != nil {
		t.Fatalf("Failed to load addons: %v", err)
	}

	utilImage := config.Spec.Util.DockerRepository + ":" + resources.UtilVersion
	kubeOneImage := resources.KubeOneImage + ":" + resources.KubeOneImageTag

	testCases := []struct {
		provider kubermaticv1.ProviderType
		expected []string
		// repositories matched without a tag, for addon images that are bumped often
		expectedRepos []string
	}{
		{
			// util is shared by the KubeOne Jobs, the seed-proxy and the encryption Job, so
			// this only proves util is listed, not which of them listed it
			provider: kubermaticv1.BringYourOwnCloudProvider,
			expected: []string{utilImage, kubeOneImage},
		},
		{
			provider:      kubermaticv1.AWSCloudProvider,
			expected:      []string{resources.RegistryK8S + "/provider-aws/cloud-controller-manager:" + cloudcontroller.AWSCCMVersion(clusterSemver)},
			expectedRepos: []string{"public.ecr.aws/ebs-csi-driver/aws-ebs-csi-driver"},
		},
		{
			provider: kubermaticv1.DigitaloceanCloudProvider,
			expected: []string{resources.RegistryDocker + "/digitalocean/digitalocean-cloud-controller-manager:" + cloudcontroller.DigitaloceanCCMVersion(clusterSemver)},
		},
		{
			provider: kubermaticv1.GCPCloudProvider,
			expected: []string{resources.RegistryK8S + "/cloud-provider-gcp/cloud-controller-manager:" + cloudcontroller.GCPCCMVersion(clusterSemver)},
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.provider), func(t *testing.T) {
			cloudSpecs := GetFilteredCloudSpecs(sets.New(string(tc.provider)))
			if len(cloudSpecs) != 1 {
				t.Fatalf("Expected 1 cloud spec for %q, got %d", tc.provider, len(cloudSpecs))
			}

			images, err := GetImagesForVersion(logrus.New(), clusterVersion, cloudSpecs[0], cniPlugin, true, config, allAddons, kubermatic.GetFakeVersions(), caBundle, "")
			if err != nil {
				t.Fatalf("Failed to get images: %v", err)
			}

			imageSet := sets.New(images...)
			for _, image := range tc.expected {
				if !imageSet.Has(image) {
					t.Errorf("Expected image %q to be listed, but it was not", image)
				}
			}

			for _, repo := range tc.expectedRepos {
				found := false
				for _, image := range images {
					if strings.HasPrefix(image, repo+":") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected an image from %q to be listed, but there was none", repo)
				}
			}
		})
	}
}

func getLatestMinorVersions(config *kubermaticv1.KubermaticConfiguration) []*version.Version {
	latestMinorVersions := version.GetLatestMinorVersions(config.Spec.Versions.Versions)

	result := make([]*version.Version, len(latestMinorVersions))
	for i, ver := range latestMinorVersions {
		result[i] = &version.Version{
			Version: semverlib.MustParse(ver),
		}
	}

	return result
}

func TestGetCloudSpecsComplete(t *testing.T) {
	cloudSpecs := GetCloudSpecs()

	for _, providerName := range kubermaticv1.SupportedProviders {
		// ignore external providers
		if providerName == kubermaticv1.EKSCloudProvider || providerName == kubermaticv1.AKSCloudProvider || providerName == kubermaticv1.GKECloudProvider {
			continue
		}

		// ignore fake
		if providerName == kubermaticv1.FakeCloudProvider {
			continue
		}

		exists := false
		for _, spec := range cloudSpecs {
			if spec.ProviderName == string(providerName) {
				exists = true
				break
			}
		}

		if !exists {
			t.Errorf("GetCloudSpecs does not contain data for %q.", providerName)
		}
	}
}

func TestFilterCloudSpecs(t *testing.T) {
	testCases := []struct {
		name           string
		providerFilter sets.Set[string]
		expectAll      bool
	}{
		{
			name:           "nil filter returns all specs",
			providerFilter: nil,
			expectAll:      true,
		},
		{
			name:           "empty filter returns all specs",
			providerFilter: sets.New[string](),
			expectAll:      true,
		},
		{
			name:           "filter with AWS only",
			providerFilter: sets.New("aws"),
			expectAll:      false,
		},
		{
			name:           "filter with multiple providers",
			providerFilter: sets.New("aws", "azure", "kubevirt"),
			expectAll:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetFilteredCloudSpecs(tc.providerFilter)

			if tc.expectAll {
				// Should return all cloud specs (15 total as of this writing)
				if len(result) < 10 {
					t.Errorf("expected all cloud specs (at least 10), got %d", len(result))
				}
			} else {
				expectedCount := tc.providerFilter.Len()
				if len(result) != expectedCount {
					t.Errorf("expected %d filtered specs, got %d", expectedCount, len(result))
				}

				// Verify that all returned specs are in the filter
				for _, spec := range result {
					if !tc.providerFilter.Has(spec.ProviderName) {
						t.Errorf("unexpected provider %s in filtered results", spec.ProviderName)
					}
				}
			}
		})
	}
}

func TestGetImagesFromManifest(t *testing.T) {
	testCases := []struct {
		name     string
		manifest string
		expected []string
	}{
		{
			name: "Deployment",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: quay.io/kubermatic/util:2.10.0
      containers:
        - name: main
          image: docker.io/envoyproxy/gateway:v1.8.5
`,
			expected: []string{"quay.io/kubermatic/util:2.10.0", "docker.io/envoyproxy/gateway:v1.8.5"},
		},
		{
			name: "EnvoyProxy with an Envoy Deployment",
			manifest: `
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: kubermatic-default-proxy
spec:
  provider:
    type: Kubernetes
    kubernetes:
      envoyDeployment:
        replicas: 3
        container:
          image: 'envoyproxy/envoy:distroless-v1.38.4'
`,
			expected: []string{"envoyproxy/envoy:distroless-v1.38.4"},
		},
		{
			name: "EnvoyProxy with an Envoy DaemonSet",
			manifest: `
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: kubermatic-default-proxy
spec:
  provider:
    type: Kubernetes
    kubernetes:
      envoyDaemonSet:
        container:
          image: 'envoyproxy/envoy:distroless-v1.38.4'
`,
			expected: []string{"envoyproxy/envoy:distroless-v1.38.4"},
		},
		{
			name: "EnvoyProxy without an image",
			manifest: `
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: kubermatic-default-proxy
spec:
  provider:
    type: Kubernetes
`,
			expected: nil,
		},
		{
			name: "unknown custom object",
			manifest: `
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: test
spec:
  selfSigned: {}
`,
			expected: nil,
		},
	}

	serializer := json.NewSerializer(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, false)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifests, err := yamlutil.ParseMultipleDocuments(strings.NewReader(tc.manifest))
			if err != nil {
				t.Fatalf("Failed to parse manifest: %v", err)
			}
			if len(manifests) != 1 {
				t.Fatalf("Expected 1 manifest, got %d", len(manifests))
			}

			images, err := getImagesFromManifest(logrus.New(), serializer, manifests[0].Raw)
			if err != nil {
				t.Fatalf("Failed to get images: %v", err)
			}

			if len(images) == 0 && len(tc.expected) == 0 {
				return
			}

			if !diff.DeepEqual(tc.expected, images) {
				t.Fatalf("Images differ:\n%v", diff.ObjectDiff(tc.expected, images))
			}
		})
	}
}
