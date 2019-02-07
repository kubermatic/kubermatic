package resources

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"sort"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	batchv1beta1client "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	batchv1beta1lister "k8s.io/client-go/listers/batch/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	policyv1beta1lister "k8s.io/client-go/listers/policy/v1beta1"
	rbacv1lister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go run ../../codegen/reconcile/main.go

const (
	annotationPrefix   = "kubermatic.io/"
	checksumAnnotation = annotationPrefix + "checksum"
)

// EnsureRole will create the role with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Role with the created one
func EnsureRole(data RoleDataProvider, create RoleCreator, lister rbacv1lister.RoleNamespaceLister, client rbacv1client.RoleInterface) error {
	var existing *rbacv1.Role
	role, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if existing, err = lister.Get(role.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(role); err != nil {
			return fmt.Errorf("failed to create Role %s: %v", role.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	role, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if DeepEqual(role, existing) {
		return nil
	}

	if _, err = client.Update(role); err != nil {
		return fmt.Errorf("failed to update Role %s: %v", role.Name, err)
	}

	return nil
}

// EnsureRoleBinding will create the RoleBinding with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing RoleBinding with the created one
func EnsureRoleBinding(data RoleBindingDataProvider, create RoleBindingCreator, lister rbacv1lister.RoleBindingNamespaceLister, client rbacv1client.RoleBindingInterface) error {
	var existing *rbacv1.RoleBinding
	rb, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build RoleBinding: %v", err)
	}

	if existing, err = lister.Get(rb.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(rb); err != nil {
			return fmt.Errorf("failed to create RoleBinding %s: %v", rb.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	rb, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build RoleBinding: %v", err)
	}

	if DeepEqual(rb, existing) {
		return nil
	}

	if _, err = client.Update(rb); err != nil {
		return fmt.Errorf("failed to update RoleBinding %s: %v", rb.Name, err)
	}

	return nil
}

// EnsureClusterRoleBinding will create the RoleBinding with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing RoleBinding with the created one
func EnsureClusterRoleBinding(data ClusterRoleBindingDataProvider, create ClusterRoleBindingCreator, lister rbacv1lister.ClusterRoleBindingLister, client rbacv1client.ClusterRoleBindingInterface) error {
	var existing *rbacv1.ClusterRoleBinding
	crb, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
	}

	if existing, err = lister.Get(crb.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(crb); err != nil {
			return fmt.Errorf("failed to create ClusterRoleBinding %s: %v", crb.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	crb, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
	}

	if DeepEqual(crb, existing) {
		return nil
	}

	if _, err = client.Update(crb); err != nil {
		return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
	}

	return nil
}

// EnsurePodDisruptionBudget will create the PodDisruptionBudget with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing PodDisruptionBudget with the created one
func EnsurePodDisruptionBudget(data *TemplateData, create PodDisruptionBudgetCreator, pdbLister policyv1beta1lister.PodDisruptionBudgetNamespaceLister, pdbClient policyv1beta1client.PodDisruptionBudgetInterface) error {
	var existing *policyv1beta1.PodDisruptionBudget
	pdb, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build PodDisruptionBudget: %v", err)
	}

	if existing, err = pdbLister.Get(pdb.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = pdbClient.Create(pdb); err != nil {
			return fmt.Errorf("failed to create PodDisruptionBudget %s: %v", pdb.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	pdb, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build PodDisruptionBudget: %v", err)
	}

	if DeepEqual(pdb, existing) {
		return nil
	}

	if _, err = pdbClient.Update(pdb); err != nil {
		return fmt.Errorf("failed to update PodDisruptionBudget %s: %v", pdb.Name, err)
	}

	return nil
}

// EnsureCronJob will create the CronJob with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing CronJob with the created one
func EnsureCronJob(data *TemplateData, create CronJobCreator, cronJobLister batchv1beta1lister.CronJobNamespaceLister, cronJobClient batchv1beta1client.CronJobInterface) error {
	var existing *batchv1beta1.CronJob
	cronjob, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build CronJob: %v", err)
	}

	if existing, err = cronJobLister.Get(cronjob.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = cronJobClient.Create(cronjob); err != nil {
			return fmt.Errorf("failed to create CronJob %s: %v", cronjob.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	cronjob, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build CronJob: %v", err)
	}

	if DeepEqual(cronjob, existing) {
		return nil
	}

	if _, err = cronJobClient.Update(cronjob); err != nil {
		return fmt.Errorf("failed to update CronJob %s: %v", cronjob.Name, err)
	}

	return nil
}

// EnsureServiceAccount will create the ServiceAccount with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing ServiceAccount with the created one
func EnsureServiceAccount(data ServiceAccountDataProvider, create ServiceAccountCreator, lister corev1lister.ServiceAccountNamespaceLister, client corev1client.ServiceAccountInterface) error {
	var existing *corev1.ServiceAccount
	sa, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build ServiceAccount: %v", err)
	}

	if existing, err = lister.Get(sa.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(sa); err != nil {
			return fmt.Errorf("failed to create ServiceAccount %s: %v", sa.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	sa, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build ServiceAccount: %v", err)
	}

	if DeepEqual(sa, existing) {
		return nil
	}

	if _, err = client.Update(sa); err != nil {
		return fmt.Errorf("failed to update ServiceAccount %s: %v", sa.Name, err)
	}

	return nil
}

// EnsureSecret will create the Secret with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Secret with the created one
func EnsureSecret(name string, data SecretDataProvider, create SecretCreator, lister corev1lister.SecretNamespaceLister, client corev1client.SecretInterface) error {
	existing, err := lister.Get(name)
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		// Secret does not exist -> Create it
		secret, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Secret %s: %v", name, err)
		}
		if secret.Annotations == nil {
			secret.Annotations = map[string]string{}
		}
		secret.Annotations[checksumAnnotation] = getChecksumForMapStringByteSlice(secret.Data)

		if _, err = client.Create(secret); err != nil {
			return fmt.Errorf("failed to create Secret %s: %v", secret.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	// Secret already exists, see if we need to update it
	secret, err := create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Secret: %v", err)
	}
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[checksumAnnotation] = getChecksumForMapStringByteSlice(secret.Data)

	annotationVal, annotationExists := existing.Annotations[checksumAnnotation]
	if annotationExists && annotationVal == secret.Annotations[checksumAnnotation] {
		// Nothing to do
		return nil
	}

	if _, err = client.Update(secret); err != nil {
		return fmt.Errorf("failed to update Secret %s: %v", secret.Name, err)
	}
	return nil
}

// EnsureConfigMap will create the ConfigMap with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing ConfigMap with the created one
func EnsureConfigMap(create ConfigMapCreator, lister corev1lister.ConfigMapNamespaceLister, client corev1client.ConfigMapInterface) error {
	var existing *corev1.ConfigMap
	cm, err := create(nil)
	if err != nil {
		return fmt.Errorf("failed to build ConfigMap: %v", err)
	}
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}
	cm.Annotations[checksumAnnotation] = getChecksumForMapStringString(cm.Data)

	if existing, err = lister.Get(cm.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(cm); err != nil {
			return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	cm, err = create(existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build ConfigMap: %v", err)
	}
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}
	cm.Annotations[checksumAnnotation] = getChecksumForMapStringString(cm.Data)

	if existing.Annotations == nil {
		existing.Annotations = map[string]string{}
	}
	annotationVal, annotationExists := existing.Annotations[checksumAnnotation]
	if annotationExists && annotationVal == cm.Annotations[checksumAnnotation] {
		// Nothing to do
		return nil
	}

	if _, err = client.Update(cm); err != nil {
		return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
	}
	return nil
}

func getChecksumForMapStringByteSlice(data map[string][]byte) string {
	// Maps are unordered so we have to sort it first
	var keyVals []string
	for k := range data {
		keyVals = append(keyVals, fmt.Sprintf("%s:%s", k, string(data[k])))
	}
	return getChecksumForStringSlice(keyVals)
}

func getChecksumForMapStringString(data map[string]string) string {
	// Maps are unordered so we have to sort it first
	var keyVals []string
	for k := range data {
		keyVals = append(keyVals, fmt.Sprintf("%s:%s", k, data[k]))
	}
	return getChecksumForStringSlice(keyVals)
}

func getChecksumForStringSlice(stringSlice []string) string {
	sort.Strings(stringSlice)
	buffer := bytes.NewBuffer(nil)
	for _, item := range stringSlice {
		buffer.WriteString(item)
	}
	return fmt.Sprintf("%v", crc32.ChecksumIEEE(buffer.Bytes()))
}

func createWithNamespace(rawcreate ObjectCreator, namespace string) ObjectCreator {
	return func(existing runtime.Object) (runtime.Object, error) {
		obj, err := rawcreate(existing)
		if err != nil {
			return nil, err
		}
		obj.(metav1.Object).SetNamespace(namespace)
		return obj, nil
	}
}

// informerStore is the minimal informer interface we need.
// We're using to ease testing
type informerStore interface {
	GetByKey(key string) (item interface{}, exists bool, err error)
}

// EnsureObject will generate the Object with the passed create function & create or update it in Kubernetes if necessary.
func EnsureObject(namespace string, rawcreate ObjectCreator, store informerStore, client ctrlruntimeclient.Client) error {
	ctx := context.Background()

	// A wrapper to ensure we always set the Namespace. This is useful as we call create twice
	create := createWithNamespace(rawcreate, namespace)

	obj, err := create(nil)
	if err != nil {
		return fmt.Errorf("failed to build Object(%T): %v", obj, err)
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return fmt.Errorf("failed to get key for Object(%T): %v", obj, err)
	}

	iobj, exists, err := store.GetByKey(key)
	if err != nil {
		return err
	}

	// Object does not exist in lister -> Create the Object
	if !exists {
		if err := client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %T '%s': %v", obj, key, err)
		}
		return nil
	}

	// Object does exist in lister -> Update it
	existing, ok := iobj.(runtime.Object)
	if !ok {
		return fmt.Errorf("failed case Object from lister to metav1.Object. Object is %T", iobj)
	}

	// Create a copy to ensure we don't modify any lister state
	existing = existing.DeepCopyObject()

	obj, err = create(existing.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("failed to build Object(%T) '%s': %v", existing, key, err)
	}

	if DeepEqual(obj.(metav1.Object), existing.(metav1.Object)) {
		return nil
	}

	if err = client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object %T '%s': %v", obj, key, err)
	}

	return nil
}
