/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubevirtnetworkcontroller

import (
	"context"
	"encoding/base64"
	"errors"

	kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(kubeovnv1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
}

func newClient(kubeconfig string) (ctrlruntimeclient.Client, error) {
	var client ctrlruntimeclient.Client
	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(kubeconfig)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return nil, err
	}

	client, err = ctrlruntimeclient.New(restConfig, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *Reconciler) getKubeVirtInfraKConfig(ctx context.Context, cluster *kubermaticv1.Cluster) (string, error) {
	if cluster.Spec.Cloud.Kubevirt.Kubeconfig != "" {
		return cluster.Spec.Cloud.Kubevirt.Kubeconfig, nil
	}

	if cluster.Spec.Cloud.Kubevirt.CredentialsReference == nil {
		return "", errors.New("no credentials provided")
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, r.Client)
	kubeconfig, err := secretKeySelectorFunc(cluster.Spec.Cloud.Kubevirt.CredentialsReference, resources.KubeVirtKubeconfig)
	if err != nil {
		return "", err
	}

	return kubeconfig, nil
}

func (r *Reconciler) SetupKubeVirtInfraClient(ctx context.Context, cluster *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	kubeconfig, err := r.getKubeVirtInfraKConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}

	kubeVirtInfraClient, err := newClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubeVirtInfraClient, nil
}
