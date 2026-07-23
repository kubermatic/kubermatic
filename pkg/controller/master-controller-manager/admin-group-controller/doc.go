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
// A user carrying one of the configured OIDC groups (populated into User.Spec.Groups
// at login) is promoted to administrator and stamped with the
// AdminGrantedByGroupAnnotation naming the granting group. When the group is removed
// from the list, or the user leaves the group, the annotated user is demoted and the
// annotation removed. Users without the annotation (manual admins, the first-user
// auto-admin, service accounts) are never modified.
//
// The controller deliberately does NOT use the worker-name mechanism
// (workerlabel.Predicate): its predicates gate on object name and field changes
// only, never on labels. Running it locally therefore requires no worker-name
// labelling of Users or the KubermaticSetting, unlike the Cluster controllers in
// the seed-controller-manager. The trade-off is that a locally-run copy processes
// the same objects as a deployed copy; that is safe because the reconcile is
// convergent — both instances compute the same desired state from the same inputs
// and later passes are no-ops (see TestTwoControllerInstancesConverge). If
// local-vs-deployed isolation is ever wanted, wire workerlabel.Predicate into both
// watches and accept that local testing then requires labelling each test User.
package admingroupcontroller
