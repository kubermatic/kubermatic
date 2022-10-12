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

package provider

import (
	"context"
	"net/http"

	"go.anx.io/go-anxcloud/pkg/client"
	"go.anx.io/go-anxcloud/pkg/vlan"
	"go.anx.io/go-anxcloud/pkg/vsphere/provisioning/templates"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func ListAnexiaVlans(ctx context.Context, token string) (apiv1.AnexiaVlanList, error) {
	response := apiv1.AnexiaVlanList{}

	cli, err := getClient(token)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}
	v := vlan.NewAPI(cli)
	vlans, err := v.List(ctx, 1, 1000, "")
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}

	for _, vlan := range vlans {
		apiVlan := apiv1.AnexiaVlan{
			ID: vlan.Identifier,
		}
		response = append(response, apiVlan)
	}

	return response, nil
}

func ListAnexiaTemplates(ctx context.Context, token, locationID string) (apiv1.AnexiaTemplateList, error) {
	response := apiv1.AnexiaTemplateList{}

	cli, err := getClient(token)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}
	t := templates.NewAPI(cli)
	templates, err := t.List(ctx, locationID, "templates", 1, 1000)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}

	for _, template := range templates {
		apiTemplate := apiv1.AnexiaTemplate{
			ID: template.ID,
		}
		response = append(response, apiTemplate)
	}

	return response, nil
}

func getClient(token string) (client.Client, error) {
	tokenOpt := client.TokenFromString(token)
	return client.New(tokenOpt)
}
