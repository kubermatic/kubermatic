/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

/*
Package httproutegatewaysync contains a controller that synchronizes HTTPRoute
hostnames to Gateway listeners for cert-manager integration.

cert-manager requires Gateway listeners to have explicit hostnames to create
certificates. KKP uses a shared Gateway model where multiple HTTPRoutes from
different namespaces attach to a single Gateway. This controller watches
HTTPRoutes and dynamically adds HTTPS listeners with explicit hostnames,
allowing cert-manager to create certificates for each hostname.

The controller watches HTTPRoutes in specified namespaces (mla, monitoring by default)
and ensures the Gateway has HTTPS listeners with explicit hostnames. It uses
stateless reconciliation - on each reconcile, it lists all HTTPRoutes, extracts
unique hostnames, and computes the desired listener state.

HTTPS listeners are named using the pattern: <sanitized-hostname>
For example, hostname "grafana.lab.kubermatic.io" becomes "grafana-lab-kubermatic-io".
Note that Gateway API doesn't allow duplicate hostnames on the same port so that's why we use sanitized hostnames.
More importantly, cert-manager would try to create two certificates for the same hostname, which is unnecessary.

Certificate/Secret names follow the pattern: <namespace>-<httproute-name>
For example, HTTPRoute "mla/grafana-iap" creates a secret named "mla-grafana-iap".
*/
package httproutegatewaysync
