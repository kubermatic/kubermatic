package crdmigration

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	KubermaticNamespace     string
	KubermaticConfiguration *operatorv1alpha1.KubermaticConfiguration
	MasterClient            ctrlruntimeclient.Client
	Seeds                   map[string]*kubermaticv1.Seed
	SeedClients             map[string]ctrlruntimeclient.Client
}
