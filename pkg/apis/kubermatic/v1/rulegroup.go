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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// RuleGroupResourceName represents "Resource" defined in Kubernetes
	RuleGroupResourceName = "rulegroups"

	// RuleGroupKindName represents "Kind" defined in Kubernetes
	RuleGroupKindName = "RuleGroup"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RuleGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RuleGroupSpec `json:"spec,omitempty"`
}

type RuleGroupSpec struct {
	// RuleGroupType is the type of this ruleGroup applies to. It can be `Metrics`.
	RuleGroupType RuleGroupType `json:"ruleGroupType"`
	// Cluster is the reference to the cluster the ruleGroup should be created in.
	Cluster corev1.ObjectReference `json:"cluster"`
	// Data contains the RuleGroup data. Ref: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#rule_group
	Data []byte `json:"data"`
}

type RuleGroupType string

const (
	// RuleGroupTypeMetrics means the RuleGroup defines the rules to generate alerts from metrics.
	RuleGroupTypeMetrics RuleGroupType = "Metrics"
	// RuleGroupTypeLogs means the RuleGroup defines the rules to generate alerts from logs.
	RuleGroupTypeLogs RuleGroupType = "Logs"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RuleGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RuleGroup `json:"items"`
}
