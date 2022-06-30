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

package kubernetesdashboard

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

type Handler interface {
	Install(*mux.Router)
	Middlewares(...endpoint.Middleware) Handler
	Options(...httptransport.ServerOption) Handler
}

type baseHandler struct {
	middlewares []endpoint.Middleware
	// Let's ignore the "structcheck" false positive since it is only an "abstract" handler and options are used by the
	// loginHandler and proxyHandler structure that embed the baseHandler.
	// disabled due to Go 1.18 compat issues: nolint:structcheck
	options []httptransport.ServerOption
}

func (this *baseHandler) chain(endpoint endpoint.Endpoint) endpoint.Endpoint {
	if len(this.middlewares) > 0 {
		for i := len(this.middlewares) - 1; i >= 0; i-- {
			endpoint = this.middlewares[i](endpoint)
		}
	}

	return endpoint
}

func isEnabled(ctx context.Context, settingsProvider provider.SettingsProvider) error {
	settings, err := settingsProvider.GetGlobalSettings(ctx)

	if err != nil {
		return utilerrors.New(http.StatusInternalServerError, "could not read global settings")
	}

	if !settings.Spec.EnableDashboard {
		return utilerrors.New(http.StatusForbidden, "Kubernetes Dashboard access is disabled by the global settings")
	}

	return nil
}
