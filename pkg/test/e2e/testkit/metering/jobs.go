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

package metering

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"k8c.io/kubermatic/v2/pkg/util/wait"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// jobTimeout covers the image pull and a report run over all clusters of
	// the seed.
	jobTimeout = 10 * time.Minute

	// testJobLabel marks the Jobs the checks start, so they can be found and
	// cleaned up.
	testJobLabel = "metering-e2e"
)

// runReportJob starts a Job from the pod template of the report's CronJob,
// the way `kubectl create job --from=cronjob/...` does, and waits until it
// succeeded. With args set, the metering tool runs with these arguments
// instead of the CronJob's. It returns the time the Job started.
func runReportJob(ctx context.Context, in Input, report string, args []string) (time.Time, error) {
	cronJob := &batchv1.CronJob{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: cronJobPrefix + report}, cronJob); err != nil {
		return time.Time{}, fmt.Errorf("failed to get the CronJob of report %s: %w", report, err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    in.SeedNamespace,
			GenerateName: cronJob.Name + "-e2e-",
			Labels:       map[string]string{testJobLabel: in.ReportPrefix},
			Annotations:  map[string]string{"cronjob.kubernetes.io/instantiate": "manual"},
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}

	// a failing report run should fail the check quickly, not after six
	// container restarts
	job.Spec.BackoffLimit = ptr.To[int32](1)

	if args != nil {
		job.Spec.Template.Spec.Containers[0].Args = args
	}

	if err := in.SeedClient.Create(ctx, job); err != nil {
		return time.Time{}, fmt.Errorf("failed to create a Job for report %s: %w", report, err)
	}

	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := in.SeedClient.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			in.Log.Warnw("Failed to delete report Job", "job", job.Name, "error", err)
		}
	}()

	// the pod restarts its container on failure and the Job controller
	// deletes the pod once the Job failed, so keep the logs of the last
	// failed attempt while the pod is still there
	var failureLogs string
	err := wait.PollImmediate(ctx, 5*time.Second, jobTimeout, func(ctx context.Context) (error, error) {
		if err := in.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(job), job); err != nil {
			return err, nil
		}

		if logs := jobLogs(ctx, in, job.Name, true); logs != "" {
			failureLogs = logs
		}

		for _, condition := range job.Status.Conditions {
			if condition.Status != corev1.ConditionTrue {
				continue
			}

			switch condition.Type {
			case batchv1.JobComplete:
				return nil, nil
			case batchv1.JobFailed:
				return nil, fmt.Errorf("Job %s failed: %s", job.Name, condition.Message)
			}
		}

		return fmt.Errorf("Job %s is still running", job.Name), nil
	})
	if err != nil {
		logs := failureLogs
		if logs == "" {
			logs = jobLogs(ctx, in, job.Name, false)
		}

		return time.Time{}, fmt.Errorf("%w\n%s", err, logs)
	}

	if job.Status.StartTime == nil {
		return time.Time{}, fmt.Errorf("Job %s has no start time", job.Name)
	}

	return job.Status.StartTime.UTC(), nil
}

// jobLogs returns the last log lines of the Job's pods, for error messages.
// With previous set, it only returns the logs of containers that restarted,
// from their failed attempt, and an empty string when none restarted.
func jobLogs(ctx context.Context, in Input, jobName string, previous bool) string {
	clientset, err := kubernetes.NewForConfig(in.SeedConfig)
	if err != nil {
		return fmt.Sprintf("(no logs: %v)", err)
	}

	pods := &corev1.PodList{}
	if err := in.SeedClient.List(ctx, pods, ctrlruntimeclient.InNamespace(in.SeedNamespace), ctrlruntimeclient.MatchingLabels{"job-name": jobName}); err != nil {
		return fmt.Sprintf("(no logs: %v)", err)
	}

	var logs []string
	for _, pod := range pods.Items {
		if previous && !restarted(pod) {
			continue
		}

		options := &corev1.PodLogOptions{TailLines: ptr.To[int64](20), Previous: previous}
		stream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, options).Stream(ctx)
		if err != nil {
			logs = append(logs, fmt.Sprintf("pod %s: no logs: %v", pod.Name, err))
			continue
		}

		body, err := io.ReadAll(stream)
		stream.Close()
		if err != nil && !errors.Is(err, io.EOF) {
			logs = append(logs, fmt.Sprintf("pod %s: failed to read logs: %v", pod.Name, err))
			continue
		}

		logs = append(logs, fmt.Sprintf("pod %s:\n%s", pod.Name, body))
	}

	return strings.Join(logs, "\n")
}

func restarted(pod corev1.Pod) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.RestartCount > 0 {
			return true
		}
	}

	return false
}

// toolArgs returns the arguments of the report's CronJob that connect the
// metering tool to Prometheus and name the output, without the time window,
// the output directory, the format and the report types.
func toolArgs(ctx context.Context, in Input, report string) ([]string, error) {
	cronJob := &batchv1.CronJob{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: cronJobPrefix + report}, cronJob); err != nil {
		return nil, fmt.Errorf("failed to get the CronJob of report %s: %w", report, err)
	}

	var args []string
	for _, arg := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args {
		for _, keep := range []string{"--ca-bundle=", "--prometheus-api=", "--output-prefix="} {
			if strings.HasPrefix(arg, keep) {
				args = append(args, arg)
			}
		}
	}

	return args, nil
}

// reportKey is the object key the metering tool writes a report to.
func reportKey(dir, seedName, reportType string, start, end time.Time, ext string) string {
	return fmt.Sprintf("%s/%s-%s-report-%s-%s.%s", dir, seedName, reportType, start.Format(time.DateOnly), end.Format(time.DateOnly), ext)
}
