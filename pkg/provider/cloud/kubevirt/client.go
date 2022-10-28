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

package kubevirt

import (
	"encoding/base64"

	kubevirtv1 "kubevirt.io/api/core/v1"
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Client represents a struct that includes controller runtime client and rest configuration
// that is needed for service accounts kubeconfig generation.
type Client struct {
	ctrlruntimeclient.Client
	// RestConfig represents a rest client configuration
	RestConfig *restclient.Config
}

// ClientOptions allows to pass specific options that influence the client behaviour.
type ClientOptions struct {
	// ControllerRuntimeOptions represents the options coming from controller runtime.
	ControllerRuntimeOptions ctrlruntimeclient.Options
	// FakeObjects allows to inject custom objects for testing.
	FakeObjects    []ctrlruntimeclient.Object
	loadFakeClient bool
}

func newClient(kubeconfig string, opts ClientOptions) (*Client, error) {
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

	if opts.loadFakeClient {
		client = ctrlruntimeclientfake.NewClientBuilder().WithObjects(opts.FakeObjects...).Build()
	} else {
		client, err = ctrlruntimeclient.New(restConfig, opts.ControllerRuntimeOptions)
		if err != nil {
			return nil, err
		}
	}

	if err = kubevirtv1.AddToScheme(client.Scheme()); err != nil {
		return nil, err
	}

	if err = kvinstancetypev1alpha1.AddToScheme(client.Scheme()); err != nil {
		return nil, err
	}

	if err = cdiv1beta1.AddToScheme(client.Scheme()); err != nil {
		return nil, err
	}

	return &Client{Client: client, RestConfig: restConfig}, nil
}

// NewClient returns controller runtime client that points to KubeVirt infra cluster.
func NewClient(kubeconfig string, opts ClientOptions) (*Client, error) {
	return newClient(kubeconfig, opts)
}

// NewFakeClient returns controller runtime fake client.
func NewFakeClient(kubeconfig string, opts ClientOptions) (*Client, error) {
	opts.loadFakeClient = true
	return newClient(kubeconfig, opts)
}
