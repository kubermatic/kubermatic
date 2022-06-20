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
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// dashboardProxyDirector is responsible for adjusting proxy request, so we can properly access Kubernetes Dashboard.
type dashboardProxyDirector struct {
	proxyURL        *url.URL
	token           string
	originalRequest *http.Request
}

func (director *dashboardProxyDirector) director() func(*http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = director.proxyURL.Scheme
		req.URL.Host = director.proxyURL.Host
		req.Host = director.proxyURL.Host
		req.URL.Path = director.getBasePath(director.originalRequest.URL.Path)

		req.Header.Set("Authorization", director.getAuthorizationHeader())
		req.Header.Set("X-Forwarded-Host", director.originalRequest.Header.Get("Host"))
	}
}

func (director *dashboardProxyDirector) getAuthorizationHeader() string {
	return fmt.Sprintf("Bearer %s", director.token)
}

// We need to get proper path to Dashboard API and strip the URL from the Kubermatic API request part.
func (director *dashboardProxyDirector) getBasePath(path string) string {
	separator := "proxy"
	if !strings.Contains(path, separator) {
		return "/"
	}

	parts := strings.Split(path, separator)
	if len(parts) != 2 {
		return "/"
	}

	return parts[1]
}

func newDashboardProxyDirector(proxyURL *url.URL, token string, request *http.Request) *dashboardProxyDirector {
	return &dashboardProxyDirector{
		proxyURL:        proxyURL,
		token:           token,
		originalRequest: request,
	}
}
