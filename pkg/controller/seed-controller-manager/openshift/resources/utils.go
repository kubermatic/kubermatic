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

package resources

import (
	"fmt"

	"github.com/docker/distribution/reference"
)

// OpenshiftImageWithRegistry will return docker image name for Openshift images. The function is
// digest-aware and can be used with the overwriteRegistry option and with image-loader.
func OpenshiftImageWithRegistry(image, componentName, version, registry string) (string, error) {
	if registry == "" {
		return image, nil
	}
	imageRef, err := reference.ParseNamed(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image: %v", err)
	}
	if reference.Domain(imageRef) == registry {
		return image, nil
	}
	if taggedImageRef, ok := imageRef.(reference.NamedTagged); ok {
		return fmt.Sprintf("%s/%s:%s", registry, reference.Path(imageRef), taggedImageRef.Tag()), nil
	} else if _, ok := imageRef.(reference.Digested); ok {
		// if the image is passed with digest, we use the component name and version to
		// tag the image
		if componentName == "" || version == "" {
			return "", fmt.Errorf("failed to set Openshift image tag. Component name and Openshift version must be set")
		}
		return fmt.Sprintf("%s/%s:%s-%s", registry, reference.Path(imageRef), version, componentName), nil
	}
	return "", nil
}
