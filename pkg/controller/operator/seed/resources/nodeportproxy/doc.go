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

// Package nodeportproxy is responsible for reconciling a seed-cluster-wide
// proxy based on Envoy and a custom envoy-manager/lb-updater tools. They
// monitor Cluster resources and allocate a port on a shared LoadBalancer
// service to access the user cluster's control plane.
//
// Note that there is also pkg/resources/nodeportproxy/, which is a special,
// per-Cluster nodeport-proxy. The package are similar, but not similar
// enough to merge them together.
package nodeportproxy
