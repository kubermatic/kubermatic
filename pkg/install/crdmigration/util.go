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

type Kind struct {
	// Name is the name of the Kind, e.g. "Cluster"
	Name string
	// Namespaced is true if the Kind is namespaced.
	Namespaced bool
	// MasterCluster is true if resources of this kind exist on master clusters.
	MasterCluster bool
	// SeedCluster is true if resources of this kind exist on seed clusters;
	// this includes resources that are just mirrored into seeds, like Users, of which the
	// primary resource lives on the master.
	SeedCluster bool
}

var (
	validClusterNamespace = regexp.MustCompile(`^cluster-[0-9a-z]{10}$`)

	// oldAPIGroup is the group we migrate away from.
	oldAPIGroup = "kubermatic.k8s.io"

	// newAPIGroup is the group we migrate to.
	newAPIGroup = "kubermatic.k8c.io"

	// allKubermaticKinds is a list of all KKP CRDs, sorted by ownership,
	// i.e. the first item (User) owns stuff, whereas items further down
	// do not own anything. For creating new resources, follow the given
	// order, when cleaning up, go in reverse.
	// Current ownerships are as follows:
	//
	// User      owns   Project
	//
	// Project   owns   UserProjectBinding
	// Project   owns   UserSSHKey
	// Project   owns   ExternalCluster
	//
	// Cluster   owns   Addon [why? it's in the cluster namespace anyway]
	// Cluster   owns   EtcdBackupConfig
	allKubermaticKinds = []Kind{
		{Name: "User", Namespaced: false, MasterCluster: true, SeedCluster: true},

		{Name: "Project", Namespaced: false, MasterCluster: true, SeedCluster: true},
		{Name: "Cluster", Namespaced: false, MasterCluster: false, SeedCluster: true},

		{Name: "Addon", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "AddonConfig", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "AdmissionPlugin", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "Alertmanager", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "AllowedRegistry", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "ClusterTemplate", Namespaced: false, MasterCluster: true, SeedCluster: true},
		{Name: "ClusterTemplateInstance", Namespaced: false, MasterCluster: false, SeedCluster: true},
		{Name: "Constraint", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "ConstraintTemplate", Namespaced: false, MasterCluster: false, SeedCluster: true},
		{Name: "EtcdBackupConfig", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "EtcdRestore", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "ExternalCluster", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "KubermaticSetting", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "MLAAdminSetting", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "Preset", Namespaced: false, MasterCluster: true, SeedCluster: false},
		{Name: "RuleGroup", Namespaced: true, MasterCluster: false, SeedCluster: true},
		{Name: "Seed", Namespaced: true, MasterCluster: true, SeedCluster: true},
		{Name: "UserProjectBinding", Namespaced: false, MasterCluster: true, SeedCluster: true},
		{Name: "UserSSHKey", Namespaced: false, MasterCluster: true, SeedCluster: false},
	}
)

func getKind(name string) Kind {
	for i, kind := range allKubermaticKinds {
		if kind.Name == name {
			return allKubermaticKinds[i]
		}
	}

	panic(fmt.Sprintf("Kind %s is not a KKP CRD and not applicable for the migration.", name))
}

func reverseKinds(kinds []Kind) []Kind {
	result := make([]Kind, len(kinds))
	end := len(kinds) - 1

	for i := end; i >= 0; i-- {
		result[end-i] = kinds[i]
	}

	return result
}

func isNamespacedKind(name string) bool {
	return getKind(name).Namespaced
}

func filterKinds(predicate func(Kind) bool) []Kind {
	result := []Kind{}

	for i, kind := range allKubermaticKinds {
		if predicate(kind) {
			result = append(result, allKubermaticKinds[i])
		}
	}

	return result
}

func getMasterClusterKinds() []Kind {
	return filterKinds(func(k Kind) bool { return k.MasterCluster })
}

func getSeedClusterKinds() []Kind {
	return filterKinds(func(k Kind) bool { return k.SeedCluster })
}

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
