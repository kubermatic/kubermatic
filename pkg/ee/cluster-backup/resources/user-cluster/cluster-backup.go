package userclusterresources

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

const (
	clusterRoleBindingName = "velero"
	clusterBackupAppName   = "velero"
	defaultBSLName         = "default-cluster-backup-bsl"
)

// NamespaceReconciler creates the namespace for velero related resources on the user cluster
func NamespaceReconciler() reconciling.NamedNamespaceReconcilerFactory {
	return func() (string, reconciling.NamespaceReconciler) {
		return resources.ClusterBackupNamespaceName, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			return ns, nil
		}
	}
}

// ServiceAccountReconciler creates the service account for velero on the user cluster
func ServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return resources.ClusterBackupServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Namespace = resources.ClusterBackupNamespaceName
			return sa, nil
		}
	}
}

// ClusterRoleBindingReconciler creates the clusterrolebinding for velero on the user cluster
func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(clusterBackupAppName, nil)
			crb.RoleRef = rbacv1.RoleRef{
				// too wide but probably needed to be able to do backups and restore.
				Name:     "cluster-admin",
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind: rbacv1.UserKind,
					// Name:      fmt.Sprintf("system:serviceaccount:%s:%s", resources.ClusterBackupNamespaceName, resources.ClusterBackupServiceAccountName),
					Name: resources.ClusterBackupServiceAccountName,
					// Namespace: resources.ClusterBackupNamespaceName,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}
}

// TODO: check and apply spec for updates
// EnsureVeleroBSL Ensure the defatul BackupStorge location is created for velero
func EnsureVeleroBSL(ctx context.Context, client ctrlruntimeclient.Client, clusterBackupConfig *resources.ClusterBackupConfig, clusterName string) error {
	err := client.Get(ctx, types.NamespacedName{Name: defaultBSLName, Namespace: resources.ClusterBackupNamespaceName}, &v1.BackupStorageLocation{})
	if err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	bucketName := clusterBackupConfig.Destination.BucketName
	endPoint := clusterBackupConfig.Destination.Endpoint

	bsl := &v1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultBSLName,
			Namespace: resources.ClusterBackupNamespaceName,
		},
		Spec: v1.BackupStorageLocationSpec{
			Default: true,
			StorageType: v1.StorageType{
				ObjectStorage: &v1.ObjectStorageLocation{
					Bucket: bucketName,
					Prefix: clusterName,
				},
			},
			Provider: "aws",
			Config: map[string]string{
				"region":           "minio",
				"s3Url":            endPoint,
				"s3ForcePathStyle": "true",
			},
		},
	}
	return client.Create(ctx, bsl)
}
