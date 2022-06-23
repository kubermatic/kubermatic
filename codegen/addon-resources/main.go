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

package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/addon"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type containerData struct {
	ContainerName string                      `json:"containerName"`
	Resources     corev1.ResourceRequirements `json:"resources"`
}

type objectData struct {
	Kind         string          `json:"kind"`
	ResourceName string          `json:"resourceName"`
	Filename     string          `json:"filename"`
	Containers   []containerData `json:"containers"`
}

type addonData struct {
	AddonName string       `json:"addonName"`
	Resources []objectData `json:"resources"`
}

type outputData struct {
	Addons []addonData `json:"addons"`
}

func main() {
	var kubermaticDir = flag.String("kubermaticdir", ".", "directory containing the kubermatic sources")
	var outputFile = flag.String("output", "addonresources.json", "path and filename for the generated file")

	flag.Parse()

	log := kubermaticlog.NewDefault().Sugar()

	addonsDir := filepath.Join(*kubermaticDir, "addons")

	log.Info("Rendering addons and collecting manifests…")

	templateData, err := createTemplateData()
	if err != nil {
		log.Fatalw("Failed to create addon templating data", zap.Error(err))
	}

	manifests, err := addon.ParseFromFolder(log, "", addonsDir, templateData)
	if err != nil {
		log.Fatalw("Failed to parse addons", zap.Error(err))
	}

	// group manifests by addon
	addonManifests := map[string][]addon.Manifest{}
	for _, m := range manifests {
		parts := strings.Split(m.SourceFile, string(filepath.Separator))
		if len(parts) < 3 {
			continue
		}

		addonName := parts[1]
		relPath := filepath.Join(parts[2:]...)

		if _, exists := addonManifests[addonName]; !exists {
			addonManifests[addonName] = []addon.Manifest{}
		}

		addonManifests[addonName] = append(addonManifests[addonName], addon.Manifest{
			SourceFile: relPath,
			Content:    m.Content,
		})
	}

	// prepare final output data
	result := outputData{}

	for addonName, manifests := range addonManifests {
		addonInfo := addonData{
			AddonName: addonName,
			Resources: []objectData{},
		}

		for _, manifest := range manifests {
			objectInfo, err := parseManifest(manifest)
			if err != nil {
				log.Fatalw("Failed to determine resources", zap.Error(err))
			}
			if objectInfo == nil {
				continue
			}

			addonInfo.Resources = append(addonInfo.Resources, *objectInfo)
		}

		result.Addons = append(result.Addons, addonInfo)
	}

	log.Infow("Writing addon data…", "filename", *outputFile)

	sort.Slice(result.Addons, func(i, j int) bool {
		return result.Addons[i].AddonName < result.Addons[j].AddonName
	})

	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalw("Failed to create output file", zap.Error(err))
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(result); err != nil {
		log.Fatalw("Failed to encode output as JSON", zap.Error(err))
	}

	log.Info("Done.")
}

func createTemplateData() (*addon.TemplateData, error) {
	dnsClusterIP := "1.2.3.4"
	variables := map[string]interface{}{
		"NodeAccessNetwork": "172.26.0.0/16",
	}

	cluster := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.25.0.0/16",
					},
				},
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{},
		},
		Status: kubermaticv1.ClusterStatus{
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane: *semver.NewSemverOrDie("2.20.0"),
			},
		},
	}

	return addon.NewTemplateData(cluster, resources.Credentials{}, "", dnsClusterIP, "", nil, variables)
}

func parseManifest(manifest addon.Manifest) (*objectData, error) {
	var u unstructured.Unstructured
	if err := yaml.UnmarshalStrict(manifest.Content.Raw, &u); err != nil {
		return nil, err
	}

	var (
		podSpec *corev1.PodSpec
	)

	switch u.GetKind() {
	case "Deployment":
		var ad appsv1.Deployment
		if err := yaml.UnmarshalStrict(manifest.Content.Raw, &ad); err != nil {
			return nil, err
		}
		podSpec = &ad.Spec.Template.Spec

	case "DaemonSet":
		var ad appsv1.DaemonSet
		if err := yaml.UnmarshalStrict(manifest.Content.Raw, &ad); err != nil {
			return nil, err
		}
		podSpec = &ad.Spec.Template.Spec

	case "StatefulSet":
		var as appsv1.StatefulSet
		if err := yaml.UnmarshalStrict(manifest.Content.Raw, &as); err != nil {
			return nil, err
		}
		podSpec = &as.Spec.Template.Spec
	}

	// this manifest contains a resource without a PodSpec, like a ConfigMap or an Ingress
	if podSpec == nil {
		return nil, nil
	}

	result := &objectData{
		ResourceName: u.GetName(),
		Filename:     manifest.SourceFile,
		Kind:         u.GetKind(),
		Containers:   []containerData{},
	}

	for _, container := range podSpec.Containers {
		if container.Resources.Size() == 0 {
			continue
		}

		result.Containers = append(result.Containers, containerData{
			ContainerName: container.Name,
			Resources:     container.Resources,
		})
	}

	return result, nil
}
