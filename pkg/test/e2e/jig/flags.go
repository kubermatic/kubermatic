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
	"math/big"
	"os"
	"strconv"
	"sync"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kkpNamespace = "kubermatic"
	buildID      string
	version      string
	sshPubKey    string

	// once is for goroutine-safe defaulting of the build ID.
	once = &sync.Once{}
)

func AddFlags(fs *flag.FlagSet) {
	flag.StringVar(&buildID, "build-id", buildID, "unique identifier for this test (defaults to $BUILD_ID)")
	flag.StringVar(&kkpNamespace, "namespace", kkpNamespace, "namespace where KKP is installed into")
	flag.StringVar(&version, "cluster-version", version, "Kubernetes version of the new user cluster (defaults to $VERSION_TO_TEST or the default version compiled into KKP)")
	flag.StringVar(&sshPubKey, "ssh-pub-key", sshPubKey, "Optional SSH public key to assign to the Machine objects (requires user-ssh-key-agent to be disabled)")
}

func SSHPublicKey() string {
	return sshPubKey
}

func BuildID() string {
	once.Do(func() {
		if buildID == "" {
			buildID = os.Getenv("BUILD_ID")

			// Space is precious in cluster names and the build ID
			// is a super long 64 bit value (e.g. 1552957086332096512, or 19 digits).
			// To give us more room without losing context, we convert from
			// base 10 to base35 ([0-9a-z], i.e. lowercased alphanumeric)
			// (resulting in "g2xnbutsi0sw", 12 characters).
			if buildID != "" {
				id, err := strconv.ParseInt(buildID, 10, 64)
				if err == nil {
					buildID = big.NewInt(id).Text(35)
				}
			}
		}

		if buildID == "" {
			buildID = rand.String(8)
		}
	})

	return buildID
}

func KubermaticNamespace() string {
	return kkpNamespace
}

func ClusterVersion(log *zap.SugaredLogger) string {
	var v string

	if version != "" {
		v = version
	} else if vv := os.Getenv("VERSION_TO_TEST"); vv != "" {
		log.Infow("Defaulting cluster version to VERSION_TO_TEST", "version", vv)
		v = vv
	} else {
		v = defaulting.DefaultKubernetesVersioning.Default.String()
		log.Infow("Defaulting cluster version to DefaultKubernetesVersioning", "version", v)
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

func Datacenter(ctx context.Context, client ctrlruntimeclient.Client, datacenter string) (*kubermaticv1.Datacenter, error) {
	if datacenter == "" {
		return nil, errors.New("no datacenter given, cannot determine Seed")
	}

	datacenterGetter, err := kubernetes.DatacenterGetterFactory(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create datacenters getter: %w", err)
	}

	return datacenterGetter(ctx, datacenter)
}
