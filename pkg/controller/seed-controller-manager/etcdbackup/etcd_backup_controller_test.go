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

package etcdbackup

import (
	"context"
	"github.com/go-test/deep"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func genTestCluster() *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *semver.NewSemverOrDie("1.16.3"),
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "testnamespace",
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver: kubermaticv1.HealthStatusUp,
			},
		},
	}
}

func genBackupConfig(cluster *kubermaticv1.Cluster, name string) *kubermaticv1.EtcdBackupConfig {
	return &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.EtcdBackupConfigSpec{
			Name: name,
			Cluster: corev1.ObjectReference{
				Kind: kubermaticv1.ClusterKindName,
				Name: cluster.GetName(),
			},
		},
	}
}

func genStoreContainer() *corev1.Container {
	return &corev1.Container{
		Name:  "test-store-container",
		Image: "some-s3cmd:latest",
		Command: []string{
			"/bin/sh",
			"-c",
			"s3cmd ...",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "FOO",
				Value: "xx",
			},
			{
				Name:  "BAR",
				Value: "yy",
			},
		},
	}
}

func genDeleteContainer() *corev1.Container {
	return &corev1.Container{
		Name:  "test-delete-container",
		Image: "some-s3cmd:latest",
		Command: []string{
			"/bin/sh",
			"-c",
			"s3cmd ...",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "FOO",
				Value: "xx",
			},
			{
				Name:  "BAR",
				Value: "yy",
			},
		},
	}
}

func genCleanupContainer() *corev1.Container {
	return &corev1.Container{
		Name:  "test-cleanup-container",
		Image: "some-s3cmd:latest",
		Command: []string{
			"/bin/sh",
			"-c",
			"s3cmd ...",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "FOO",
				Value: "xx",
			},
			{
				Name:  "BAR",
				Value: "yy",
			},
		},
	}
}

func genBackupJob(backupName string, jobName string) *batchv1.Job {
	// jerry-rig a cluster, BackupConfig and BackupStatus instance to create a job object
	// that's similar to the ones an actual reconciliation will create
	cluster := genTestCluster()
	backupConfig := genBackupConfig(cluster, "testbackup")
	backup := &kubermaticv1.BackupStatus{
		BackupName: backupName,
		JobName:    jobName,
	}
	reconciler := Reconciler{
		log:            kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		Client:         ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, cluster, backupConfig),
		storeContainer: genStoreContainer(),
		recorder:       record.NewFakeRecorder(10),
		clock:          clock.RealClock{},
	}
	job := reconciler.backupJob(backupConfig, cluster, backup)
	job.ResourceVersion = "1"
	return job
}

func genDeleteJob(backupName string, jobName string) *batchv1.Job {
	// same thing as genBackupJob, but for delete jobs
	cluster := genTestCluster()
	backupConfig := genBackupConfig(cluster, "testbackup")
	backup := &kubermaticv1.BackupStatus{
		BackupName:    backupName,
		DeleteJobName: jobName,
	}
	reconciler := Reconciler{
		log:             kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		Client:          ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, cluster, backupConfig),
		deleteContainer: genDeleteContainer(),
		recorder:        record.NewFakeRecorder(10),
		clock:           clock.RealClock{},
	}
	job := reconciler.deleteJob(backupConfig, cluster, backup)
	job.ResourceVersion = "1"
	return job
}

func genCleanupJob(jobName string) *batchv1.Job {
	// same thing as genBackupJob, but for cleanup jobs
	cluster := genTestCluster()
	backupConfig := genBackupConfig(cluster, "testbackup")
	reconciler := Reconciler{
		log:              kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		Client:           ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, cluster, backupConfig),
		cleanupContainer: genCleanupContainer(),
		recorder:         record.NewFakeRecorder(10),
		clock:            clock.RealClock{},
	}
	job := reconciler.cleanupJob(backupConfig, cluster, jobName)
	job.ResourceVersion = "1"
	return job
}

func jobAddCondition(j *batchv1.Job, jobType batchv1.JobConditionType, status corev1.ConditionStatus, lastTransitionTime time.Time, message string) *batchv1.Job {
	j.Status.Conditions = append(j.Status.Conditions, batchv1.JobCondition{
		Type:               jobType,
		Status:             status,
		LastTransitionTime: metav1.Time{Time: lastTransitionTime},
		Message:            message,
	})
	return j
}

func genClusterRootCaSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CASecretName,
			Namespace: genTestCluster().Status.NamespaceName,
		},
		Data: map[string][]byte{
			"ca.crt": kubermaticv1.NewBytes("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURBakNDQWVxZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREF5TVRBd0xnWURWUVFERXlkeWIyOTAKTFdOaExtSm5aalp1T1dweU9HTXVaR0pzTVM1a1pYWXViV1YwWVd0MVltVXVaR1V3SGhjTk1qQXhNREEzTURFMApPVEl5V2hjTk16QXhNREExTURFME9USXlXakF5TVRBd0xnWURWUVFERXlkeWIyOTBMV05oTG1KblpqWnVPV3B5Ck9HTXVaR0pzTVM1a1pYWXViV1YwWVd0MVltVXVaR1V3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXcKZ2dFS0FvSUJBUURBMS9WbTFFOG51cUhIOUxtUWREWkc1K3h3MVZ6MkcxQnAxSC9QUFJ0WE9aajhEWkl4ZzBiSQpydHNmdXRPWUZQZ1dYdEpWdHFvSUxaYUt3VkhSTVNYZ1Q1YXFJTFQ3bm5SYmhiWFdtR2tEVWV0T3g4WGZ0UjhNCjByeHc1RHhUSlZSeVBzWDlORUpsN1lTKzFrVC9jcEQvK2tlbisyckNBaXp2MEthb3Ezd0RuTnBlVEphV04xeTkKeHNKakpNZGJzNlZwTU9pSm4wWjJidFVNTUdqVHJodjVkK24vV2YrQ09aK1g0eHJ2NzFROHJxZjIraVFoeDhJWAppc0tnQ0Rrc3hucFhFbkJreFlQdnlhbjErUm5zOTZkUnJNZWY1NWNtc0czYkcyMjVpeTlqN0E4LzNhZXR6Si9JCjVHOFUvVXc3YlVpdjRtUDdYZkE1V0V3STRhKzNGb3hGQWdNQkFBR2pJekFoTUE0R0ExVWREd0VCL3dRRUF3SUMKcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFDa2tNeHJkNkdHSXljVwpWL3c3bHdkbWRlaHIxOW9kQVJGczgyYnhOWVFBdXI0MzR1K0JmbjBqOFB5VXJUd0NUUEROVUt2OGcrL2V3Wm1LCndnQUFTNklLVXFGcmtDNVpUMU1aMVNlKzJmdmpwS0ZxVUxoamV4ZUI5RjMrWVRuQ3BPUGt0S0JtU21CVTFHU0oKV3FkK2ZwWGszektoVFFXVlE4UlZHYUExenZXSHltUTNCZlo1aDVrWG10bDVWOUp3RU5vdVMveHVWd0FndjFjcApaekZYQ1luQVREM3d5K0N5NzNEUVU1MC9hSHZwclUxTGcrKzg1ZUF5amJhTTRIVVBWT2YwdHNzeVMvREd3aUptCnovL1FiaUZpTWt2UjJZWTNBK1ZQb3V4SS9rM2IxMXFJdm9qMG9nVFkyNitJb1lDeGhJOHJkTU1Bb2M0ODRTdWQKM3dkZ05hcXQKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="),
			"ca.key": kubermaticv1.NewBytes("LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBd05mMVp0UlBKN3FoeC9TNWtIUTJSdWZzY05WYzlodFFhZFIvenowYlZ6bVkvQTJTCk1ZTkd5SzdiSDdyVG1CVDRGbDdTVmJhcUNDMldpc0ZSMFRFbDRFK1dxaUMwKzU1MFc0VzExcGhwQTFIclRzZkYKMzdVZkROSzhjT1E4VXlWVWNqN0YvVFJDWmUyRXZ0WkUvM0tRLy9wSHAvdHF3Z0lzNzlDbXFLdDhBNXphWGt5VwpsamRjdmNiQ1l5VEhXN09sYVREb2laOUdkbTdWRERCbzA2NGIrWGZwLzFuL2dqbWZsK01hNys5VVBLNm45dm9rCkljZkNGNHJDb0FnNUxNWjZWeEp3Wk1XRDc4bXA5ZmtaN1BlblVhekhuK2VYSnJCdDJ4dHR1WXN2WSt3UFA5Mm4KcmN5ZnlPUnZGUDFNTzIxSXIrSmorMTN3T1ZoTUNPR3Z0eGFNUlFJREFRQUJBb0lCQUhzR0NvOFVLSDh1NStDWgpOZng2dHRtYlFWSm1PMHpoOWZYZDA3K1F1eTExR0N6TU41U1FyQXFBeWxlK3B4Z2hZSGRjL0pBajNPc2tzaUpJCjIvbzVnWEZOTk0vSjE4dWh0WmRoZ0xTclBHc2F5cVBTZkdDUHVvUkN1R2JJbzlkalBERTU3TEx3c044M25IVG0KV3RRZjhYTHI1dVM2VnN2MytWVHdaakc2WEg0c2FIcUZ0OE10Ni9taDR0UFViQ2VPaHdCZXlNUUQwOXhrM3k4ZAoyMmVoV1QwMTRWSXBTMklBVFdGKzJkTW13Y1pEOGQ3S2hLeDA4UUE0dlZHUGIrQWxWMVJ3eitkODVkQkwzSkpPClVkWndRamw2VEhaVS9hbmMyT2hDRjFsQXEvWTFKZmFPZWJXbFZvOFRVaE9HL0JpQzd2ZjRvTGs1SVRxYkhIZVQKVExMWUxhMENnWUVBeUZGdCtpNHN1TzFPQnNaSHZ2WnhxNUtSV3NWNXBpRUpzZlNqOEJ0cW5uazkxSjVBZG81aApFYi8rY0lKaXdqQm1wYjk0cTRTYXQ0eHY2ck9Bc3NpZGlCZlhjQWVaTTNFa05QdzhYNHZFdWFiSWdES2svaEkzCm1ubG1ZblVvbG5vUXpOYWg3QUczVkR6eTJNODFrNkpzUjJ5cXZuR3hMSEQ3SjU4dEVSRGF5SWNDZ1lFQTluS2wKY2JGQjJEeWVUU0VyWWtVeGJZMlRnTEhVUnBreWxtVldDK0VGT0NLYXJaTU1UOWpLT1VYRlFpdVFMVzFkVDE2Sgp2QVh0QnhVMDIyeXN2cGo3Z3BUblhzZXlsZTkzVTVFa3RBZ1NIRmh0ejFscGx2eWRBaWUvQUtkYU9EbUJQOEZsCm5RVHc2cWNHM3JLRkJrOE9KZlBpRDE2OW5tMW5RZEx4eE8razA5TUNnWUVBdlJVSDcxL0lmU0lhUlpEQnhrSlAKbDNqNDFTcVRvam1MTWp2T3h1VEtKaDRoTytISXpWK2x4cUJvcG9DY2dkbzMrZm9iQ0NOWit5bUh0bzJMVExiSwo0OXhGVWcwS0VpR1k0SjY2eWlGZkp6S0VEV1pBa1VaV3ord0p0YVFMRk1iUnR0aGQ3U3pOaEtrblBYbVJnL0tMCnJIdXBTNng3Wll5YnRaR3RjMjlxWkY4Q2dZQXJKcW5IUFdVdENuZ2hReVNJZ1ZzRk5wdlVGYzc0U1l5cy9yTlIKUXlZWnpSMU9OUWdiMXZhWmpwamFYQ3hUZCttMW92VDA0Z2k5aTc0RWlZTzVuNm15RklacWR3YlM3K254ek9FagpVS0p6S2h5WUNLelBUZzNqdWJmYzBuQ2VsWnNHNGNMNytraUFuWnc3VkFDc3VSemVFbFRMb2lnTFhGYVBGUE5XCkt5dXVGd0tCZ1FDd3V4RGFRMGduaXkxaUF3T1A1WU94cGw3bXBibFNROGxJaDQrRnJlQTdVWFhGQ3BSamdWdWoKMVdTRS9mSGs2WEZRT3pvcVFibFpYZ0hTREJ4SlF3cTlhUllueXQ5czFSK0FQVzFlVVJLSE9ra1FjYjVNK0QzbQovYkRkRWRUOGlsTFFTWGlWcEVKdDExay9zK3h4ZC9kMFdNL1RNV1VHOVZEVjhmWHVqNmkxWVE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="),
		},
	}
}

