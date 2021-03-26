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

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	utilerror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func printFileUnbuffered(filename string) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	return printUnbuffered(fd)
}

// printUnbuffered uses io.Copy to print data to stdout.
// It should be used for all bigger logs, to avoid buffering
// them in memory and getting oom killed because of that.
func printUnbuffered(src io.Reader) error {
	_, err := io.Copy(os.Stdout, src)
	return err
}

type logExporter interface {
	Export(ctx context.Context, log *zap.SugaredLogger, client kubernetes.Interface, pod *corev1.Pod, containerName string) error
}

type logPrinter struct{}

func (*logPrinter) Export(ctx context.Context, log *zap.SugaredLogger, client kubernetes.Interface, pod *corev1.Pod, containerName string) error {
	readCloser, err := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName}).Stream(ctx)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	return printUnbuffered(readCloser)
}

type logDumper struct {
	directory string
}

func (d *logDumper) Export(ctx context.Context, log *zap.SugaredLogger, client kubernetes.Interface, pod *corev1.Pod, containerName string) error {
	if err := os.MkdirAll(d.directory, 0755); err != nil {
		return err
	}

	filename := filepath.Join(d.directory, fmt.Sprintf("%s-%s-%s.log", pod.Namespace, pod.Name, containerName))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stream, err := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName}).Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	if _, err := io.Copy(f, stream); err != nil {
		return err
	}

	return nil
}

func exportControlPlane(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	k8sclient kubernetes.Interface,
	clusterName string,
	logExporter logExporter,
	eventExporter eventExporter,
) {
	log.Info("Exporting control plane")

	cluster := &kubermaticv1.Cluster{}
	if err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		log.Errorw("Failed to get cluster", zap.Error(err))
		return
	}

	log.Debugw("Cluster health status", "status", cluster.Status.ExtendedHealth)

	log.Info("Exporting events for cluster")
	if err := exportEventsForObject(ctx, log, client, "default", cluster.UID, eventExporter); err != nil {
		log.Errorw("Failed to log cluster events", zap.Error(err))
	}

	if err := exportEventsAndLogsForAllPods(
		ctx,
		log,
		client,
		k8sclient,
		cluster.Status.NamespaceName,
		logExporter,
		eventExporter,
	); err != nil {
		log.Errorw("Failed to export events and logs of pods", zap.Error(err))
	}
}

func exportUserCluster(
	ctx context.Context,
	log *zap.SugaredLogger,
	connProvider *clusterclient.Provider,
	cluster *kubermaticv1.Cluster,
	logExporter logExporter,
	eventExporter eventExporter,
) {
	log.Info("Attempting to log usercluster pod events and logs")
	cfg, err := connProvider.GetClientConfig(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to get usercluster admin kubeconfig", zap.Error(err))
		return
	}

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Errorw("Failed to construct k8sClient for usercluster", zap.Error(err))
		return
	}

	client, err := connProvider.GetClient(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to construct client for usercluster", zap.Error(err))
		return
	}

	if err := exportEventsAndLogsForAllPods(
		ctx,
		log,
		client,
		k8sClient,
		"",
		logExporter,
		eventExporter,
	); err != nil {
		log.Errorw("Failed to export events and logs for usercluster pods", zap.Error(err))
	}
}

func exportEventsAndLogsForAllPods(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	k8sclient kubernetes.Interface,
	namespace string,
	logExporter logExporter,
	eventExporter eventExporter,
) error {
	log.Infow("Exporting logs for all pods", "namespace", namespace)

	pods := &corev1.PodList{}
	if err := client.List(ctx, pods, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	var errs []error
	for _, pod := range pods.Items {
		log := log.With("pod", pod.Name)
		if !podIsReady(&pod) {
			log.Error("Pod is not ready")
		}
		log.Info("Exporting events for pod")
		if err := exportEventsForObject(ctx, log, client, pod.Namespace, pod.UID, eventExporter); err != nil {
			log.Errorw("Failed to log events for pod", zap.Error(err))
			errs = append(errs, err)
		}
		log.Info("Exporting logs for pod")
		if err := exportLogsForPod(ctx, log, k8sclient, &pod, logExporter); err != nil {
			log.Errorw("Failed to print logs for pod", zap.Error(utilerror.NewAggregate(err)))
			errs = append(errs, err...)
		}
	}

	return utilerror.NewAggregate(errs)
}

func exportLogsForPod(ctx context.Context, log *zap.SugaredLogger, k8sclient kubernetes.Interface, pod *corev1.Pod, exporter logExporter) []error {
	var errs []error
	for _, container := range pod.Spec.Containers {
		containerLog := log.With("container", container.Name)
		containerLog.Info("Exporting logs for container")
		if err := exporter.Export(ctx, log, k8sclient, pod, container.Name); err != nil {
			containerLog.Errorw("Failed to print logs for container", zap.Error(err))
			errs = append(errs, err)
		}
	}
	for _, initContainer := range pod.Spec.InitContainers {
		containerLog := log.With("initContainer", initContainer.Name)
		containerLog.Infow("Exporting logs for initContainer")
		if err := exporter.Export(ctx, log, k8sclient, pod, initContainer.Name); err != nil {
			containerLog.Errorw("Failed to print logs for initContainer", zap.Error(err))
			errs = append(errs, err)
		}
	}
	return errs
}

type eventExporter interface {
	Export(ctx context.Context, log *zap.SugaredLogger, namespace string, uid types.UID, event corev1.Event)
}

type eventLogger struct{}

func (*eventLogger) Export(ctx context.Context, log *zap.SugaredLogger, namespace string, uid types.UID, event corev1.Event) {
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

func exportEventsForAllMachines(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	exporter eventExporter,
) {
	machines := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machines); err != nil {
		log.Errorw("Failed to list machines", zap.Error(err))
		return
	}

	for _, machine := range machines.Items {
		machineLog := log.With("name", machine.Name)
		machineLog.Infow("Logging events for machine")
		if err := exportEventsForObject(ctx, log, client, machine.Namespace, machine.UID, exporter); err != nil {
			machineLog.Errorw("Failed to log events for machine", "namespace", machine.Namespace, zap.Error(err))
		}
	}
}

func exportEventsForObject(
	ctx context.Context,
	log *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	namespace string,
	uid types.UID,
	exporter eventExporter,
) error {
	events := &corev1.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     namespace,
		FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(uid)),
	}
	if err := client.List(ctx, events, listOpts); err != nil {
		return fmt.Errorf("failed to get events: %v", err)
	}

	for _, event := range events.Items {
		exporter.Export(ctx, log, namespace, uid, event)
	}

	return nil
}
