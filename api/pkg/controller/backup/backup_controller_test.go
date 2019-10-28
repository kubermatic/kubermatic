package backup

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "testnamespace",
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Etcd: kubermaticv1.HealthStatusUp,
			},
		},
	}

	caKey, err := triple.NewPrivateKey()
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
			resources.CACertSecretKey: triple.EncodeCertPEM(caCert),
			resources.CAKeySecretKey:  triple.EncodePrivateKeyPEM(caKey),
		},
	}

	reconciler := &Reconciler{
		log:                  kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		storeContainer:       testStoreContainer,
		cleanupContainer:     testCleanupContainer,
		backupContainerImage: DefaultBackupContainerImage,
		Client:               ctrlruntimefakeclient.NewFakeClient(caSecret, cluster),
	}

	if _, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}); err != nil {
		t.Fatalf("Error syncing cluster: %v", err)
	}

	cronJobs := &batchv1beta1.CronJobList{}
	if err := reconciler.List(context.Background(), cronJobs); err != nil {
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

	if err := reconciler.List(context.Background(), cronJobs); err != nil {
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

	secrets := &corev1.SecretList{}
	listOpts := &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}
	if err := reconciler.List(context.Background(), secrets, listOpts); err != nil {
		t.Fatalf("failed to list secrets: %v", err)
	}

	if len(secrets.Items) != 1 {
		t.Fatalf("Expected exactly one secret, got %d", len(secrets.Items))
	}

	expectedName := "cluster-test-cluster-etcd-client-certificate"
	secret := secrets.Items[0]
	if secret.Name != expectedName {
		t.Fatalf("Expected secret name to be %q but was %q", expectedName, secret.Name)
	}

	if len(secret.OwnerReferences) != 1 {
		t.Fatalf("Expectede exactly one owner reference on the secret, got %d", len(secret.OwnerReferences))
	}

	if secret.OwnerReferences[0].Kind != "Cluster" {
		t.Errorf("Expected ownerRef.Kind to be 'Cluster' but was %q", secret.OwnerReferences[0].Kind)
	}
	if secret.OwnerReferences[0].APIVersion != "kubermatic.k8s.io/v1" {
		t.Errorf("Expected ownerRef.APIVersion to be 'kubermatic.k8s.io/v1' but was %q", secret.OwnerReferences[0].APIVersion)
	}
	if secret.OwnerReferences[0].Name != "test-cluster" {
		t.Errorf("Expected ownerRef.Name to be 'test-cluster' but was %q", secret.OwnerReferences[0].Name)
	}
}

func TestCleanupJobSpec(t *testing.T) {
	reconciler := Reconciler{
		cleanupContainer: testCleanupContainer,
	}

	cleanupJob := reconciler.cleanupJob(&kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"}})

	if cleanupJob.Namespace != metav1.NamespaceSystem {
		t.Errorf("expected cleanup jobs Namespace to be %q but was %q", metav1.NamespaceSystem, cleanupJob.Namespace)
	}

	if containerLen := len(cleanupJob.Spec.Template.Spec.Containers); containerLen != 1 {
		t.Errorf("expected cleanup job to have exactly one container, got %d", containerLen)
	}
}
