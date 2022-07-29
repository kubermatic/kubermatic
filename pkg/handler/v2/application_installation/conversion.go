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

func convertInternalToAPIApplicationInstallation(app *appskubermaticv1.ApplicationInstallation) *apiv2.ApplicationInstallation {
	var apiCondition []apiv2.ApplicationInstallationCondition

	for condType, condition := range app.Status.Conditions {
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

	return &apiv2.ApplicationInstallation{
		ObjectMeta: apiv1.ObjectMeta{
			CreationTimestamp: apiv1.Time(app.CreationTimestamp),
			Name:              app.Name,
		},
		Namespace: app.Namespace,
		Spec:      &app.Spec,
		Status: &apiv2.ApplicationInstallationStatus{
			Conditions:         apiCondition,
			ApplicationVersion: app.Status.ApplicationVersion,
			Method:             app.Status.Method,
		},
	}
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
		Spec: *app.Spec,
	}
}
