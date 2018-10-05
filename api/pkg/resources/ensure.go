package resources

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1beta1client "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	batchv1beta1lister "k8s.io/client-go/listers/batch/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	policyv1beta1lister "k8s.io/client-go/listers/policy/v1beta1"
	rbacv1lister "k8s.io/client-go/listers/rbac/v1"
)

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

// EnsureStatefulSet will create the StatefulSet with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing StatefulSet with the created one
func EnsureStatefulSet(data StatefulSetDataProvider, create StatefulSetCreator, lister appsv1lister.StatefulSetNamespaceLister, client appsv1client.StatefulSetInterface) error {
	var existing *appsv1.StatefulSet
	statefulSet, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build StatefulSet: %v", err)
	}

	if existing, err = lister.Get(statefulSet.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(statefulSet); err != nil {
			return fmt.Errorf("failed to create StatefulSet %s: %v", statefulSet.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	statefulSet, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build StatefulSet: %v", err)
	}

	if DeepEqual(statefulSet, existing) {
		return nil
	}

	// In case we update something immutable we need to delete&recreate. Creation happens on next sync
	if !equality.Semantic.DeepEqual(statefulSet.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
		propagation := metav1.DeletePropagationForeground
		return client.Delete(statefulSet.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
	}

	if _, err = client.Update(statefulSet); err != nil {
		return fmt.Errorf("failed to update StatefulSet %s: %v", statefulSet.Name, err)
	}

	return nil
}

// EnsureDeployment will create the Deployment with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Deployment with the created one
func EnsureDeployment(data DeploymentDataProvider, create DeploymentCreator, lister appsv1lister.DeploymentNamespaceLister, client appsv1client.DeploymentInterface) error {
	var existing *appsv1.Deployment
	dep, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Deployment: %v", err)
	}

	if existing, err = lister.Get(dep.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(dep); err != nil {
			return fmt.Errorf("failed to create Deployment %s: %v", dep.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	dep, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Deployment: %v", err)
	}

	if DeepEqual(dep, existing) {
		return nil
	}

	// In case we update something immutable we need to delete&recreate. Creation happens on next sync
	if !equality.Semantic.DeepEqual(dep.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
		propagation := metav1.DeletePropagationForeground
		return client.Delete(dep.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
	}

	if _, err = client.Update(dep); err != nil {
		return fmt.Errorf("failed to update Deployment %s: %v", dep.Name, err)
	}

	return nil
}

// EnsureService will create the Service with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Service with the created one
func EnsureService(data ServiceDataProvider, create ServiceCreator, lister corev1lister.ServiceNamespaceLister, client corev1client.ServiceInterface) error {
	var existing *corev1.Service
	service, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Service: %v", err)
	}

	if existing, err = lister.Get(service.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(service); err != nil {
			return fmt.Errorf("failed to create Service %s: %v", service.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	service, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Service: %v", err)
	}

	if DeepEqual(service, existing) {
		return nil
	}

	if _, err = client.Update(service); err != nil {
		return fmt.Errorf("failed to update Service %s: %v", service.Name, err)
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

type ConfigMapGetter func(name string) (*corev1.ConfigMap, error)

func GetConfigMapViaLister(lister corev1lister.ConfigMapNamespaceLister) ConfigMapGetter {
	return lister.Get
}

func GetConfigMapViaClient(client corev1client.ConfigMapInterface) ConfigMapGetter {
	return func(name string) (*corev1.ConfigMap, error) {
		return client.Get(name, metav1.GetOptions{})
	}
}

// EnsureConfigMap will create the ConfigMap with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing ConfigMap with the created one
func EnsureConfigMap(data ConfigMapDataProvider, create ConfigMapCreator, get ConfigMapGetter, client corev1client.ConfigMapInterface) error {
	var existing *corev1.ConfigMap
	cm, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build ConfigMap: %v", err)
	}
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
	}
	cm.Annotations[checksumAnnotation] = getChecksumForMapStringString(cm.Data)

	if existing, err = get(cm.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = client.Create(cm); err != nil {
			return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
		}
		return nil
	}
	existing = existing.DeepCopy()

	cm, err = create(data, existing.DeepCopy())
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
