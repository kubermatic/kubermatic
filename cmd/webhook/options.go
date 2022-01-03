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
	"context"
	"flag"
	"fmt"
	"io/ioutil"

	"go.uber.org/zap"
	"k8s.io/klog"

	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/webhook"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

type appOptions struct {
	seedName     string
	namespace    string
	featureGates features.FeatureGate
	caBundle     *certificates.CABundle

	webhook webhook.Options
	pprof   pprof.Opts
	log     kubermaticlog.Options

	// for development purposes, a local configuration file
	// can be used to provide the KubermaticConfiguration
	kubermaticConfiguration *operatorv1alpha1.KubermaticConfiguration
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

	flag.StringVar(&c.seedName, "seed-name", "", "The name of the seed this controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.Var(&c.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&c.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for Seed resources")
	flag.StringVar(&caBundleFile, "ca-bundle", "", "File containing the PEM-encoded CA bundle for all userclusters")
	flag.StringVar(&configFile, "kubermatic-configuration-file", "", "(for development only) path to a KubermaticConfiguration YAML file")

	c.webhook.AddFlags(flag.CommandLine)
	c.pprof.AddFlags(flag.CommandLine)
	c.log.AddFlags(flag.CommandLine)

	flag.Parse()

	if err = c.webhook.Validate(); err != nil {
		return c, fmt.Errorf("invalid webhook configuration: %v", err)
	}

	if configFile != "" {
		if c.kubermaticConfiguration, err = loadKubermaticConfiguration(configFile); err != nil {
			return c, fmt.Errorf("invalid KubermaticConfiguration: %w", err)
		}
	}

	caBundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return c, fmt.Errorf("invalid CA bundle file (%q): %v", caBundleFile, err)
	}
	c.caBundle = caBundle

	return c, nil
}

// controllerContext holds all controllerRunOptions plus everything that
// needs to be initialized first
type controllerContext struct {
	ctx                  context.Context
	runOptions           appOptions
	mgr                  manager.Manager
	clientProvider       *client.Provider
	seedGetter           provider.SeedGetter
	configGetter         provider.KubermaticConfigurationGetter
	dockerPullConfigJSON []byte
	log                  *zap.SugaredLogger
	versions             kubermatic.Versions
}

func loadKubermaticConfiguration(filename string) (*operatorv1alpha1.KubermaticConfiguration, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse file as YAML: %v", err)
	}

	defaulted, err := defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to process: %v", err)
	}

	return defaulted, nil
}
