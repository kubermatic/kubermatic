package seedsync

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func namespaceCreator(name string) reconciling.NamedNamespaceCreatorGetter {
	return func() (string, reconciling.NamespaceCreator) {
		return name, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			return ns, nil
		}
	}
}

func seedCreator(seed *kubermaticv1.Seed) reconciling.NamedSeedCreatorGetter {
	return func() (string, reconciling.SeedCreator) {
		return seed.Name, func(_ *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
			return seed, nil
		}
	}
}
