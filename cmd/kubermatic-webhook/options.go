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
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	"k8c.io/kubermatic/v2/pkg/webhook"

	"k8s.io/klog/v2"
)

type appOptions struct {
	seedName     string
	namespace    string
	featureGates features.FeatureGate
	caBundle     *certificates.CABundle

	webhook webhook.Options
	pprof   flagopts.PProf
	log     kubermaticlog.Options

	// for development purposes, a local configuration file
	// can be used to provide the KubermaticConfiguration
	kubermaticConfiguration *kubermaticv1.KubermaticConfiguration
}

func initApplicationOptions() (appOptions, error) {
	c := appOptions{
		featureGates: features.FeatureGate{},
		log:          kubermaticlog.NewDefaultOptions(),
	}

	var (
		caBundleFile string
		configFile   string
		err          error
	)

	klog.InitFlags(nil)

	flag.StringVar(&c.seedName, "seed-name", "", "The name of the seed this controller is running in. It will be used to build the absolute url for a user cluster.")
	flag.Var(&c.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for Seed resources")
	flag.StringVar(&caBundleFile, "ca-bundle", "", "File containing the PEM-encoded CA bundle for all user clusters")
	flag.StringVar(&configFile, "kubermatic-configuration-file", "", "(for development only) path to a KubermaticConfiguration YAML file")

	c.webhook.AddFlags(flag.CommandLine, "webhook")
	c.pprof.AddFlags(flag.CommandLine)
	c.log.AddFlags(flag.CommandLine)

	flag.Parse()

	if err = c.webhook.Validate(); err != nil {
		return c, fmt.Errorf("invalid webhook configuration: %w", err)
	}

	if configFile != "" {
		if c.kubermaticConfiguration, err = loadKubermaticConfiguration(configFile); err != nil {
			return c, fmt.Errorf("invalid KubermaticConfiguration: %w", err)
		}
	}

	caBundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return c, fmt.Errorf("invalid CA bundle file (%q): %w", caBundleFile, err)
	}
	c.caBundle = caBundle

	return c, nil
}

func loadKubermaticConfiguration(filename string) (*kubermaticv1.KubermaticConfiguration, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer f.Close()

	config := &kubermaticv1.KubermaticConfiguration{}
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse file as YAML: %w", err)
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to process: %w", err)
	}

	return defaulted, nil
}
