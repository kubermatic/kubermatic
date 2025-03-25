//go:build integration

/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/images"
)

func TestProcessImagesFromHelmChartsAndSystemApps(t *testing.T) {
	log := logrus.New()

	helmBinary, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("Skipping test due to missing helm binary")
	}

	helmClient, err := helm.NewCLI(helmBinary, "", "", 5*time.Minute, log)
	if err != nil {
		t.Fatalf("failed to create Helm client: %v", err)
	}

	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("failed to determine versions: %v", err)
	}

	var containerImages []string
	chartImages, err := images.GetImagesForHelmCharts(context.Background(), log, config, helmClient, "../../../../charts/monitoring", "", "", "")
	if err != nil {
		t.Errorf("error calling GetImagesForHelmCharts: %v", err)
	}
	containerImages = append(containerImages, chartImages...)

	appImages, err := images.GetImagesFromSystemApplicationDefinitions(log, config, helmClient, 5*time.Minute, "")
	if err != nil {
		t.Errorf("Error calling GetImagesFromSystemApplicationDefinitions: %v", err)
	}
	containerImages = append(containerImages, appImages...)

	if _, _, err := images.CopyImages(context.Background(), log, true, containerImages, "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling CopyImages: %v", err)
	}
}

func TestArchiveImages(t *testing.T) {
	// Set up a fake registry
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name                  string
		images                []v1.Image
		expectedErr           bool
		expectedArchivedCount int
	}{
		{
			name:                  "empty image list",
			images:                []v1.Image{},
			expectedErr:           false,
			expectedArchivedCount: 0,
		},
		{
			name: "unsupported image",
			images: []v1.Image{
				func() v1.Image {
					img, err := random.Image(1024, 5)
					if err != nil {
						t.Fatal(err)
					}
					return mutate.MediaType(img, types.DockerManifestSchema1Signed)
				}(),
			},
			expectedErr:           false,
			expectedArchivedCount: 0,
		},
		{
			name: "valid image",
			images: []v1.Image{
				func() v1.Image {
					img, err := random.Image(1024, 5)
					if err != nil {
						t.Fatal(err)
					}
					return img
				}(),
			},
			expectedErr:           false,
			expectedArchivedCount: 1,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sources := []string{}
			for _, image := range tt.images {
				src := fmt.Sprintf("%s/test/fake", u.Host)
				sources = append(sources, src)
				if err := crane.Push(image, src); err != nil {
					t.Fatal(err)
				}
			}

			archive := fmt.Sprintf("%s/archive.tar.gz", t.TempDir())
			copiedCount, _, err := images.ArchiveImages(context.Background(), logrus.New(), archive, false, sources)
			if err != nil {
				t.Fatalf("Error calling ArchiveImages: %v", err)
			}
			if copiedCount != tt.expectedArchivedCount {
				t.Errorf("expected copiedCount and fullCount to be 1, got %d", copiedCount)
			}
		})
	}
}

func TestLoadImages(t *testing.T) {
	// fake registry to push images to
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	// push a archive to the fake registry
	if err := images.LoadImages(context.Background(), logrus.New(), "./testdata/valid-archive.tar.gz", false, u.Host, "kubermatic-installer/test"); err != nil {
		t.Fatalf("Error calling LoadImages: %v", err)
	}

	// validate that we can load the image from the fake registry
	imageSource := fmt.Sprintf("%s/test/fake:v1", u.Host)
	ref, err := name.ParseReference(imageSource)
	if err != nil {
		t.Fatalf("failed to parse reference %s: %v", imageSource, err)
	}

	_, err = remote.Image(ref)
	if err != nil {
		t.Errorf("Error calling remote.Image: %v", err)
	}
}
