/*
Copyright 2017 The Kubernetes Authors.

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

package config

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// AuthenticationInfoResolverWrapper can be used to inject Dial function to the
// rest.Config generated by the resolver.
type AuthenticationInfoResolverWrapper func(AuthenticationInfoResolver) AuthenticationInfoResolver

// AuthenticationInfoResolver builds rest.Config base on the server or service
// name and service namespace.
type AuthenticationInfoResolver interface {
	// ClientConfigFor builds rest.Config based on the server.
	ClientConfigFor(server string) (*rest.Config, error)
	// ClientConfigForService builds rest.Config based on the serviceName and
	// serviceNamespace.
	ClientConfigForService(serviceName, serviceNamespace string) (*rest.Config, error)
}

// AuthenticationInfoResolverDelegator implements AuthenticationInfoResolver.
type AuthenticationInfoResolverDelegator struct {
	ClientConfigForFunc        func(server string) (*rest.Config, error)
	ClientConfigForServiceFunc func(serviceName, serviceNamespace string) (*rest.Config, error)
}

func (a *AuthenticationInfoResolverDelegator) ClientConfigFor(server string) (*rest.Config, error) {
	return a.ClientConfigForFunc(server)
}

func (a *AuthenticationInfoResolverDelegator) ClientConfigForService(serviceName, serviceNamespace string) (*rest.Config, error) {
	return a.ClientConfigForServiceFunc(serviceName, serviceNamespace)
}

type defaultAuthenticationInfoResolver struct {
	kubeconfig clientcmdapi.Config
}

// NewDefaultAuthenticationInfoResolver generates an AuthenticationInfoResolver
// that builds rest.Config based on the kubeconfig file. kubeconfigFile is the
// path to the kubeconfig.
func NewDefaultAuthenticationInfoResolver(kubeconfigFile string) (AuthenticationInfoResolver, error) {
	if len(kubeconfigFile) == 0 {
		return &defaultAuthenticationInfoResolver{}, nil
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigFile
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	clientConfig, err := loader.RawConfig()
	if err != nil {
		return nil, err
	}

	return &defaultAuthenticationInfoResolver{kubeconfig: clientConfig}, nil
}

func (c *defaultAuthenticationInfoResolver) ClientConfigFor(server string) (*rest.Config, error) {
	return c.clientConfig(server)
}

func (c *defaultAuthenticationInfoResolver) ClientConfigForService(serviceName, serviceNamespace string) (*rest.Config, error) {
	return c.clientConfig(serviceName + "." + serviceNamespace + ".svc")
}

func (c *defaultAuthenticationInfoResolver) clientConfig(target string) (*rest.Config, error) {
	// exact match
	if authConfig, ok := c.kubeconfig.AuthInfos[target]; ok {
		return restConfigFromKubeconfig(authConfig)
	}

	// star prefixed match
	serverSteps := strings.Split(target, ".")
	for i := 1; i < len(serverSteps); i++ {
		nickName := "*." + strings.Join(serverSteps[i:], ".")
		if authConfig, ok := c.kubeconfig.AuthInfos[nickName]; ok {
			return restConfigFromKubeconfig(authConfig)
		}
	}

	// if we're trying to hit the kube-apiserver and there wasn't an explicit config, use the in-cluster config
	if target == "kubernetes.default.svc" {
		// if we can find an in-cluster-config use that.  If we can't, fall through.
		inClusterConfig, err := rest.InClusterConfig()
		if err == nil {
			return setGlobalDefaults(inClusterConfig), nil
		}
	}

	// star (default) match
	if authConfig, ok := c.kubeconfig.AuthInfos["*"]; ok {
		return restConfigFromKubeconfig(authConfig)
	}

	// use the current context from the kubeconfig if possible
	if len(c.kubeconfig.CurrentContext) > 0 {
		if currContext, ok := c.kubeconfig.Contexts[c.kubeconfig.CurrentContext]; ok {
			if len(currContext.AuthInfo) > 0 {
				if currAuth, ok := c.kubeconfig.AuthInfos[currContext.AuthInfo]; ok {
					return restConfigFromKubeconfig(currAuth)
				}
			}
		}
	}

	// anonymous
	return setGlobalDefaults(&rest.Config{}), nil
}

func restConfigFromKubeconfig(configAuthInfo *clientcmdapi.AuthInfo) (*rest.Config, error) {
	config := &rest.Config{}

	// blindly overwrite existing values based on precedence
	if len(configAuthInfo.Token) > 0 {
		config.BearerToken = configAuthInfo.Token
	} else if len(configAuthInfo.TokenFile) > 0 {
		tokenBytes, err := ioutil.ReadFile(configAuthInfo.TokenFile)
		if err != nil {
			return nil, err
		}
		config.BearerToken = string(tokenBytes)
	}
	if len(configAuthInfo.Impersonate) > 0 {
		config.Impersonate = rest.ImpersonationConfig{
			UserName: configAuthInfo.Impersonate,
			Groups:   configAuthInfo.ImpersonateGroups,
			Extra:    configAuthInfo.ImpersonateUserExtra,
		}
	}
	if len(configAuthInfo.ClientCertificate) > 0 || len(configAuthInfo.ClientCertificateData) > 0 {
		config.CertFile = configAuthInfo.ClientCertificate
		config.CertData = configAuthInfo.ClientCertificateData
		config.KeyFile = configAuthInfo.ClientKey
		config.KeyData = configAuthInfo.ClientKeyData
	}
	if len(configAuthInfo.Username) > 0 || len(configAuthInfo.Password) > 0 {
		config.Username = configAuthInfo.Username
		config.Password = configAuthInfo.Password
	}
	if configAuthInfo.AuthProvider != nil {
		return nil, fmt.Errorf("auth provider not supported")
	}

	return setGlobalDefaults(config), nil
}

func setGlobalDefaults(config *rest.Config) *rest.Config {
	config.UserAgent = "kube-apiserver-admission"
	config.Timeout = 30 * time.Second

	return config
}
