package backup

import (
	"testing"
	"time"

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
	testCleanupContainer = corev1.Container{Name: "kubermatic-cleanup",
		Image: "busybox",
	}
)

func TestEnsureBackupCronJob(t *testing.T) {
	fakeKubeClient := fakekubernetesclientset.NewSimpleClientset()
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}}
	cluster.Status.NamespaceName = "testnamespace"
	fakeKubermaticClient := fakekubermaticclientset.NewSimpleClientset(runtime.Object(cluster))

	kubeInformers := kuberinformers.NewSharedInformerFactory(fakeKubeClient, 10*time.Millisecond)
	kubermaticInformers := externalversions.NewSharedInformerFactory(fakeKubermaticClient, 10*time.Millisecond)

	controller, err := New(testStoreContainer,
		testCleanupContainer,
		20*time.Minute,
		DefaultBackupContainerImage,
		"",
		NewMetrics(),
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

	if *cronJobs.Items[0].Spec.SuccessfulJobsHistoryLimit != 0 {
		t.Errorf("Expected spec.SuccessfulJobsHistoryLimit to be 0 but was %v",
			*cronJobs.Items[0].Spec.SuccessfulJobsHistoryLimit)
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
