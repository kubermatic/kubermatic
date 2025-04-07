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

package runner

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func logEventsForAllMachines(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	machines := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machines); err != nil {
		log.Errorw("Failed to list machines", zap.Error(err))
		return
	}

	for _, machine := range machines.Items {
		machineLog := log.With("machine", machine.Name)
		machineLog.Info("Logging events for machine")
		if err := logEventsObject(ctx, machineLog, client, machine.Namespace, machine.UID); err != nil {
			machineLog.Errorw("Failed to log events for machine", "namespace", machine.Namespace, zap.Error(err))
		}
	}
}

func logEventsObject(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, uid types.UID) error {
	events := &corev1.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     namespace,
		FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(uid)),
	}
	if err := client.List(ctx, events, listOpts); err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	for _, event := range events.Items {
		var msg string
		if event.Type == corev1.EventTypeWarning {
			// Make sure this gets highlighted
			msg = "ERROR"
		}
		log.Infow(
			msg,
			"EventType", event.Type,
			"Number", event.Count,
			"Reason", event.Reason,
			"Message", event.Message,
			"Source", event.Source.Component,
		)
	}
	return nil
}
