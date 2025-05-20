//go:build ee

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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	velerocontroller "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/user-cluster/velero-controller"
	policybindingcontroller "k8c.io/kubermatic/v2/pkg/ee/policy-binding-controller"
	resourceusagecontroller "k8c.io/kubermatic/v2/pkg/ee/resource-usage-controller"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func addFlags(fs *flag.FlagSet, opt *controllerRunOptions) {
	fs.StringVar(&opt.clusterBackup.backupStorageLocation, "cluster-backup-storage-location", "", "Name of the ClusterBackupStorageLocation in the kubermatic namespace that is used by this cluster. Used to scope down RBAC requirements.")
	fs.StringVar(&opt.clusterBackup.credentialSecret, "cluster-backup-credential-secret", "", "Name of the credential Secret for the chosen CBSL in the kubermatic namespace. Used to scope down RBAC requirements.")
}

func setupSeedManager(restConfig *rest.Config, opt controllerRunOptions) (manager.Manager, error) {
	cbslCacheOptions := cache.ByObject{
		Namespaces: map[string]cache.Config{
			resources.KubermaticNamespace: {},
		},
	}

	secretCacheOptions := cache.ByObject{
		Namespaces: map[string]cache.Config{
			opt.namespace: {},
		},
	}

	if loc := opt.clusterBackup.backupStorageLocation; loc != "" {
		cbslCacheOptions.Namespaces[resources.KubermaticNamespace] = cache.Config{
			FieldSelector: fields.SelectorFromSet(fields.Set{"metadata.name": loc}),
		}
	}

	if cred := opt.clusterBackup.credentialSecret; cred != "" {
		secretCacheOptions.Namespaces[resources.KubermaticNamespace] = cache.Config{
			FieldSelector: fields.SelectorFromSet(fields.Set{"metadata.name": cred}),
		}
	}

	return manager.New(restConfig, manager.Options{
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				opt.namespace: {},
			},
			// The cluster backup feature (EE) needs to read CBSL and their credentials from the kubermatic namespace,
			// but only has permission to read exactly these 2 objects from the kubermatic namespace. For its own
			// cluster namespace however, we are allowed to access all Secrets.
			ByObject: map[ctrlruntimeclient.Object]cache.ByObject{
				&kubermaticv1.ClusterBackupStorageLocation{}: cbslCacheOptions,
				&corev1.Secret{}: secretCacheOptions,
			},
		},
	})
}

func setupControllers(
	log *zap.SugaredLogger,
	seedMgr, userMgr manager.Manager,
	clusterName string,
	versions kubermatic.Versions,
	overwriteRegistry string,
	caBundle *certificates.CABundle,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker,
	namespace string,
) error {
	if err := resourceusagecontroller.Add(log, seedMgr, userMgr, clusterName, caBundle, clusterIsPaused); err != nil {
		return fmt.Errorf("failed to create cluster-backup controller: %w", err)
	}

	if err := velerocontroller.Add(seedMgr, userMgr, log, clusterName, versions, overwriteRegistry); err != nil {
		return fmt.Errorf("failed to create cluster-backup controller: %w", err)
	}

	if err := policybindingcontroller.Add(seedMgr, userMgr, log, namespace, clusterName, clusterIsPaused); err != nil {
		return fmt.Errorf("failed to create policy-binding controller: %w", err)
	}

	return nil
}
