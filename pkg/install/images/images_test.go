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

	"go.uber.org/zap"

	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

// These tests do not just verify that the Docker images can be extracted
// properly, but also that all addons can be processed for each cluster
// combination (i.e. there are no broken Go templates).
// When working on this test, remember that it also tests pkg/addon/'s code.

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.NewLogrus()

	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("failed to determine versions: %v", err)
	}

	kubermaticVersions := kubermatic.NewFakeVersions()
	clusterVersions := getVersionsFromKubermaticConfiguration(config)
	addonPath := "../../../addons"

	caBundle, err := certificates.NewCABundleFromFile("../../../charts/kubermatic-operator/static/ca-bundle.pem")
	if err != nil {
		t.Fatalf("failed to load CA bundle: %v", err)
	}

	allAddons, err := addonutil.LoadAddonsFromDirectory(addonPath)
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
				clog := plog.WithField("cni", fmt.Sprintf("%s_%s", cniPlugin.Type, cniPlugin.Version))

				images, err := GetImagesForVersion(clog, clusterVersion, cloudSpec, cniPlugin, false, config, allAddons, kubermaticVersions, caBundle, "")
				if err != nil {
					t.Errorf("Error calling getImagesForVersion: %v", err)
				}
				imageSet.Insert(images...)
			}
		}
	}

	if _, _, err := CopyImages(context.Background(), log, true, sets.List(imageSet), "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}
