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

package v2

import (
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   v1beta1.ConstraintTemplateSpec   `json:"spec"`
	Status v1beta1.ConstraintTemplateStatus `json:"status"`
}

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name string `json:"name"`

	Spec   v1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
}

// Violation represents a gatekeeper constraint violation
type Violation struct {
	EnforcementAction string `json:"enforcementAction,omitempty"`
	Kind              string `json:"kind,omitempty"`
	Message           string `json:"message,omitempty"`
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
}
