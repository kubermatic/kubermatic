/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

// Package registry groups all container registry related types and helpers in one place.
package registry

import (
	"fmt"
	"strings"

	"github.com/distribution/reference"
)

const (
	// RegistryDocker exists to prevent an import loop between the resources and this package.
	RegistryDocker = "docker.io"
)

func Must(s string, err error) string {
	if err != nil {
		panic(err)
	}

	return s
}

// WithOverwriteFunc is a function that takes a string and either returns that string or a defined override value.
type WithOverwriteFunc func(string) string

// GetOverwriteFunc returns a WithOverwriteFunc based on the given override value.
// Deprecated: This function should not be used anymore. Use the much more
// flexible GetImageRewriterFunc instead.
func GetOverwriteFunc(overwriteRegistry string) WithOverwriteFunc {
	if overwriteRegistry == "" {
		return func(s string) string {
			return s
		}
	}

	return func(_ string) string {
		return overwriteRegistry
	}
}

// ImageRewriter is a function that takes a Docker image reference
// (for example "docker.io/repo/image:tag@sha256:abc123") and
// potentially changes the registry to point to a local registry.
// It's a distinct type from WithOverwriteFunc as it does not just
// work on a registry, but a full image reference.
type ImageRewriter func(string) (string, error)

// GetImageRewriterFunc returns a ImageRewriter that will apply the given
// overwriteRegistry to a given docker image reference.
func GetImageRewriterFunc(overwriteRegistry string) ImageRewriter {
	return func(image string) (string, error) {
		return RewriteImage(image, overwriteRegistry)
	}
}

// RewriteImage will apply the given overwriteRegistry to a given docker
// image reference.
func RewriteImage(image, overwriteRegistry string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q: %w", image, err)
	}

	domain := reference.Domain(named)
	origDomain := domain
	if origDomain == "" {
		origDomain = RegistryDocker
	}

	if overwriteRegistry != "" {
		domain = overwriteRegistry
	}
	if domain == "" {
		domain = RegistryDocker
	}

	// construct name image name
	image = domain + "/" + reference.Path(named)

	if tagged, ok := named.(reference.Tagged); ok {
		image += ":" + tagged.Tag()
	}

	// If the registry (domain) has been changed, remove the
	// digest as it's unlikely that a) the repo digest has
	// been kept when mirroring the image and b) the chance
	// of a local registry being poisoned with bad images is
	// much lower anyhow.
	if origDomain == domain {
		if digested, ok := named.(reference.Digested); ok {
			image += "@" + string(digested.Digest())
		}
	}

	return image, nil
}

func ToOCIURL(s string) string {
	if strings.Contains(s, "://") {
		return s
	}

	return "oci://" + s
}
