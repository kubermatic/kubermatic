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

package etcd

import (
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type etcdBackupConfigReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	BackupSchedule() time.Duration
}

// BackupConfigReconciler returns the function to reconcile the EtcdBackupConfigs.
func BackupConfigReconciler(data etcdBackupConfigReconcilerData, seed *kubermaticv1.Seed) reconciling.NamedEtcdBackupConfigReconcilerFactory {
	return func() (string, reconciling.EtcdBackupConfigReconciler) {
		return resources.EtcdDefaultBackupConfigName, func(config *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
			if config.Labels == nil {
				config.Labels = make(map[string]string)
			}
			if data.Cluster().Labels != nil {
				config.Labels[kubermaticv1.ProjectIDLabelKey] = data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey]
			}

			backupScheduleString, err := parseDuration(data.BackupSchedule())
			if err != nil {
				return nil, fmt.Errorf("failed to parse backup duration: %w", err)
			}
			config.Spec.Name = resources.EtcdDefaultBackupConfigName
			config.Spec.Schedule = backupScheduleString
			keep := 20
			config.Spec.Keep = &keep
			config.Spec.Cluster = corev1.ObjectReference{
				Kind:       kubermaticv1.ClusterKindName,
				Name:       data.Cluster().Name,
				UID:        data.Cluster().UID,
				APIVersion: "kubermatic.k8c.io/v1",
			}

			if seed.IsDefaultEtcdAutomaticBackupEnabled() {
				config.Spec.Destination = seed.Spec.EtcdBackupRestore.DefaultDestination
			}

			return config, nil
		}
	}
}

func parseDuration(interval time.Duration) (string, error) {
	scheduleString := fmt.Sprintf("@every %vm", interval.Round(time.Minute).Minutes())
	// We verify the validity of the scheduleString here, because the etcd_backup_controller
	// only does that inside its sync loop, which means it is entirely possible to create
	// an EtcdBackupConfig with an invalid Spec.Schedule
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	_, err := parser.Parse(scheduleString)
	if err != nil {
		return "", err
	}
	return scheduleString, nil
}
