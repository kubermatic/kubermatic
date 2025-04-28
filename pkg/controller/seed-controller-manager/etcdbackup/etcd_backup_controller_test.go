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
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	etcdbackup "k8c.io/kubermatic/v2/pkg/resources/etcd/backup"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	clocktesting "k8s.io/utils/clock/testing"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type jobFunc func(data *resources.TemplateData) []batchv1.Job

func genTestCluster() *kubermaticv1.Cluster {
	version := *semver.NewSemverOrDie("1.16.3")

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: version,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "testnamespace",
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver: kubermaticv1.HealthStatusUp,
			},
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane: version,
				Apiserver:    version,
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
	}
}

func getConfigGetter(t *testing.T) provider.KubermaticConfigurationGetter {
	config := &kubermaticv1.KubermaticConfiguration{}

	configGetter, err := kubernetesprovider.StaticKubermaticConfigurationGetterFactory(config)
	if err != nil {
		t.Fatalf("failed to create config getter: %v", err)
	}

	return configGetter
}

func genBackupJob(data *resources.TemplateData, backupName, jobName string) *batchv1.Job {
	// jerry-rig a cluster, BackupConfig and BackupStatus instance to create a job object
	// that's similar to the ones an actual reconciliation will create
	cluster := genTestCluster()
	backupConfig := genBackupConfig(cluster, "testbackup")
	backup := &kubermaticv1.BackupStatus{
		BackupName: backupName,
		JobName:    jobName,
	}

	job := etcdbackup.BackupJob(data, backupConfig, backup)
	job.ResourceVersion = "1"
	// remove all env variables from the job so they're comparable against the
	// ones we get from fake clusters during tests, where we strip the variables too
	job.Spec.Template.Spec.Containers[0].Env = nil
	return job
}

