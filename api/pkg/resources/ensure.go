package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1beta1client "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	batchv1beta1lister "k8s.io/client-go/listers/batch/v1beta1"
	policyv1beta1lister "k8s.io/client-go/listers/policy/v1beta1"
	rbacv1lister "k8s.io/client-go/listers/rbac/v1"
)

// EnsureRole will create the role with the passed create function & create or update it if necessary.
// To check if it's necessary it will do a lookup of the resource at the lister & compare the existing Role with the created one
func EnsureRole(data RoleDataProvider, create RoleCreator, roleLister rbacv1lister.RoleNamespaceLister, roleClient rbacv1client.RoleInterface) error {
	var existing *rbacv1.Role
	role, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if existing, err = roleLister.Get(role.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = roleClient.Create(role); err != nil {
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

	if _, err = roleClient.Update(role); err != nil {
		return fmt.Errorf("failed to update Role %s: %v", role.Name, err)
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
