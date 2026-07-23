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

// Package admingroupcontroller contains a controller that keeps User.Spec.IsAdmin
// in sync with the admin group names configured in KubermaticSetting.Spec.AdminGroups.
//
// Users carrying one of the configured OIDC groups are promoted to administrator and
// stamped with the AdminGrantedByGroupAnnotation. When the group is removed from the
// list, or the user leaves the group, the annotated user is demoted. Users without
// the annotation (manual admins, first-user auto-admin, service accounts) are never
// modified.
package admingroupcontroller
