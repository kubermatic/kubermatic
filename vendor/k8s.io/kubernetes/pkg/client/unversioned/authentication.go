/*
Copyright 2016 The Kubernetes Authors.

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

package unversioned

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	"k8s.io/kubernetes/pkg/apis/authentication"
	"k8s.io/kubernetes/pkg/client/restclient"
)

type AuthenticationInterface interface {
	TokenReviewsInterface
}

// AuthenticationClient is used to interact with Kubernetes authentication features.
type AuthenticationClient struct {
	*restclient.RESTClient
}

func (c *AuthenticationClient) TokenReviews() TokenReviewInterface {
	return newTokenReviews(c)
}

func NewAuthentication(c *restclient.Config) (*AuthenticationClient, error) {
	config := *c
	if err := setAuthenticationDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AuthenticationClient{client}, nil
}

func NewAuthenticationOrDie(c *restclient.Config) *AuthenticationClient {
	client, err := NewAuthentication(c)
	if err != nil {
		panic(err)
	}
	return client
}

func setAuthenticationDefaults(config *restclient.Config) error {
	// if authentication group is not registered, return an error
	g, err := registered.Group(authentication.GroupName)
	if err != nil {
		return err
	}
	config.APIPath = defaultAPIPath
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	// TODO: Unconditionally set the config.Version, until we fix the config.
	//if config.Version == "" {
	copyGroupVersion := g.GroupVersion
	config.GroupVersion = &copyGroupVersion
	//}

	config.NegotiatedSerializer = api.Codecs
	return nil
}
