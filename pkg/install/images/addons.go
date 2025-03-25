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
	"fmt"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

var serializer = json.NewSerializerWithOptions(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, json.SerializerOptions{})

func getImagesFromAddons(log logrus.FieldLogger, addons map[string]*addon.Addon, cluster *kubermaticv1.Cluster) ([]string, error) {
	credentials := resources.Credentials{}

	addonData, err := addon.NewTemplateData(cluster, credentials, "", "", "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create addon template data: %w", err)
	}

	var images []string
	for addonName, addonObj := range addons {
		addonImages, err := getImagesFromAddon(log.WithField("addon", addonName), addonObj, serializer, addonData)
		if err != nil {
			return nil, fmt.Errorf("failed to get images for addon %s: %w", addonName, err)
		}
		images = append(images, addonImages...)
	}

	return images, nil
}

func getImagesFromAddon(log logrus.FieldLogger, addonObj *addon.Addon, decoder runtime.Decoder, data *addon.TemplateData) ([]string, error) {
	log.Debug("Processing addon…")

	manifests, err := addonObj.Render("", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render addon: %w", err)
	}

	var images []string
	for _, manifest := range manifests {
		manifestImages, err := getImagesFromManifest(log, decoder, manifest.Raw)
		if err != nil {
			return nil, err
		}
		images = append(images, manifestImages...)
	}

	return images, nil
}
