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

package jig

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	"k8c.io/kubermatic/v2/pkg/semver"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kkpNamespace = "kubermatic"
	datacenter   string
	project      string
	version      string
)

func AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&kkpNamespace, "namespace", kkpNamespace, "namespace where KKP is installed into")
	flag.StringVar(&datacenter, "datacenter", datacenter, "KKP datacenter to use (must match whatever is used in the tests to run)")
	flag.StringVar(&project, "project", project, "KKP project to use (if not given, a new project might be created)")
	flag.StringVar(&version, "cluster-version", version, "Kubernetes version of the new user cluster (defaults to $VERSION_TO_TEST or the default version compiled into KKP)")
}

func KubermaticNamespace() string {
	return kkpNamespace
}

func ClusterVersion(log *zap.SugaredLogger) string {
	var v string

	if version != "" {
		v = version
	} else if vv := os.Getenv("VERSION_TO_TEST"); vv != "" {
		log.Info("Defaulting cluster version to VERSION_TO_TEST", "version", vv)
		v = vv
	} else {
		v = defaults.DefaultKubernetesVersioning.Default.String()
		log.Info("Defaulting cluster version to DefaultKubernetesVersioning", "version", v)
	}

	// consistently output a leading "v"
	if v != "" && v[0] != 'v' {
		v = "v" + v
	}

	return v
}

func ClusterSemver(log *zap.SugaredLogger) semver.Semver {
	return *semver.NewSemverOrDie(ClusterVersion(log))
}

func DatacenterName() string {
	return datacenter
}

func Seed(ctx context.Context, client ctrlruntimeclient.Client) (*kubermaticv1.Seed, *kubermaticv1.Datacenter, error) {
	if datacenter == "" {
		return nil, nil, errors.New("no -datacenter given, cannot determine Seed")
	}

	seedsGetter, err := seedsGetterFactory(ctx, client, kkpNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Seeds getter: %w", err)
	}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list Seeds: %w", err)
	}

	for _, seed := range seeds {
		for name, dc := range seed.Spec.Datacenters {
			if name == datacenter {
				return seed, &dc, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no Seed contains a datacenter %q", datacenter)
}
