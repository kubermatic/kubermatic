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
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
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

	if _, _, err := CopyImages(context.Background(), log, true, sets.List(imageSet), "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
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
