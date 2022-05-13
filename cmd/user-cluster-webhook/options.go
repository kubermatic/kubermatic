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
	"flag"
	"fmt"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/webhook"

	"k8s.io/klog/v2"
)

type appOptions struct {
	webhook  webhook.Options
	pprof    pprof.Opts
	log      kubermaticlog.Options
	caBundle *certificates.CABundle
}

func initApplicationOptions() (appOptions, error) {
	c := appOptions{
		log: kubermaticlog.NewDefaultOptions(),
	}

	klog.InitFlags(nil)

	c.webhook.AddFlags(flag.CommandLine)
	c.pprof.AddFlags(flag.CommandLine)
	c.log.AddFlags(flag.CommandLine)

	var caBundleFile string
	flag.StringVar(&caBundleFile, "ca-bundle", "", "File containing the PEM-encoded CA bundle for all userclusters")

	flag.Parse()

	caBundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return c, fmt.Errorf("invalid CA bundle file (%q): %w", caBundleFile, err)
	}
	c.caBundle = caBundle

	if err := c.webhook.Validate(); err != nil {
		return c, fmt.Errorf("invalid webhook configuration: %w", err)
	}

	return c, nil
}
