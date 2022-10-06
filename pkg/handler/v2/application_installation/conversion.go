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

package applicationinstallation

import (
	"sort"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func convertInternalToAPIApplicationInstallation(in *appskubermaticv1.ApplicationInstallation) *apiv2.ApplicationInstallation {
	out := &apiv2.ApplicationInstallation{
		ObjectMeta: apiv1.ObjectMeta{
			CreationTimestamp: apiv1.Time(in.CreationTimestamp),
			Name:              in.Name,
		},
		Namespace: in.Namespace,
		Spec: &apiv2.ApplicationInstallationSpec{
			Namespace: apiv2.NamespaceSpec{
				Name:        in.Spec.Namespace.Name,
				Create:      in.Spec.Namespace.Create,
				Labels:      in.Spec.Namespace.Labels,
				Annotations: in.Spec.Namespace.Annotations,
			},
			ApplicationRef: in.Spec.ApplicationRef,
			Values:         in.Spec.Values,
		},
		Status: &apiv2.ApplicationInstallationStatus{
			ApplicationVersion: in.Status.ApplicationVersion,
			Method:             in.Status.Method,
		},
	}

	var apiCondition []apiv2.ApplicationInstallationCondition
	for condType, condition := range in.Status.Conditions {
		apiCondition = append(apiCondition, apiv2.ApplicationInstallationCondition{
			Type:               condType,
			Status:             condition.Status,
			LastHeartbeatTime:  apiv1.NewTime(condition.LastHeartbeatTime.Time),
			LastTransitionTime: apiv1.NewTime(condition.LastTransitionTime.Time),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	// ensure a stable sorting order
	sort.Slice(apiCondition, func(i, j int) bool {
		return apiCondition[i].Type < apiCondition[j].Type
	})
	out.Status.Conditions = apiCondition

	if in.DeletionTimestamp != nil {
		ts := apiv1.NewTime(in.DeletionTimestamp.Time)
		out.DeletionTimestamp = &ts
	}

	return out
}

func convertInternalToAPIApplicationInstallationForList(in *appskubermaticv1.ApplicationInstallation) *apiv2.ApplicationInstallationListItem {
	out := &apiv2.ApplicationInstallationListItem{
		Name:              in.Name,
		CreationTimestamp: apiv1.Time(in.CreationTimestamp),
		Spec: &apiv2.ApplicationInstallationListItemSpec{
			Namespace: apiv2.NamespaceSpec{
				Name:        in.Spec.Namespace.Name,
				Create:      in.Spec.Namespace.Create,
				Labels:      in.Spec.Namespace.Labels,
				Annotations: in.Spec.Namespace.Annotations,
			},
			ApplicationRef: in.Spec.ApplicationRef,
		},
		Status: &apiv2.ApplicationInstallationListItemStatus{
			Method:             in.Status.Method,
			ApplicationVersion: in.Status.ApplicationVersion,
		},
	}

	// TODO @vgramer decide if we want to extract this into a separate method, as duplicate code is used in convertInternalToAPIApplicationInstallation
	// TODO personally I don't have a strong preference
	var apiCondition []apiv2.ApplicationInstallationCondition
	for condType, condition := range in.Status.Conditions {
		apiCondition = append(apiCondition, apiv2.ApplicationInstallationCondition{
			Type:               condType,
			Status:             condition.Status,
			LastHeartbeatTime:  apiv1.NewTime(condition.LastHeartbeatTime.Time),
			LastTransitionTime: apiv1.NewTime(condition.LastTransitionTime.Time),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}
	// ensure a stable sorting order
	sort.Slice(apiCondition, func(i, j int) bool {
		return apiCondition[i].Type < apiCondition[j].Type
	})
	out.Status.Conditions = apiCondition

	return out
}

func convertAPItoInternalApplicationInstallationBody(app *apiv2.ApplicationInstallationBody) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			Kind:       appskubermaticv1.ApplicationInstallationKindName,
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name:        app.Spec.Namespace.Name,
				Create:      app.Spec.Namespace.Create,
				Labels:      app.Spec.Namespace.Labels,
				Annotations: app.Spec.Namespace.Annotations,
			},
			ApplicationRef: app.Spec.ApplicationRef,
			Values:         app.Spec.Values,
		},
	}
}
