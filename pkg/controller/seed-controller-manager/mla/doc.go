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

/*
Package mla contains controllers that are responsible for configuring MLA (Monitoring, Logging, and Alerting)
for user clusters.
- org grafana controller - create/update/delete Grafana organizations based on Kubermatic Projects
- user grafana controller - create/update/delete Grafana Users to organizations based on Kubermatic UserProjectBindings
- datasource grafana controller - create/update/delete Grafana Datasources to organizations based on Kubermatic Clusters
- alertmanager configuration controller - manage alertmanager configuration based on Kubermatic Clusters
*/
package mla
