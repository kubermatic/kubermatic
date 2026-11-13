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
	"slices"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/images"
)

const (
	helmCommandTimeout = 5 * time.Minute
)

func TestProcessImagesFromHelmChartsAndSystemAppsAndDefaultApps(t *testing.T) {
	log := logrus.New()

	helmBinary, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("Skipping test due to missing helm binary")
	}

	helmClient, err := helm.NewCLI(helmBinary, "", "", helmCommandTimeout, log)
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

	appImages, err := images.GetImagesFromSystemApplicationDefinitions(log, config, helmClient, helmCommandTimeout, "")
	if err != nil {
		t.Errorf("Error calling GetImagesFromSystemApplicationDefinitions: %v", err)
	}
	defaultAppImages, err := images.GetImagesFromDefaultApplicationDefinitions(log, config, helmClient, helmCommandTimeout, "")
	if err != nil {
		t.Errorf("Error calling GetImagesFromDefaultApplicationDefinitions: %v", err)
	}
	containerImages = append(containerImages, appImages...)
	containerImages = append(containerImages, defaultAppImages...)

	if _, _, err := images.CopyImages(context.Background(), log, true, true, containerImages, "test-registry:5000", "kubermatic-installer/test", nil); err != nil {
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
			copiedCount, _, err := images.ArchiveImages(context.Background(), logrus.New(), archive, false, sources, v1.Platform{OS: "linux", Architecture: "amd64"})
			if err != nil {
				t.Fatalf("Error calling ArchiveImages: %v", err)
			}
			if copiedCount != tt.expectedArchivedCount {
				t.Errorf("expected copiedCount and fullCount to be 1, got %d", copiedCount)
			}
		})
	}
}

// pushPlatformIndex pushes an index with one random image per platform plus an
// attestation manifest, like docker buildx creates them.
func pushPlatformIndex(t *testing.T, ref string, platforms ...v1.Platform) {
	t.Helper()

	var index v1.ImageIndex = empty.Index
	for _, platform := range append(platforms, v1.Platform{OS: "unknown", Architecture: "unknown"}) {
		img, err := random.Image(1024, 1)
		if err != nil {
			t.Fatal(err)
		}

		config, err := img.ConfigFile()
		if err != nil {
			t.Fatal(err)
		}

		config.OS = platform.OS
		config.Architecture = platform.Architecture
		config.Variant = platform.Variant
		config.OSVersion = platform.OSVersion

		img, err = mutate.ConfigFile(img, config)
		if err != nil {
			t.Fatal(err)
		}

		index = mutate.AppendManifests(index, mutate.IndexAddendum{
			Add:        img,
			Descriptor: v1.Descriptor{Platform: &platform},
		})
	}

	parsed, err := name.ParseReference(ref)
	if err != nil {
		t.Fatal(err)
	}

	if err := remote.WriteIndex(parsed, index); err != nil {
		t.Fatal(err)
	}
}