func genBackupDeleteJob(data *resources.TemplateData, backupName, jobName string) *batchv1.Job {
	// same thing as genBackupJob, but for delete jobs
	cluster := genTestCluster()
	backupConfig := genBackupConfig(cluster, "testbackup")
	backup := &kubermaticv1.BackupStatus{
		BackupName:    backupName,
		DeleteJobName: jobName,
	}

	job := etcdbackup.BackupDeleteJob(data, backupConfig, backup)
	job.ResourceVersion = "1"
	// remove all env variables from the job so they're comparable against the
	// ones we get from fake clusters during tests, where we strip the variables too
	job.Spec.Template.Spec.Containers[0].Env = nil
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

func genBackupStatusList(n int, gen func(i int) kubermaticv1.BackupStatus) []kubermaticv1.BackupStatus {
	var result []kubermaticv1.BackupStatus
	for i := range n {
		result = append(result, gen(i))
	}
	return result
}

func genJobList(n int, gen func(i int) batchv1.Job) []batchv1.Job {
	var result []batchv1.Job
	for i := range n {
		result = append(result, gen(i))
	}
	return result
}

func TestEnsurePendingBackupIsScheduled(t *testing.T) {
	testCases := []struct {
		name              string
		creationTime      time.Time
		currentTime       time.Time
		schedule          string
		backupConfigName  string
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
					ScheduledTime: metav1.NewTime(time.Unix(10, 0).UTC()),
					BackupName:    "testbackup.db.gz",
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
					ScheduledTime: metav1.NewTime(time.Unix(100, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-40.db",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(100, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-40.db",
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
			name:            "before the next scheduled time slot, no backup is scheduled and we reconcile at that time slot",
			creationTime:    time.Unix(60, 0).UTC(),
			currentTime:     time.Unix(120, 0).UTC(),
			schedule:        "*/10 * * * *",
			existingBackups: nil,
			expectedBackups: nil,
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 480 * time.Second, // diff. between current time and next 10min slot
			},
		},
		{
			name:            "with no backups, schedules with @every descriptors reconcile based on on the config creation time",
			creationTime:    time.Unix(60, 0).UTC(),
			currentTime:     time.Unix(120, 0).UTC(),
			schedule:        "@every 10m",
			existingBackups: nil,
			expectedBackups: nil,
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 540 * time.Second, // diff. between current time and backup config creation time + 10m
			},
		},
		{
			name:         "with most recent backup scheduled, no new backup is scheduled and we reconcile at the next time slot",
			creationTime: time.Unix(60, 0).UTC(),
			currentTime:  time.Unix(660, 0).UTC(),
			schedule:     "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(600, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-10-00.db",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(600, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-10-00.db",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 540 * time.Second,
			},
		},
		{
			name:         "with most recent backup NOT scheduled, that backup is scheduled and we reconcile at the next time slot",
			creationTime: time.Unix(60, 0).UTC(),
			currentTime:  time.Unix(1260, 0).UTC(), //
			schedule:     "*/10 * * * *",
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(600, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-10-00.db",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(600, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-10-00.db",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(1200, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-20-00.db.gz",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 540 * time.Second,
			},
		},
		{
			name:            "with multiple past backups missing, still only the most recent one is scheduled",
			creationTime:    time.Unix(0, 0).UTC(),
			currentTime:     time.Unix(3600*24*17, 0).UTC(),
			schedule:        "@every 120h",
			existingBackups: nil,
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(3600*24*15, 0).UTC()),
					BackupName:    "testbackup-1970-01-16t00-00-00.db.gz",
					JobName:       "testcluster-backup-testbackup-create-xxxx",
					DeleteJobName: "testcluster-backup-testbackup-delete-xxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 3600 * 24 * 3 * time.Second,
			},
		},
		{
			name:             "If the backup config name is long, job names are cut short properly",
			currentTime:      time.Unix(10, 0).UTC(),
			backupConfigName: "long-backup-config-name-abcdefghijk",
			schedule:         "",
			existingBackups:  []kubermaticv1.BackupStatus{},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(10, 0).UTC()),
					BackupName:    "long-backup-config-name-abcdefghijk.db.gz",
					JobName:       "testcluster-backup-long-backup-config-name-abcdefghijk-creaxxxx",
					DeleteJobName: "testcluster-backup-long-backup-config-name-abcdefghijk-delexxxx",
				},
			},
			expectedReconcile: &reconcile.Result{
				Requeue:      true,
				RequeueAfter: 0,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clocktesting.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Spec.Schedule = tc.schedule
			backupConfig.SetCreationTimestamp(metav1.Time{Time: tc.creationTime})
			backupConfig.Status.CurrentBackups = tc.existingBackups
			if tc.backupConfigName != "" {
				backupConfig.Name = tc.backupConfigName
			}

			reconciler := Reconciler{
				log:                 kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:              fake.NewClientBuilder().WithObjects(cluster, backupConfig).Build(),
				scheme:              scheme.Scheme,
				recorder:            record.NewFakeRecorder(10),
				clock:               clock,
				randStringGenerator: constRandStringGenerator("xxxx"),
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(), nil
				},
			}

			reconcileAfter, err := reconciler.ensurePendingBackupIsScheduled(context.Background(), backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensurePendingBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if d := diff.ObjectDiff(backupConfig.Status, readbackBackupConfig.Status); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedBackups, readbackBackupConfig.Status.CurrentBackups); d != "" {
				t.Errorf("backups differ from expected:\n%v", d)
			}

			if !diff.SemanticallyEqual(reconcileAfter, tc.expectedReconcile) {
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
		existingJobs      jobFunc
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      jobFunc
	}{
		{
			name:        "backup job scheduled in the past it started, job scheduled in the future is not",
			currentTime: time.Unix(90, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-00.db",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-02-00.db",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-00.db",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-02-00.db",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
				}
			},
		},

		{
			name:        "finished backup job is marked as finished in the backup status",
			currentTime: time.Unix(90, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-00.db",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(70, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-10.db",
					JobName:       "testcluster-backup-testbackup-create-bbbb",
					DeleteJobName: "testcluster-backup-testbackup-delete-bbbb",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-02-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-10", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(80, 0).UTC(), "Job has reached the specified backoff limit"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(70, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-10.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(80, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "Job has reached the specified backoff limit",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-02-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: nil,
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-10", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(80, 0).UTC(), "Job has reached the specified backoff limit"),
				}
			},
		},
		{
			name:        "still-running backup job is not changed, reconcile after assumed job runtime",
			currentTime: time.Unix(90, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-00.db",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime: metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-01-00.db",
					JobName:       "testcluster-backup-testbackup-create-aaaa",
					DeleteJobName: "testcluster-backup-testbackup-delete-aaaa",
					BackupPhase:   kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clocktesting.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithCluster(cluster).
				WithVersions(kubermatic.NewFakeVersions()).
				WithEtcdLauncherImage(defaulting.DefaultEtcdLauncherImage).
				WithEtcdBackupStoreContainer(genStoreContainer(), false).
				WithEtcdBackupDeleteContainer(genDeleteContainer(), false).
				WithEtcdBackupDestination(genDefaultBackupDestination()).
				Build()

			initObjs := []ctrlruntimeclient.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs(td) {
				initObjs = append(initObjs, j.DeepCopy())
			}

			fc := fake.NewClientBuilder().WithObjects(initObjs...).Build()

			reconciler := Reconciler{
				log:      kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:   fc,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),
				clock:    clock,
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(), nil
				},
				configGetter: getConfigGetter(t),
			}

			reconcileAfter, err := reconciler.startPendingBackupJobs(ctx, td, backupConfig)
			if err != nil {
				t.Fatalf("ensurePendingBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if d := diff.ObjectDiff(backupConfig.Status, readbackBackupConfig.Status); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedBackups, readbackBackupConfig.Status.CurrentBackups); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedJobs(td), getSortedJobs(t, reconciler)); d != "" {
				t.Errorf("jobs differ from expected ones:\n%v", d)
			}

			if !diff.SemanticallyEqual(reconcileAfter, tc.expectedReconcile) {
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
		existingJobs      jobFunc
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      jobFunc
	}{
		{
			name:        "delete job for completed backup is started",
			currentTime: time.Unix(170, 0).UTC(),
			keep:        1,
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-03-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-03-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
				}
			},
		},
		{
			name:        "failed jobs are deleted immediately",
			currentTime: time.Unix(170, 0).UTC(),
			keep:        10,
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
				}
			},
		},
		{
			name:        "already-running delete jobs counted into keep count",
			currentTime: time.Unix(170, 0).UTC(),
			keep:        1,
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-bbbb"),
				}
			},
		},
		{
			name:        "already-finished deletion is not restarted",
			currentTime: time.Unix(240, 0).UTC(),
			keep:        0,
			existingBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeleteFinishedTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "delete job completed",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// unchanged
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeleteFinishedTime: metav1.NewTime(time.Unix(120, 0).UTC()),
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "delete job completed",
				},
			},
			expectedReconcile: nil,
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
		},
		{
			name:        "not more than maxSimultaneousDeleteJobsPerConfig delete jobs are started",
			currentTime: time.Unix(400, 0).UTC(),
			keep:        1,
			existingBackups: genBackupStatusList(maxSimultaneousDeleteJobsPerConfig+2, func(i int) kubermaticv1.BackupStatus {
				return kubermaticv1.BackupStatus{
					ScheduledTime:      metav1.NewTime(time.Unix(60+int64(i)*60, 0).UTC()),
					BackupName:         fmt.Sprintf("testbackup-%v.db", i),
					JobName:            fmt.Sprintf("testcluster-backup-testbackup-%v-create", i),
					BackupFinishedTime: metav1.NewTime(time.Unix(90+int64(i)*60, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      fmt.Sprintf("testcluster-backup-testbackup-%v-delete", i),
				}
			}),
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{}
			},
			expectedBackups: genBackupStatusList(maxSimultaneousDeleteJobsPerConfig+2, func(i int) kubermaticv1.BackupStatus {
				result := kubermaticv1.BackupStatus{
					ScheduledTime:      metav1.NewTime(time.Unix(60+int64(i)*60, 0).UTC()),
					BackupName:         fmt.Sprintf("testbackup-%v.db", i),
					JobName:            fmt.Sprintf("testcluster-backup-testbackup-%v-create", i),
					BackupFinishedTime: metav1.NewTime(time.Unix(90+int64(i)*60, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      fmt.Sprintf("testcluster-backup-testbackup-%v-delete", i),
				}
				if i > 0 && i <= maxSimultaneousDeleteJobsPerConfig {
					result.DeletePhase = kubermaticv1.BackupStatusPhaseRunning
				}
				return result
			}),
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return genJobList(maxSimultaneousDeleteJobsPerConfig, func(i int) batchv1.Job {
					return *genBackupDeleteJob(data, fmt.Sprintf("testbackup-%v", i+1), fmt.Sprintf("testcluster-backup-testbackup-%v-delete", i+1))
				})
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clocktesting.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Spec.Schedule = "xxx" // must be non-empty
			backupConfig.Spec.Keep = intPtr(tc.keep)
			backupConfig.Status.CurrentBackups = tc.existingBackups

			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithCluster(cluster).
				WithVersions(kubermatic.NewFakeVersions()).
				WithEtcdLauncherImage(defaulting.DefaultEtcdLauncherImage).
				WithEtcdBackupStoreContainer(genStoreContainer(), false).
				WithEtcdBackupDeleteContainer(genDeleteContainer(), false).
				WithEtcdBackupDestination(genDefaultBackupDestination()).
				Build()

			initObjs := []ctrlruntimeclient.Object{
				cluster,
				backupConfig,
			}

			for _, j := range tc.existingJobs(td) {
				initObjs = append(initObjs, j.DeepCopy())
			}

			fc := fake.NewClientBuilder().WithObjects(initObjs...).Build()

			reconciler := Reconciler{
				log:      kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:   fc,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),
				clock:    clock,
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(), nil
				},
				configGetter: getConfigGetter(t),
			}

			reconcileAfter, err := reconciler.startPendingBackupDeleteJobs(ctx, td, backupConfig)
			if err != nil {
				t.Fatalf("ensurePendingBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if d := diff.ObjectDiff(backupConfig.Status, readbackBackupConfig.Status); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedBackups, readbackBackupConfig.Status.CurrentBackups); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedJobs(td), getSortedJobs(t, reconciler)); d != "" {
				t.Errorf("jobs differ from expected ones:\n%v", d)
			}

			if !diff.SemanticallyEqual(reconcileAfter, tc.expectedReconcile) {
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
		existingJobs      jobFunc
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
	}{
		{
			name:        "deletion is marked as complete if corresponding job has completed",
			currentTime: time.Unix(170, 0).UTC(),
			existingBackups: []kubermaticv1.BackupStatus{
				// 3 backups with deletions marked as running, a 4th backup is only scheduled
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(240, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-04-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// first backup's deletion job succeeded, second one's failed, third one's is still running
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-bbbb"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(160, 0).UTC(), "job timed out"),
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-cccc"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: 1st backup's deletion marked as completed, 2nd one's restarted, 3rd and 4th unchanged
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: metav1.NewTime(time.Unix(100, 0).UTC()),
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
					DeleteMessage:      "Job failed: job timed out. Restarted.",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(240, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-04-00.db",
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
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
			},
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// both backup's deletion jobs ended
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-bbbb"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(160, 0).UTC(), "job completed"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: both backups' deletions marked as completed, and we reconcile after the retention period
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: metav1.NewTime(time.Unix(100, 0).UTC()),
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: metav1.NewTime(time.Unix(160, 0).UTC()),
					DeleteMessage:      "job completed",
				},
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: succeededJobRetentionTime},
		},
		{
			name:        "not more than maxSimultaneousDeleteJobsPerConfig delete jobs are restarted",
			currentTime: time.Unix(170, 0).UTC(),
			existingBackups: genBackupStatusList(maxSimultaneousDeleteJobsPerConfig+2, func(i int) kubermaticv1.BackupStatus {
				return kubermaticv1.BackupStatus{
					ScheduledTime:      metav1.NewTime(time.Unix(60+int64(i)*60, 0).UTC()),
					BackupName:         fmt.Sprintf("testbackup-%v.db", i),
					JobName:            fmt.Sprintf("testcluster-backup-%v-create", i),
					BackupFinishedTime: metav1.NewTime(time.Unix(150+int64(i)*60, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      fmt.Sprintf("testcluster-backup-%v-delete", i),
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				}
			}),
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return genJobList(maxSimultaneousDeleteJobsPerConfig+1, func(i int) batchv1.Job {
					// 0th job is missing, the others have failed
					i++
					return *jobAddCondition(genBackupDeleteJob(data, fmt.Sprintf("testbackup-%v", i), fmt.Sprintf("testcluster-backup-%v-delete", i)),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(100+int64(i)*60, 0).UTC(), "job timed out")
				})
			},
			expectedBackups: genBackupStatusList(maxSimultaneousDeleteJobsPerConfig+2, func(i int) kubermaticv1.BackupStatus {
				result := kubermaticv1.BackupStatus{
					ScheduledTime:      metav1.NewTime(time.Unix(60+int64(i)*60, 0).UTC()),
					BackupName:         fmt.Sprintf("testbackup-%v.db", i),
					JobName:            fmt.Sprintf("testcluster-backup-%v-create", i),
					BackupFinishedTime: metav1.NewTime(time.Unix(150+int64(i)*60, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      fmt.Sprintf("testcluster-backup-%v-delete", i),
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				}
				if i == 0 {
					result.DeleteMessage = "job was deleted, restarted it"
				} else if i < maxSimultaneousDeleteJobsPerConfig {
					result.DeleteMessage = "Job failed: job timed out. Restarted."
				}
				return result
			}),
			expectedReconcile: &reconcile.Result{RequeueAfter: assumedJobRuntime},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clocktesting.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithCluster(cluster).
				WithVersions(kubermatic.NewFakeVersions()).
				WithEtcdLauncherImage(defaulting.DefaultEtcdLauncherImage).
				WithEtcdBackupStoreContainer(genStoreContainer(), false).
				WithEtcdBackupDeleteContainer(genDeleteContainer(), false).
				WithEtcdBackupDestination(genDefaultBackupDestination()).
				Build()

			initObjs := []ctrlruntimeclient.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs(td) {
				initObjs = append(initObjs, j.DeepCopy())
			}

			fc := fake.NewClientBuilder().WithObjects(initObjs...).Build()

			reconciler := Reconciler{
				log:      kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:   fc,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),
				clock:    clock,
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(), nil
				},
				configGetter: getConfigGetter(t),
			}

			reconcileAfter, err := reconciler.updateRunningBackupDeleteJobs(ctx, td, backupConfig)
			if err != nil {
				t.Fatalf("ensurePendingBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if d := diff.ObjectDiff(backupConfig.Status, readbackBackupConfig.Status); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedBackups, readbackBackupConfig.Status.CurrentBackups); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if !diff.SemanticallyEqual(reconcileAfter, tc.expectedReconcile) {
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
		existingJobs      jobFunc
		expectedBackups   []kubermaticv1.BackupStatus
		expectedReconcile *reconcile.Result
		expectedJobs      jobFunc
	}{
		{
			name: "successfully completed backup jobs are deleted when their retention time runs out",
			existingBackups: []kubermaticv1.BackupStatus{
				// 2 backups with backup jobs marked as completed
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				// 2 backups with backup and delete jobs marked as completed,
				// with deletion finished times the same as the first two backups' backup finished times
				// (just so we can test them with the same current time)
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-cccc",
					BackupFinishedTime: metav1.NewTime(time.Unix(80, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-cccc",
					DeleteFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-dddd",
					BackupFinishedTime: metav1.NewTime(time.Unix(140, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-dddd",
					DeleteFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			// current time is such that the 1st and 3rd backup's deletion times are past the retention time but the 2nd and 4th's aren't
			currentTime: time.Unix(145, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// corresponding backup and delete jobs all completed successfully
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-cccc"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job completed"),
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-dddd"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				}
			},
			// result: 1st and 3rd backup's backup/delete jobs deleted, 3rd backup's status entry also deleted b/c its backup and delete jobs are both deleted
			expectedBackups: []kubermaticv1.BackupStatus{
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-dddd",
					BackupFinishedTime: metav1.NewTime(time.Unix(140, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-dddd",
					DeleteFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteMessage:      "job complete",
				},
			},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-delete-dddd"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				}
			},
			// reconcile when the 2nd & 4th backup's retention times (for the backup and delete job, respectively) run out
			expectedReconcile: &reconcile.Result{RequeueAfter: 5 * time.Second},
		},
		{
			name: "failed backup jobs are deleted when their retention time runs out",
			existingBackups: []kubermaticv1.BackupStatus{
				// 2 backups with backup jobs marked as failed
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			currentTime: time.Unix(145, 0).Add(failedJobRetentionTime).UTC(),
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// corresponding jobs have failed
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-create-aaaa"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(90, 0).UTC(), "job failed"),
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// backups unchanged
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(120, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-02-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(150, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseFailed,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
				},
			},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// job that was past the successful job retention time is deleted
					*jobAddCondition(genBackupJob(data, "testbackup-1970-01-01t00-02-00", "testcluster-backup-testbackup-create-bbbb"),
						batchv1.JobFailed, corev1.ConditionTrue, time.Unix(150, 0).UTC(), "job failed"),
				}
			},
			expectedReconcile: &reconcile.Result{RequeueAfter: 5 * time.Second},
		},
		{
			name: "completed delete jobs are deleted, as are the corresponding status entries if the create jobs were deleted already",
			existingBackups: []kubermaticv1.BackupStatus{
				// one backup with deletion marked as completed, one with deletion marked as running, a 3rd backup is only scheduled
				{
					ScheduledTime:      metav1.NewTime(time.Unix(60, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-01-00.db",
					JobName:            "testcluster-backup-testbackup-create-aaaa",
					BackupFinishedTime: metav1.NewTime(time.Unix(90, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-aaaa",
					DeletePhase:        kubermaticv1.BackupStatusPhaseCompleted,
					DeleteFinishedTime: metav1.NewTime(time.Unix(100, 0).UTC()),
					DeleteMessage:      "job completed",
				},
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(240, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-04-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			// current time is such that the first backup's deletion time is past the retention time but the others aren't
			currentTime: time.Unix(120, 0).Add(succeededJobRetentionTime).UTC(),
			existingJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					// first backup's deletion job succeeded, second one's is still running
					*jobAddCondition(genBackupDeleteJob(data, "testbackup-1970-01-01t00-01-00", "testcluster-backup-testbackup-delete-aaaa"),
						batchv1.JobComplete, corev1.ConditionTrue, time.Unix(100, 0).UTC(), "job completed"),
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-bbbb"),
				}
			},
			expectedBackups: []kubermaticv1.BackupStatus{
				// result: 1st backup's job and status entry are deleted, other two unchanged
				{
					ScheduledTime:      metav1.NewTime(time.Unix(180, 0).UTC()),
					BackupName:         "testbackup-1970-01-01t00-03-00.db",
					JobName:            "testcluster-backup-testbackup-create-bbbb",
					BackupFinishedTime: metav1.NewTime(time.Unix(210, 0).UTC()),
					BackupPhase:        kubermaticv1.BackupStatusPhaseCompleted,
					BackupMessage:      "job completed",
					DeleteJobName:      "testcluster-backup-testbackup-delete-bbbb",
					DeletePhase:        kubermaticv1.BackupStatusPhaseRunning,
				},
				{
					ScheduledTime: metav1.NewTime(time.Unix(240, 0).UTC()),
					BackupName:    "testbackup-1970-01-01t00-04-00.db",
					JobName:       "testcluster-backup-testbackup-create-cccc",
					DeleteJobName: "testcluster-backup-testbackup-delete-cccc",
				},
			},
			expectedJobs: func(data *resources.TemplateData) []batchv1.Job {
				return []batchv1.Job{
					*genBackupDeleteJob(data, "testbackup-1970-01-01t00-03-00", "testcluster-backup-testbackup-delete-bbbb"),
				}
			},
			// reconcile when the 2nd backup's retention time runs out
			expectedReconcile: &reconcile.Result{RequeueAfter: 90 * time.Second},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			cluster := genTestCluster()
			backupConfig := genBackupConfig(cluster, "testbackup")

			clock := clocktesting.NewFakeClock(tc.currentTime.UTC())
			backupConfig.SetCreationTimestamp(metav1.Time{Time: clock.Now()})
			backupConfig.Status.CurrentBackups = tc.existingBackups

			td := resources.NewTemplateDataBuilder().
				WithContext(ctx).
				WithCluster(cluster).
				WithVersions(kubermatic.NewFakeVersions()).
				WithEtcdLauncherImage(defaulting.DefaultEtcdLauncherImage).
				WithEtcdBackupStoreContainer(genStoreContainer(), false).
				WithEtcdBackupDeleteContainer(genDeleteContainer(), false).
				WithEtcdBackupDestination(genDefaultBackupDestination()).
				Build()

			initObjs := []ctrlruntimeclient.Object{
				cluster,
				backupConfig,
			}
			for _, j := range tc.existingJobs(td) {
				initObjs = append(initObjs, j.DeepCopy())
			}

			reconciler := Reconciler{
				log:      kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:   fake.NewClientBuilder().WithObjects(initObjs...).Build(),
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),
				clock:    clock,
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(), nil
				},
				configGetter: getConfigGetter(t),
			}

			reconcileAfter, err := reconciler.deleteFinishedBackupJobs(ctx, reconciler.log, backupConfig, cluster)
			if err != nil {
				t.Fatalf("ensurePendingBackupIsScheduled returned an error: %v", err)
			}

			readbackBackupConfig := &kubermaticv1.EtcdBackupConfig{}
			if err := reconciler.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: backupConfig.GetNamespace(), Name: backupConfig.GetName()}, readbackBackupConfig); err != nil {
				t.Fatalf("Error reading back completed backupConfig: %v", err)
			}

			if d := diff.ObjectDiff(backupConfig.Status, readbackBackupConfig.Status); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedBackups, readbackBackupConfig.Status.CurrentBackups); d != "" {
				t.Errorf("backupsConfig status differs from read back one:\n%v", d)
			}

			if d := diff.ObjectDiff(tc.expectedJobs(td), getSortedJobs(t, reconciler)); d != "" {
				t.Errorf("jobs differ from expected ones:\n%v", d)
			}

			if !diff.SemanticallyEqual(reconcileAfter, tc.expectedReconcile) {
				t.Errorf("reconcile time differs from expected, expected: %v, actual: %v", tc.expectedReconcile, reconcileAfter)
			}
		})
	}
}

func getSortedJobs(t *testing.T, reconciler Reconciler) []batchv1.Job {
	jobList := batchv1.JobList{}
	if err := reconciler.List(context.Background(), &jobList); err != nil {
		t.Fatalf("Error reading created jobList: %v", err)
	}
	jobs := jobList.DeepCopy().Items
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Name < jobs[j].Name
	})
	// remove all env variables from the jobs so they're comparable against the
	// fake ones we generate with the gen*Job functions above
	for i := range jobs {
		jobs[i].Spec.Template.Spec.Containers[0].Env = nil
	}

	if jobs == nil {
		jobs = []batchv1.Job{}
	}

	return jobs
}

func intPtr(i int) *int {
	return &i
}

func constRandStringGenerator(str string) func() string {
	return func() string {
		return str
	}
}

func TestMultipleBackupDestination(t *testing.T) {
	testCases := []struct {
		name               string
		backupConfig       *kubermaticv1.EtcdBackupConfig
		expectedReconcile  *reconcile.Result
		expectedJobEnvVars []corev1.EnvVar
		expectedErr        string
	}{
		{
			name: "test reconcile with specified backup destination",
			backupConfig: func() *kubermaticv1.EtcdBackupConfig {
				c := genBackupConfig(genTestCluster(), "testbackup")
				c.Spec.Destination = "s3"
				return c
			}(),
			expectedJobEnvVars: []corev1.EnvVar{
				etcdbackup.GenSecretEnvVar(etcdbackup.AccessKeyIdEnvVarKey, etcdbackup.AccessKeyIdEnvVarKey, genDefaultBackupDestination()),
				etcdbackup.GenSecretEnvVar(etcdbackup.SecretAccessKeyEnvVarKey, etcdbackup.SecretAccessKeyEnvVarKey, genDefaultBackupDestination()),
				{
					Name:  etcdbackup.BucketNameEnvVarKey,
					Value: genDefaultBackupDestination().BucketName,
				},
				{
					Name:  etcdbackup.BackupEndpointEnvVarKey,
					Value: genDefaultBackupDestination().Endpoint,
				},
			},
		},
		{
			name: "backup should fail if destination has no credentials set",
			backupConfig: func() *kubermaticv1.EtcdBackupConfig {
				c := genBackupConfig(genTestCluster(), "testbackup")
				c.Spec.Destination = "no-credentials"
				return c
			}(),
			expectedJobEnvVars: []corev1.EnvVar{},
			expectedErr:        fmt.Sprintf("failed to get template data: credentials not set for backup destination %q", "no-credentials"),
		},
		{
			name: "backup should fail destination is missing",
			backupConfig: func() *kubermaticv1.EtcdBackupConfig {
				c := genBackupConfig(genTestCluster(), "testbackup")
				c.Spec.Destination = "missing"
				return c
			}(),
			expectedJobEnvVars: []corev1.EnvVar{},
			expectedErr:        fmt.Sprintf("failed to get template data: cannot find backup destination %q", "missing"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initObjs := []ctrlruntimeclient.Object{
				genTestCluster(),
				tc.backupConfig,
				genClusterRootCaSecret(),
			}

			reconciler := Reconciler{
				log:      kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:   fake.NewClientBuilder().WithObjects(initObjs...).Build(),
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(10),
				clock:    clocktesting.NewFakeClock(time.Unix(60, 0).UTC()),
				caBundle: certificates.NewFakeCABundle(),
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return generator.GenTestSeed(addSeedDestinations), nil
				},
				randStringGenerator: constRandStringGenerator("bob"),
				configGetter:        getConfigGetter(t),

				etcdLauncherImage: defaulting.DefaultEtcdLauncherImage,
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: tc.backupConfig.Namespace, Name: tc.backupConfig.Name}})
			if err != nil {
				if tc.expectedErr != "" {
					if err.Error() != tc.expectedErr {
						t.Errorf("error differs from expected ones: expected %q, got %q", tc.expectedErr, err.Error())
					}
					return
				}
				t.Fatal(err)
			}

			jobList := batchv1.JobList{}
			if err := reconciler.List(context.Background(), &jobList); err != nil {
				t.Fatalf("Error reading created joblist: %v", err)
			}

			if len(jobList.Items) != 1 {
				t.Fatalf("expected 1 job, got %d", len(jobList.Items))
			}

			job := jobList.Items[0]
			envVars := job.Spec.Template.Spec.Containers[0].Env

			for _, expectedEnvVar := range tc.expectedJobEnvVars {
				if !containsEnvVar(envVars, expectedEnvVar) {
					t.Fatalf("expected job env vars %v to contain %v", envVars, expectedEnvVar)
				}
			}
		})
	}
}

func addSeedDestinations(seed *kubermaticv1.Seed) {
	seed.Spec.EtcdBackupRestore = &kubermaticv1.EtcdBackupRestore{
		DefaultDestination: "s3",
		Destinations: map[string]*kubermaticv1.BackupDestination{
			"s3": genDefaultBackupDestination(),
			"no-credentials": {
				BucketName: "no-cred",
				Endpoint:   "no-cred.com",
			},
		},
	}
}

func genDefaultBackupDestination() *kubermaticv1.BackupDestination {
	return &kubermaticv1.BackupDestination{
		Endpoint:   "aws.s3.com",
		BucketName: "s3",
		Credentials: &corev1.SecretReference{
			Name:      "credentials-s3",
			Namespace: metav1.NamespaceSystem,
		},
	}
}

func containsEnvVar(envVars []corev1.EnvVar, envVar corev1.EnvVar) bool {
	for _, e := range envVars {
		if len(deep.Equal(e, envVar)) == 0 {
			return true
		}
	}
	return false
}

func TestIsInsecure(t *testing.T) {
	testcases := []struct {
		url      string
		insecure bool
	}{
		{url: "foo.com", insecure: false},
		{url: "foo.com:443", insecure: false},
		{url: "https", insecure: false},
		{url: "https:433", insecure: false},
		{url: "http://foo.com", insecure: true},
		{url: "hTtP://foo.com", insecure: true},
		{url: "https://foo.com", insecure: false},
		{url: "HtTpS://foo.com", insecure: false},
		{url: "HtTpS://foo.com:80", insecure: false},
	}

	for _, testcase := range testcases {
		t.Run(testcase.url, func(t *testing.T) {
			if isInsecureURL(testcase.url) != testcase.insecure {
				t.Fatalf("Expected insecure=%v, but got the opposite.", testcase.insecure)
			}
		})
	}
}

func isInsecureURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	// a hostname like "foo.com:9000" is parsed as {scheme: "foo.com", host: ""},
	// so we must make sure to not mis-interpret "http:9000" ({scheme: "http", host: ""}) as
	// an HTTP url

	return strings.ToLower(parsed.Scheme) == "http" && parsed.Host != ""
}
