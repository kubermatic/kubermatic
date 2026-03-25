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
	"bytes"
	"context"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	goyaml "gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type cobraFuncE func(cmd *cobra.Command, args []string) error

func handleErrors(logger *logrus.Logger, action cobraFuncE) cobraFuncE {
	return func(cmd *cobra.Command, args []string) error {
		err := action(cmd, args)
		if err != nil {
			logger.Errorf("❌ Operation failed: %v.", err)
		}

		return err
	}
}

func findKubermaticConfiguration(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (*kubermaticv1.KubermaticConfiguration, error) {
	getter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(client, namespace)
	if err != nil {
		return nil, err
	}

	return getter(ctx)
}

func loadKubermaticConfiguration(filename string) (*kubermaticv1.KubermaticConfiguration, *unstructured.Unstructured, error) {
	// the config file is optional during upgrades, so we do not yet error out if it's not given
	if filename == "" {
		return nil, nil, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	raw := &unstructured.Unstructured{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(raw); err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s: %w", filename, err)
	}

	config := &kubermaticv1.KubermaticConfiguration{}
	if err = yaml.UnmarshalStrict(content, config); err != nil {
		return nil, nil, fmt.Errorf("%s is not a valid KubermaticConfiguration: %w", filename, err)
	}

	return config, raw, nil
}

func loadHelmValues(filenames []string) (*yamled.Document, error) {
	if len(filenames) == 0 {
		doc, _ := yamled.Load(bytes.NewReader([]byte("---\n")))
		return doc, nil
	}

	// We create an empty map into which we merge everything.
	mergedValues := make(map[string]any)

	for _, filename := range filenames {
		if filename == "" {
			continue
		}

		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		// decoding the current file
		currentMap := make(map[string]any)

		// using unmarshal directly on the bytes
		if err := goyaml.Unmarshal(content, &currentMap); err != nil {
			return nil, fmt.Errorf("failed to decode %s: %w", filename, err)
		}

		// mergo.WithOverride ensures that values from "currentMap"
		// overwrite existing values in "mergedValues"
		if err := mergo.Merge(&mergedValues, currentMap, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("failed to merge values from %s: %w", filename, err)
		}
	}

	// Convert the finished map back to bytes for yamled
	var buf bytes.Buffer
	encoder := goyaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(mergedValues); err != nil {
		return nil, fmt.Errorf("failed to encode merged values: %w", err)
	}

	// load yamled (as expected by the rest of the code)
	values, err := yamled.Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to load final values: %w", err)
	}

	return values, nil
}
