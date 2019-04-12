package backup

import (
	"context"
	"testing"

	kubermaticscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	testStoreContainer = corev1.Container{Name: "kubermatic-store",
		Image:        "busybox",
		VolumeMounts: []corev1.VolumeMount{{Name: SharedVolumeName, MountPath: "/etcd-backups"}}}
	testCleanupContainer = corev1.Container{Name: "kubermatic-cleanup",
		Image: "busybox",
	}
)

func TestEnsureBackupCronJob(t *testing.T) {
	if err := kubermaticscheme.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermatic scheme: %v", err)
	}

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "testnamespace",
			Health: kubermaticv1.ClusterHealth{
				ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
					Etcd: true,
				},
			},
		},
	}

	caKey, err := certutil.NewPrivateKey()
	if err != nil {
		t.Fatalf("unable to create a private key for the CA: %v", err)
	}

	config := certutil.Config{CommonName: "foo"}
	caCert, err := certutil.NewSelfSignedCACert(config, caKey)
	if err != nil {
		t.Fatalf("unable to create a self-signed certificate for a new CA: %v", err)
	}
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Status.NamespaceName,
			Name:      resources.CASecretName,
		},
		Data: map[string][]byte{
			resources.CACertSecretKey: certutil.EncodeCertPEM(caCert),
			resources.CAKeySecretKey:  certutil.EncodePrivateKeyPEM(caKey),
		},
	}

	reconciler := &Reconciler{
		storeContainer:       testStoreContainer,
		cleanupContainer:     testCleanupContainer,
		backupContainerImage: DefaultBackupContainerImage,
		metrics:              NewMetrics(),
		Client:               ctrlruntimefakeclient.NewFakeClient(caSecret, cluster),
	}

	if _, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}); err != nil {
		t.Fatalf("Error syncing cluster: %v", err)
	}

	cronJobs := &batchv1beta1.CronJobList{}
	if err := reconciler.List(context.Background(), &ctrlruntimeclient.ListOptions{}, cronJobs); err != nil {
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
	if err := reconciler.Update(context.Background(), &cronJobs.Items[0]); err != nil {
		t.Fatalf("Failed to update cronjob")
	}
	if _, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}); err != nil {
		t.Fatalf("Error syncin cluster: %v", err)
	}

	if err := reconciler.List(context.Background(), &ctrlruntimeclient.ListOptions{}, cronJobs); err != nil {
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

func TestCleanupJobSpec(t *testing.T) {
	reconciler := Reconciler{
		cleanupContainer: testCleanupContainer,
	}

	cleanupJob := reconciler.cleanupJob(&kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}})
	if len(cleanupJob.OwnerReferences) != 1 {
		t.Fatalf("cleanup job has no owner ref")
	}

	if cleanupJob.OwnerReferences[0].Kind != "Cluster" {
		t.Errorf("cleanup jobs ownerRef.Kind is not 'Cluster' but %q", cleanupJob.OwnerReferences[0].Kind)
	}

	if cleanupJob.OwnerReferences[0].Name != "test-cluster" {
		t.Errorf("cleanup jobs owner ref does not point to the right cluster, expected 'test-cluster', got %q", cleanupJob.OwnerReferences[0].Name)
	}

	if cleanupJob.Namespace != metav1.NamespaceSystem {
		t.Errorf("expected cleanup jobs Namespace to be %q but was %q", metav1.NamespaceSystem, cleanupJob.Namespace)
	}

	if containerLen := len(cleanupJob.Spec.Template.Spec.Containers); containerLen != 1 {
		t.Errorf("expected cleanup job to have exactly one container, got %d", containerLen)
	}
}
