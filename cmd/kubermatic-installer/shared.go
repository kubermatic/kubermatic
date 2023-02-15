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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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

func loadHelmValues(filename string) (*yamled.Document, error) {
	if filename == "" {
		doc, _ := yamled.Load(bytes.NewReader([]byte("---\n")))
		return doc, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values, err := yamled.Load(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", filename, err)
	}

	return values, nil
}
