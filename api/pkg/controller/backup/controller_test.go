package backup

import (
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"

	fakekubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kuberinformers "k8s.io/client-go/informers"
	fakekubernetesclientset "k8s.io/client-go/kubernetes/fake"
)

var (
	testStoreContainer = corev1.Container{Name: "kubermatic-store",
		Image:        "busybox",
		VolumeMounts: []corev1.VolumeMount{corev1.VolumeMount{Name: SharedVolumeName, MountPath: "/etcd-backups"}}}
)

func TestEnsureBackupCronJob(t *testing.T) {
	fakeKubeClient := fakekubernetesclientset.NewSimpleClientset()
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}
	fakeKubermaticClient := fakekubermaticclientset.NewSimpleClientset(runtime.Object(cluster))

	kubeInformers := kuberinformers.NewSharedInformerFactory(fakeKubeClient, 10*time.Millisecond)
	kubermaticInformers := externalversions.NewSharedInformerFactory(fakeKubermaticClient, 10*time.Millisecond)

	metricNamespace := "ms"
	subsystem := "subsys"
	metrics := Metrics{
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running backup controller workers",
		}, nil),
		CronJobCreationTimestamp: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_creation_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was created",
		}, []string{"cluster"}),
		CronJobUpdateTimestamp: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_update_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was last updated",
		}, []string{"cluster"}),
	}

	controller, err := New(testStoreContainer,
		20*time.Minute,
		DefaultBackupContainerImage,
		"",
		metrics,
		fakeKubermaticClient,
		fakeKubeClient,
		kubermaticInformers.Kubermatic().V1().Clusters(),
		kubeInformers.Batch().V1beta1().CronJobs())
	if err != nil {
		t.Fatalf("Failed to construct backup controller: %v", err)
	}

	stopChannel := make(chan struct{})
	defer close(stopChannel)

	kubeInformers.Start(stopChannel)
	kubeInformers.WaitForCacheSync(stopChannel)
	kubermaticInformers.Start(stopChannel)
	kubermaticInformers.WaitForCacheSync(stopChannel)
	if err := controller.sync(cluster.Name); err != nil {
		t.Fatalf("Error syncing controller: %v", err)
	}

	cronJobs, err := fakeKubeClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing cronjobs: %v", err)
	}

	if len(cronJobs.Items) != 1 {
		t.Fatalf("Expected exactly one cronjob, got %v", len(cronJobs.Items))
	}

	cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{}
	cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{}
	_, err = fakeKubeClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).Update(&cronJobs.Items[0])
	if err != nil {
		t.Fatalf("Failed to update cronjob")
	}
	kubermaticInformers.WaitForCacheSync(stopChannel)
	if err := controller.sync(cluster.Name); err != nil {
		t.Fatalf("Error syncing controller after updating cronJob: %v", err)
	}

	cronJobs, err = fakeKubeClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing cronjobs after updating cronJob: %v", err)
	}

	if len(cronJobs.Items) != 1 {
		t.Fatalf("Expected exactly one cronjob after updating cronJob, got %v", len(cronJobs.Items))
	}

	if len(cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected exactly one container after manipulating cronjob, got %v", len(cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.Containers))
	}
	if len(cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers) != 1 {
		t.Errorf("Expected exactly one initcontainer after manipulating cronjob, got %v", len(cronJobs.Items[0].Spec.JobTemplate.Spec.Template.Spec.InitContainers))
	}
}
