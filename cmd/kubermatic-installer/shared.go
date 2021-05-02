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
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func handleErrors(logger *logrus.Logger, action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		err := action(ctx)
		if err != nil {
			logger.Errorf("❌ Operation failed: %v.", err)
			err = cli.NewExitError("", 1)
		}

		return err
	}
}

func setupLogger(logger *logrus.Logger, action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if ctx.GlobalBool("verbose") {
			logger.SetLevel(logrus.DebugLevel)
		}

		return action(ctx)
	}
}

func loadKubermaticConfiguration(filename string) (*operatorv1alpha1.KubermaticConfiguration, *unstructured.Unstructured, error) {
	if filename == "" {
		return nil, nil, errors.New("no file specified via --config flag")
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	raw := &unstructured.Unstructured{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(raw); err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(config); err != nil {
		return nil, raw, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	return config, raw, nil
}

func loadHelmValues(filename string) (*yamled.Document, error) {
	if filename == "" {
		return nil, errors.New("no file specified via --helm-values flag")
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values, err := yamled.Load(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	return values, nil
}
