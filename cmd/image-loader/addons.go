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

package main

import (
	"fmt"
	"io/ioutil"
	"path"

	"go.uber.org/zap"

	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func getImagesFromAddons(log *zap.SugaredLogger, addonsPath string, cluster *kubermaticv1.Cluster) ([]string, error) {
	credentials := resources.Credentials{}

	addonData, err := addonutil.NewTemplateData(cluster, credentials, "", "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create addon template data: %v", err)
	}

	infos, err := ioutil.ReadDir(addonsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to list addons: %v", err)
	}

	serializer := json.NewSerializerWithOptions(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, json.SerializerOptions{})
	var images []string
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		addonName := info.Name()
		addonImages, err := getImagesFromAddon(log, path.Join(addonsPath, addonName), serializer, addonData)
		if err != nil {
			return nil, fmt.Errorf("failed to get images for addon %s: %v", addonName, err)
		}
		images = append(images, addonImages...)
	}

	return images, nil
}

func getImagesFromAddon(log *zap.SugaredLogger, addonPath string, decoder runtime.Decoder, data *addonutil.TemplateData) ([]string, error) {
	log = log.With(zap.String("addon", path.Base(addonPath)))
	log.Debug("Processing manifests...")

	allManifests, err := addonutil.ParseFromFolder(log, "", addonPath, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addon templates in %s: %v", addonPath, err)
	}

	var images []string
	for _, manifest := range allManifests {
		manifestImages, err := getImagesFromManifest(log, decoder, manifest.Raw)
		if err != nil {
			return nil, err
		}
		images = append(images, manifestImages...)
	}
	return images, nil
}