func TestEnsureNextBackupIsScheduled(t *testing.T) {
	testCases := []struct {
		name              string
		currentTime       time.Time
		schedule          string
		existingBackups   []kubermaticv1.BackupStatus
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
	}{
		{
			name:            "scheduling on a no-schedule Config with no backups schedules one one-shot backup immediately",
			currentTime:     time.Unix(10, 0).UTC(),
			schedule:        "",
			existingBackups: []kubermaticv1.BackupStatus{},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(10, 0).UTC()},
					BackupName:    "testbackup",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 0,
			},
		},
		{
			name:        "scheduling on a no-schedule Config with a scheduled backup doesn't change anything",
			currentTime: time.Unix(10, 0).UTC(),
			schedule:    "",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(100, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-40",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(100, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-40",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 90 * time.Second,
			},
		},
		{
			name:            "scheduling on a Config with no backups schedules one backup",
			currentTime:     time.Unix(10, 0).UTC(),
			schedule:        "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 590 * time.Second,
			},
		},
		{
			name:        "scheduling on a Config with future backups already scheduled changes nothing",
			currentTime: time.Unix(10, 0).UTC(), //
			schedule:    "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 590 * time.Second,
			},
		},
		{
			name:        "scheduling on a Config with no future backups scheduled schedules the next one",
			currentTime: time.Unix(700, 0).UTC(),
			schedule:    "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(1200, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-20-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 500 * time.Second,
			},
		},
		{
			name:        "scheduling on a Config with a future backups scheduled for a time not matching the schedule schedules a new backup",
			currentTime: time.Unix(700, 0).UTC(),
			schedule:    "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
				{
					// this could only happen if you change the .schedule field of a BackupConfig later.
					ScheduledTime: &metav1.Time{Time: time.Unix(720, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-12-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(600, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-10-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(720, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-12-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(1200, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-20-00",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 500 * time.Second,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Spec.Schedule = tc.schedule
			backupConfig.Status.CurrentBackups = tc.existingBackups

			reconciler := Reconciler{
				log:                 kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:              ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, cluster, backupConfig),
				recorder:            record.NewFakeRecorder(10),
				clock:               clock,
				randStringGenerator: constRandStringGenerator("xxxx"),
			}

			reconcileAfter, err := reconciler.ensureNextBackupIsScheduled(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(backupConfig.Status, readbackBackupConfig.Status); diff != nil {
				t.Errorf("backupsConfig status differs from read back one, diff: %v", diff)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			if deep.Equal(reconcileAfter, tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func TestStartPendingBackupJobs(t *testing.T) {
	testCases := []struct {
		name              string
		currentTime       time.Time
		existingBackups   []kubermaticv1.BackupStatus
		existingJobs      []batchv1.Job
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      []batchv1.Job
	}{
		{
			name:        "backup job scheduled in the past it started, job scheduled in the future is not",
			currentTime: time.Unix(90, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-00",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			existingJobs: []batchv1.Job{},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-00",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: []batchv1.Job{
				*genBackupJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
			},
		},
		{
			name:        "finished backup job is marked as finished in the backup status",
			currentTime: time.Unix(90, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-00",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(70, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-01-10",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: []batchv1.Job{
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-10", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(80, 0).UTC(), "Job has reached the specified backoff limit"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(70, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-10",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "Job has reached the specified backoff limit",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: nil,
			expectedJobs: []batchv1.Job{
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-10", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(80, 0).UTC(), "Job has reached the specified backoff limit"),
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			initObjs := []runtime.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs {
				initObjs = append(initObjs, j.DeepCopy())
			}
			reconciler := Reconciler{
				log:            kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:         ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, initObjs...),
				storeContainer: genStoreContainer(),
				recorder:       record.NewFakeRecorder(10),
				clock:          clock,
			}

			reconcileAfter, err := reconciler.startPendingBackupJobs(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(backupConfig.Status, readbackBackupConfig.Status); diff != nil {
				t.Errorf("backupsConfig status differs from read back one, diff: %v", diff)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			jobs := batchv1.JobList{}
			if err := reconciler.List(context.Background(), &jobs); err != nil {
				t.Fatalf("Error reading created jobs: %v", err)
			}
			if diff := deep.Equal(jobs.Items, tc.expectedJobs); diff != nil {
				t.Errorf("jobs differ from expected ones: %v", diff)
			}

			if deep.Equal(reconcileAfter, tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func TestStartPendingBackupDeleteJobs(t *testing.T) {
	testCases := []struct {
		name              string
		currentTime       time.Time
		keep              int
		existingBackups   []kubermaticv1.BackupStatus
		existingJobs      []batchv1.Job
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      []batchv1.Job
	}{
		{
			name:        "delete job for completed backup is started",
			currentTime: time.Unix(170, 0).UTC(),
			keep:        1,
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: []batchv1.Job{},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-02-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: []batchv1.Job{
				*genDeleteJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Spec.Schedule = "xxx" // must be non-empty
			backupConfig.Spec.Keep = intPtr(tc.keep)
			backupConfig.Status.CurrentBackups = tc.existingBackups

			initObjs := []runtime.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs {
				initObjs = append(initObjs, j.DeepCopy())
			}
			reconciler := Reconciler{
				log:             kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:          ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, initObjs...),
				deleteContainer: genDeleteContainer(),
				recorder:        record.NewFakeRecorder(10),
				clock:           clock,
			}

			reconcileAfter, err := reconciler.startPendingBackupDeleteJobs(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(backupConfig.Status, readbackBackupConfig.Status); diff != nil {
				t.Errorf("backupsConfig status differs from read back one, diff: %v", diff)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			jobs := batchv1.JobList{}
			if err := reconciler.List(context.Background(), &jobs); err != nil {
				t.Fatalf("Error reading created jobs: %v", err)
			}
			if diff := deep.Equal(jobs.Items, tc.expectedJobs); diff != nil {
				t.Errorf("jobs differ from expected ones: %v", diff)
			}

			if deep.Equal(reconcileAfter, tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func TestUpdateRunningBackupDeleteJobs(t *testing.T) {
	testCases := []struct {
		name              string
		currentTime       time.Time
		existingBackups   []kubermaticv1.BackupStatus
		existingJobs      []batchv1.Job
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
	}{
		{
			name:        "deletion is marked as complete if corresponding job has completed",
			currentTime: time.Unix(170, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				// 3 backups with deletions marked as running, a 4th backup is only scheduled
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-03-00",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(210, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(240, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-04-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: []batchv1.Job{
				// first backup's deletion job succeeded, second one's failed, third one's is still running
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-bbbb"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(160, 0).UTC(), "job timed out"),
				*genDeleteJob("testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-cccc"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: 1st backup's deletion marked as completed, 2nd one's restarted, 3rd and 4th unchanged
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(100, 0).UTC()},
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
					DeleteMessage:      "Job failed: job timed out. Restarted.",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-03-00",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(210, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(240, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-04-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
		},
		{
			name:        "if all backup deletions are marked as completed, nothing is changed and we reconcile after the job retention time",
			currentTime: time.Unix(170, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				// 2 backups with deletions marked as running
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			existingJobs: []batchv1.Job{
				// both backup's deletion jobs ended
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-bbbb"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(160, 0).UTC(), "job completed"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: both backups' deletions marked as completed, and we reconcile after the retention period
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(100, 0).UTC()},
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(160, 0).UTC()},
					DeleteMessage:      "job completed",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: succeededJobRetentionTime},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			initObjs := []runtime.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs {
				initObjs = append(initObjs, j.DeepCopy())
			}
			reconciler := Reconciler{
				log:             kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:          ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, initObjs...),
				deleteContainer: genDeleteContainer(),
				recorder:        record.NewFakeRecorder(10),
				clock:           clock,
			}

			reconcileAfter, err := reconciler.updateRunningBackupDeleteJobs(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(backupConfig.Status, readbackBackupConfig.Status); diff != nil {
				t.Errorf("backupsConfig status differs from read back one, diff: %v", diff)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			if deep.Equal(reconcileAfter, tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func TestDeleteFinishedBackupJobs(t *testing.T) {
	testCases := []struct {
		name              string
		currentTime       time.Time
		existingBackups   []kubermaticv1.BackupStatus
		existingJobs      []batchv1.Job
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      []batchv1.Job
	}{
		{
			name: "successfully completed backup jobs are deleted when their retention time runs out",
			existingBackups: []kubermaticv1.BackupStatus{
				// 2 backups with backup jobs marked as completed
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				// 2 backups with backup and delete jobs marked as completed,
				// with deletion finished times the same as the first two backups' backup finished times
				// (just so we can test them with the same current time)
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-dddd",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(140, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-dddd",
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			// current time is such that the 1st and 3rd backup's deletion times are past the retention time but the 2nd and 4th's aren't
			currentTime: time.Unix(145, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				// corresponding backup and delete jobs all completed successfully
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-cccc"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-dddd"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
			},
			// result: 1st and 3rd backup's backup/delete jobs deleted, 3rd backup's status entry also deleted b/c its backup and delete jobs are both deleted
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-dddd",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(140, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-dddd",
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			expectedJobs: []batchv1.Job{
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-dddd"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
			},
			// reconcile when the 2nd & 4th backup's retention times (for the backup and delete job, respectively) run out
			expectedReconcile: &reconcile.Result{RequeueAfter: 5 * time.Second},
		},
		{
			name: "failed backup jobs are deleted when their retention time runs out",
			existingBackups: []kubermaticv1.BackupStatus{
				// 2 backups with backup jobs marked as failed
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			currentTime: time.Unix(145, 0).Add(failedJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				// corresponding jobs have failed
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job failed"),
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// backups unchanged
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(120, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-02-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(150, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			expectedJobs: []batchv1.Job{
				// job that was past the successful job retention time is deleted
				*jobAddCondition(genBackupJob("testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: 5 * time.Second},
		},
		{
			name: "completed delete jobs are deleted, as are the corresponding status entries if the create jobs were deleted already",
			existingBackups: []kubermaticv1.BackupStatus{
				// one backup with deletion marked as completed, one with deletion marked as running, a 3rd backup is only scheduled
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-01-00",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(100, 0).UTC()},
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-03-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(210, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(240, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-04-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			// current time is such that the first backup's deletion time is past the retention time but the others aren't
			currentTime: time.Unix(120, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				// first backup's deletion job succeeded, second one's is still running
				*jobAddCondition(genDeleteJob("testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
				*genDeleteJob("testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-bbbb"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: 1st backup's job and status entry are deleted, other two unchanged
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(180, 0).UTC()},
					BackupName:         "testbackup-1970-01-01t00-03-00",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(210, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(240, 0).UTC()},
					BackupName:    "testbackup-1970-01-01t00-04-00",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedJobs: []batchv1.Job{
				*genDeleteJob("testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-bbbb"),
			},
			// reconcile when the 2nd backup's retention time runs out
			expectedReconcile: &reconcile.Result{RequeueAfter: 90 * time.Second},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			initObjs := []runtime.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs {
				initObjs = append(initObjs, j.DeepCopy())
			}
			reconciler := Reconciler{
				log:             kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:          ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, initObjs...),
				deleteContainer: genDeleteContainer(),
				recorder:        record.NewFakeRecorder(10),
				clock:           clock,
			}

			reconcileAfter, err := reconciler.deleteFinishedBackupJobs(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(backupConfig.Status, readbackBackupConfig.Status); diff != nil {
				t.Errorf("backupsConfig status differs from read back one, diff: %v", diff)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			jobs := batchv1.JobList{}
			if err := reconciler.List(context.Background(), &jobs); err != nil {
				t.Fatalf("Error reading created jobs: %v", err)
			}
			if diff := deep.Equal(jobs.Items, tc.expectedJobs); diff != nil {
				t.Errorf("jobs differ from expected ones: %v", diff)
			}

			if deep.Equal(reconcileAfter, tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func TestFinalization(t *testing.T) {
	testCases := []struct {
		name                       string
		schedule                   string
		currentTime                time.Time
		existingBackups            []kubermaticv1.BackupStatus
		existingJobs               []batchv1.Job
		cleanupContainerDefined    bool
		existingCleanupRunningFlag bool
		expectedBackups            []kubermaticv1.BackupStatus
		expectedReconcile          *reconcile.Result
		expectedJobs               []batchv1.Job
		expectedFinalizer          bool
	}{
		{
			name:     "finalize single completed immediate backup",
			schedule: "",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "oneshot",
					JobName:            "testcluster-backup-oneshot-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-oneshot-delete-xxxx",
				},
			},
			currentTime: time.Unix(90, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				*jobAddCondition(genBackupJob("backup-done-delete-not-started", "testcluster-backup-oneshot-create-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "oneshot",
					JobName:            "testcluster-backup-oneshot-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-oneshot-delete-xxxx",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			expectedJobs: []batchv1.Job{
				// completed backup job deleted, delete job started
				*genDeleteJob("oneshot", "testcluster-backup-oneshot-delete-xxxx"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedFinalizer: true,
		},
		{
			name:     "finalize single deleted immediate backup",
			schedule: "",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "oneshot",
					JobName:            "testcluster-backup-oneshot-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-oneshot-delete-xxxx",
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			currentTime: time.Unix(90, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				*jobAddCondition(genBackupJob("oneshot", "testcluster-backup-oneshot-create-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
				*jobAddCondition(genDeleteJob("oneshot", "testcluster-backup-oneshot-delete-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
			},
			expectedBackups:   nil,
			expectedJobs:      []batchv1.Job{},
			expectedReconcile: &reconcile.Result{},
			expectedFinalizer: false,
		},
		{
			name:     "finalize multiple scheduled backups",
			schedule: "*/20 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "only-scheduled",
					JobName:       "testcluster-backup-only-scheduled-create-xxxx",
					DeleteJobName: "testcluster-backup-only-scheduled-delete-xxxx",
				},
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "backup-running",
					JobName:       "testcluster-backup-backup-running-create-xxxx",
					DeleteJobName: "testcluster-backup-backup-running-delete-xxxx",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "backup-done-delete-not-started",
					JobName:            "testcluster-backup-backup-done-delete-not-started-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-backup-done-delete-not-started-delete-xxxx",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "backup-failed-delete-not-started",
					JobName:            "testcluster-backup-backup-failed-delete-not-started-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-backup-failed-delete-not-started-delete-xxxx",
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "backup-and-delete-done",
					JobName:            "testcluster-backup-and-delete-done-testbackup-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-and-delete-done-testbackup-delete-xxxx",
					DeleteFinishedTime: &metav1.Time{Time: time.Unix(90, 0).UTC()},
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			currentTime: time.Unix(90, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: []batchv1.Job{
				*genBackupJob("backup-running", "testcluster-backup-backup-running-create-xxxx"),
				*jobAddCondition(genBackupJob("backup-done-delete-not-started", "testcluster-backup-backup-done-delete-not-started-create-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
				*jobAddCondition(genBackupJob("backup-failed-delete-not-started", "testcluster-backup-backup-failed-delete-not-started-create-xxxx"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				*jobAddCondition(genBackupJob("backup-and-delete-done", "testcluster-backup-and-delete-done-testbackup-create-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
				*jobAddCondition(genDeleteJob("backup-and-delete-done", "testcluster-backup-and-delete-done-testbackup-delete-xxxx"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job succeeded"),
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:    "backup-running",
					JobName:       "testcluster-backup-backup-running-create-xxxx",
					DeleteJobName: "testcluster-backup-backup-running-delete-xxxx",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "backup-done-delete-not-started",
					JobName:            "testcluster-backup-backup-done-delete-not-started-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-backup-done-delete-not-started-delete-xxxx",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      &metav1.Time{Time: time.Unix(60, 0).UTC()},
					BackupName:         "backup-failed-delete-not-started",
					JobName:            "testcluster-backup-backup-failed-delete-not-started-create-xxxx",
					BackupFinishedTime: &metav1.Time{Time: time.Unix(80, 0).UTC()},
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-backup-failed-delete-not-started-delete-xxxx",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			expectedJobs: []batchv1.Job{
				// all completed & failed jobs deleted, previously non-started delete jobs started
				*genDeleteJob("backup-done-delete-not-started", "testcluster-backup-backup-done-delete-not-started-delete-xxxx"),
				*genDeleteJob("backup-failed-delete-not-started", "testcluster-backup-backup-failed-delete-not-started-delete-xxxx"),
				*genBackupJob("backup-running", "testcluster-backup-backup-running-create-xxxx"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedFinalizer: true,
		},
		{
			name:                    "cleanup job started if container defined and no remaining backups",
			schedule:                "",
			cleanupContainerDefined: true,
			existingBackups:         nil,
			currentTime:             time.Unix(60, 0).UTC(),
			existingJobs:            []batchv1.Job{},
			expectedBackups:         nil,
			expectedJobs: []batchv1.Job{
				*genCleanupJob("testcluster-backup-testbackup-cleanup"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: 30 * time.Second},
			expectedFinalizer: true,
		},
		{
			name:                       "running cleanup job kept running",
			schedule:                   "",
			cleanupContainerDefined:    true,
			existingBackups:            nil,
			existingCleanupRunningFlag: true,
			currentTime:                time.Unix(60, 0).UTC(),
			existingJobs: []batchv1.Job{
				*genCleanupJob("testcluster-backup-testbackup-cleanup"),
			},
			expectedBackups: nil,
			expectedJobs: []batchv1.Job{
				*genCleanupJob("testcluster-backup-testbackup-cleanup"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: 30 * time.Second},
			expectedFinalizer: true,
		},
		{
			name:                       "failed cleanup job restarted",
			schedule:                   "",
			cleanupContainerDefined:    true,
			existingBackups:            nil,
			existingCleanupRunningFlag: true,
			currentTime:                time.Unix(60, 0).UTC(),
			existingJobs: []batchv1.Job{
				*jobAddCondition(genCleanupJob("testcluster-backup-testbackup-cleanup"),
					batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "cleanup job failed"),
			},
			expectedBackups: nil,
			expectedJobs: []batchv1.Job{
				*genCleanupJob("testcluster-backup-testbackup-cleanup"),
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: 30 * time.Second},
			expectedFinalizer: true,
		},
		{
			name:                       "succeeded cleanup job deleted, finalizer removed",
			schedule:                   "",
			cleanupContainerDefined:    true,
			existingBackups:            nil,
			existingCleanupRunningFlag: true,
			currentTime:                time.Unix(60, 0).UTC(),
			existingJobs: []batchv1.Job{
				*jobAddCondition(genCleanupJob("testcluster-backup-testbackup-cleanup"),
					batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "cleanup job completed"),
			},
			expectedBackups:   nil,
			expectedJobs:      []batchv1.Job{},
			expectedReconcile: &reconcile.Result{},
			expectedFinalizer: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clock.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.SetDeletionTimestamp(&metav1.Time{Time: clock.Now()})
			backupConfig.Spec.Keep = intPtr(100)
			backupConfig.Status.CurrentBackups = tc.existingBackups
			backupConfig.Status.CleanupRunning = tc.existingCleanupRunningFlag
			kuberneteshelper.AddFinalizer(backupConfig, DeleteAllBackupsFinalizer)

			initObjs := []runtime.Object{
				cluster,
				backupConfig,
				genClusterRootCaSecret(),
			}
			for _, j := range tc.existingJobs {
				initObjs = append(initObjs, j.DeepCopy())
			}
			reconciler := Reconciler{
				log:             kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:          ctrlruntimefakeclient.NewFakeClientWithScheme(scheme.Scheme, initObjs...),
				storeContainer:  genStoreContainer(),
				deleteContainer: genDeleteContainer(),
				recorder:        record.NewFakeRecorder(10),
				clock:           clock,
			}
			if tc.cleanupContainerDefined {
				reconciler.cleanupContainer = genCleanupContainer()
			}

			reconcileAfter, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: backupConfig.Namespace, Name: backupConfig.Name}})
			if err != nil {
				t.Fatalf("ensureNextBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), client.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if diff := deep.Equal(readbackBackupConfig.Status.CurrentBackups, tc.expectedBackups); diff != nil {
				t.Errorf("backups differ from expected, diff: %v", diff)
			}

			jobs := batchv1.JobList{}
			if err := reconciler.List(context.Background(), &jobs); err != nil {
				t.Fatalf("Error reading created jobs: %v", err)
			}
			if diff := deep.Equal(jobs.Items, tc.expectedJobs); diff != nil {
				t.Errorf("jobs differ from expected ones: %v", diff)
			}

			if tc.expectedFinalizer != kuberneteshelper.HasFinalizer(readbackBackupConfig, DeleteAllBackupsFinalizer) {
				t.Errorf("finalizer presence: expected %v, was %v", tc.expectedFinalizer, !tc.expectedFinalizer)
			}

			if deep.Equal(reconcileAfter, *tc.expectedReconcile) != nil {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func constRandStringGenerator(str string) func() string {
	return func() string {
		return str
	}
}
