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

package vsphere

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type RESTSession struct {
	Client *rest.Client
}

func newRESTSession(ctx context.Context, dc *kubermaticv1.DatacenterSpecVSphere, username, password string, caBundle *x509.CertPool) (*RESTSession, error) {
	endpoint, err := url.Parse(dc.Endpoint)
	if err != nil {
		return nil, err
	}

	u := endpoint.JoinPath("/sdk")

	// creating the govmoni Client in roundabout way because we need to set the proper CA bundle: reference https://github.com/vmware/govmomi/issues/1200
	soapClient := soap.NewClient(u, dc.AllowInsecure)
	// set our CA bundle
	soapClient.DefaultTransport().TLSClientConfig.RootCAs = caBundle

	vim25Client, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, err
	}

	client := rest.NewClient(vim25Client)

	user := url.UserPassword(username, password)
	if dc.InfraManagementUser != nil {
		user = url.UserPassword(dc.InfraManagementUser.Username, dc.InfraManagementUser.Password)
	}

	if err = client.Login(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return &RESTSession{
		Client: client,
	}, nil
}

// Logout closes the idling vCenter connections.
func (s *RESTSession) Logout(ctx context.Context) {
	if err := s.Client.Logout(ctx); err != nil {
		utilruntime.HandleError(fmt.Errorf("vsphere REST client failed to logout: %w", err))
	}
}
