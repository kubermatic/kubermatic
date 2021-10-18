/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	validClusterNamespace = regexp.MustCompile(`^cluster-[0-9a-z]{10}$`)

	// oldAPIGroup is the group we migrate away from.
	oldAPIGroup = "kubermatic.k8s.io"

	// newAPIGroup is the group we migrate to.
	newAPIGroup = "kubermatic.k8c.io"

	// allKubermaticKinds is a list of all KKP CRDs;
	// not all of these live on all clusters
	allKubermaticKinds = []string{
		"Addon",
		"AddonConfig",
		"AdmissionPlugin",
		"Alertmanager",
		"AllowedRegistry",
		"Cluster",
		"ClusterTemplate",
		"ClusterTemplateInstance",
		"Constraint",
		"ConstraintTemplate",
		"EtcdBackupConfig",
		"EtcdRestore",
		"ExternalCluster",
		"KubermaticSetting",
		"MLAAdminSetting",
		"Preset",
		"Project",
		"RuleGroup",
		"Seed",
		"User",
		"UserProjectBinding",
		"UserSSHKey",
	}
)

// getUserclusterNamespaces is purposefully "dumb" and doesn't list Cluster
// objects to deduce the namespaces or check whether the namespaces have
// the proper ownerRef to a Cluster object, but instead basically just greps
// all namespaces for "cluster-XXXXXXXXXX". This is to ensure even half broken
// namespaces are not accidentally ignored during the preflight checks.
func getUserclusterNamespaces(ctx context.Context, client ctrlruntimeclient.Client) ([]string, error) {
	nsList := corev1.NamespaceList{}

	if err := client.List(ctx, &nsList); err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := []string{}
	for _, namespace := range nsList.Items {
		if validClusterNamespace.MatchString(namespace.Name) {
			namespaces = append(namespaces, namespace.Name)
		}
	}

	return namespaces, nil
}