func TestCopyImagesPlatforms(t *testing.T) {
	amd64 := v1.Platform{OS: "linux", Architecture: "amd64"}
	arm64 := v1.Platform{OS: "linux", Architecture: "arm64"}

	testCases := []struct {
		name      string
		single    bool // source is an image, not an index
		pinned    bool // source is referenced by digest
		platforms []v1.Platform
		// expected lists the platforms of the copied index, nil means that
		// the source is copied unchanged.
		expected []string
	}{
		{
			name:      "all platforms without a filter",
			platforms: nil,
			expected:  nil,
		},
		{
			name:      "one platform",
			platforms: []v1.Platform{amd64},
			expected:  []string{"linux/amd64"},
		},
		{
			name:      "platform without variant matches all variants",
			platforms: []v1.Platform{arm64},
			expected:  []string{"linux/arm64/v8"},
		},
		{
			name:      "two platforms",
			platforms: []v1.Platform{amd64, arm64},
			expected:  []string{"linux/amd64", "linux/arm64/v8"},
		},
		{
			name:      "image pinned by digest keeps all platforms",
			pinned:    true,
			platforms: []v1.Platform{amd64},
			expected:  nil,
		},
		{
			name:      "no matching platform copies all platforms",
			platforms: []v1.Platform{{OS: "linux", Architecture: "s390x"}},
			expected:  nil,
		},
		{
			name:      "single image is copied unchanged",
			single:    true,
			platforms: []v1.Platform{amd64},
			expected:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := httptest.NewServer(registry.New())
			defer src.Close()
			dst := httptest.NewServer(registry.New())
			defer dst.Close()

			srcURL, err := url.Parse(src.URL)
			if err != nil {
				t.Fatal(err)
			}
			dstURL, err := url.Parse(dst.URL)
			if err != nil {
				t.Fatal(err)
			}

			source := fmt.Sprintf("%s/test/fake:v1", srcURL.Host)
			if tc.single {
				img, err := random.Image(1024, 1)
				if err != nil {
					t.Fatal(err)
				}
				if err := crane.Push(img, source); err != nil {
					t.Fatal(err)
				}
			} else {
				pushPlatformIndex(t, source, amd64, v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}, v1.Platform{OS: "windows", Architecture: "amd64", OSVersion: "10.0.20348.3207"})
			}

			srcDesc, err := crane.Head(source)
			if err != nil {
				t.Fatal(err)
			}

			if tc.pinned {
				source = fmt.Sprintf("%s@%s", source, srcDesc.Digest)
			}

			if _, _, err := images.CopyImages(context.Background(), logrus.New(), false, false, []string{source}, dstURL.Host, "kubermatic-installer/test", tc.platforms); err != nil {
				t.Fatalf("Error calling CopyImages: %v", err)
			}

			dstRef, err := name.ParseReference(fmt.Sprintf("%s/test/fake:v1", dstURL.Host))
			if err != nil {
				t.Fatal(err)
			}

			dstDesc, err := remote.Get(dstRef)
			if err != nil {
				t.Fatalf("Failed to get the copied image: %v", err)
			}

			if tc.expected == nil {
				if dstDesc.Digest != srcDesc.Digest {
					t.Errorf("Expected the copied image to have the source digest %s, got %s", srcDesc.Digest, dstDesc.Digest)
				}
				return
			}

			index, err := dstDesc.ImageIndex()
			if err != nil {
				t.Fatal(err)
			}

			// make sure that the copied index is complete, i.e. all its images and blobs exist
			if err := validate.Index(index); err != nil {
				t.Fatalf("Copied index is invalid: %v", err)
			}

			manifest, err := index.IndexManifest()
			if err != nil {
				t.Fatal(err)
			}

			var platforms []string
			for _, desc := range manifest.Manifests {
				platforms = append(platforms, desc.Platform.String())
			}

			if !slices.Equal(platforms, tc.expected) {
				t.Errorf("Expected platforms %v, got %v", tc.expected, platforms)
			}
		})
	}
}

func TestArchiveImagesPlatform(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	arm64 := v1.Platform{OS: "linux", Architecture: "arm64"}
	source := fmt.Sprintf("%s/test/fake:v1", u.Host)
	pushPlatformIndex(t, source, v1.Platform{OS: "linux", Architecture: "amd64"}, arm64)

	archive := fmt.Sprintf("%s/archive.tar", t.TempDir())
	if _, _, err := images.ArchiveImages(context.Background(), logrus.New(), archive, false, []string{source}, arm64); err != nil {
		t.Fatalf("Error calling ArchiveImages: %v", err)
	}

	tag, err := name.NewTag(source)
	if err != nil {
		t.Fatal(err)
	}

	archived, err := tarball.ImageFromPath(archive, &tag)
	if err != nil {
		t.Fatalf("Failed to load the archived image: %v", err)
	}

	config, err := archived.ConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	if platform := config.Platform(); platform == nil || !platform.Equals(arm64) {
		t.Errorf("Expected the archive to contain the %s image, got %v", arm64, platform)
	}

	// a dry run counts only the images that have the platform
	s390x := v1.Platform{OS: "linux", Architecture: "s390x"}
	count, fullCount, err := images.ArchiveImages(context.Background(), logrus.New(), archive, true, []string{source}, s390x)
	if err != nil {
		t.Fatalf("Error calling ArchiveImages: %v", err)
	}
	if count != 0 || fullCount != 1 {
		t.Errorf("Expected 0 of 1 images in a dry run for %s, got %d of %d", s390x, count, fullCount)
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
