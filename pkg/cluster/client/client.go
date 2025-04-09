/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package client

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewInternal returns a new instance of the client connection provider that
// only works from within the seed cluster but has the advantage that it doesn't leave
// the seed cluster's network.
func NewInternal(seedClient ctrlruntimeclient.Client) (*Provider, error) {
	return &Provider{
		seedClient:         seedClient,
		useExternalAddress: false,
		restMapperCache:    restmapper.New(),
	}, nil
}

// NewExternal returns a new instance of the client connection provider
// that uses the external cluster address and hence works from everywhere.
// Use NewInternal if possible.
func NewExternal(seedClient ctrlruntimeclient.Client) (*Provider, error) {
	return NewExternalWithProxy(seedClient, "")
}

// NewExternalWithProxy provides the same client connection provider
// as NewExternal but with support to use a Proxy.
func NewExternalWithProxy(seedClient ctrlruntimeclient.Client, proxy string) (*Provider, error) {
	return &Provider{
		seedClient:         seedClient,
		useExternalAddress: true,
		restMapperCache:    restmapper.New(),
		proxyURL:           proxy,
	}, nil
}

type Provider struct {
	seedClient         ctrlruntimeclient.Client
	useExternalAddress bool
	proxyURL           string

	// We keep the existing cluster mappings to avoid the discovery on each call to the API server
	restMapperCache *restmapper.Cache
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster. For internal use
// by ourselves only.
func (p *Provider) GetAdminKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) ([]byte, error) {
	s := &corev1.Secret{}
	if err := p.seedClient.Get(ctx, types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.InternalUserClusterAdminKubeconfigSecretName}, s); err != nil {
		return nil, err
	}
	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	if p.useExternalAddress {
		replacedConfig, err := setExternalAddress(c, d)
		if err != nil {
			return nil, err
		}

		d = replacedConfig
	}

	if p.proxyURL != "" {
		replacedConfig, err := setProxyURL(p.proxyURL, d)
		if err != nil {
			return nil, err
		}

		d = replacedConfig
	}

	return d, nil
}

func setExternalAddress(c *kubermaticv1.Cluster, config []byte) ([]byte, error) {
	cfg, err := clientcmd.Load(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	address := c.Status.Address
	for _, cluster := range cfg.Clusters {
		cluster.Server = address.URL
	}
	data, err := clientcmd.Write(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return data, nil
}

func setProxyURL(proxyUrl string, config []byte) ([]byte, error) {
	cfg, err := clientcmd.Load(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	for _, cluster := range cfg.Clusters {
		cluster.ProxyURL = proxyUrl
	}
	data, err := clientcmd.Write(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	return data, nil
}

// ConfigOption defines a function that applies additional configuration to restclient.Config in a generic way.
type ConfigOption func(*restclient.Config) *restclient.Config

// GetClientConfig returns the client config used for initiating a connection for the given cluster.
func (p *Provider) GetClientConfig(ctx context.Context, c *kubermaticv1.Cluster, options ...ConfigOption) (*restclient.Config, error) {
	b, err := p.GetAdminKubeconfig(ctx, c)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.Load(b)
	if err != nil {
		return nil, err
	}

	if p.useExternalAddress {
		address := c.Status.Address
		for _, cluster := range cfg.Clusters {
			cluster.Server = address.URL
		}
	}

	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		"",
		&clientcmd.ConfigOverrides{},
		nil,
	)

	clientConfig, err := iconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Avoid blocking of the controller by increasing the QPS for user cluster interaction
	clientConfig.QPS = 20
	clientConfig.Burst = 50

	// apply all options
	for _, opt := range options {
		clientConfig = opt(clientConfig)
	}

	return clientConfig, err
}

// GetClient returns a dynamic client.
func (p *Provider) GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...ConfigOption) (ctrlruntimeclient.Client, error) {
	config, err := p.GetClientConfig(ctx, c, options...)
	if err != nil {
		return nil, err
	}

	return p.restMapperCache.Client(config)
}

// GetK8sClient returns a k8s go client.
func (p *Provider) GetK8sClient(ctx context.Context, c *kubermaticv1.Cluster, options ...ConfigOption) (kubernetes.Interface, error) {
	config, err := p.GetClientConfig(ctx, c, options...)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
