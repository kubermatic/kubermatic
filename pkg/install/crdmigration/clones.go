/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	newv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version/cni"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func DuplicateResources(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	// clone master cluster resources
	if err := cloneResourcesInCluster(ctx, logger.WithField("master", true), opt.MasterClient, opt.KubermaticNamespace, opt.EtcdTimeout); err != nil {
		return fmt.Errorf("processing the master cluster failed: %w", err)
	}

	// clone seed cluster resources
	for seedName, seedClient := range opt.SeedClients {
		if err := cloneResourcesInCluster(ctx, logger.WithField("seed", seedName), seedClient, opt.KubermaticNamespace, opt.EtcdTimeout); err != nil {
			return fmt.Errorf("processing the seed cluster failed: %w", err)
		}
	}

	return nil
}

func cloneResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, etcdTimeout time.Duration) error {
	// reset our runtime UID cache
	uidCache = map[string]types.UID{}

	type cloneFn func(context.Context, logrus.FieldLogger, ctrlruntimeclient.Client) (int, error)

	oldKubermaticAPIVersion := kubermaticv1.SchemeGroupVersion.String()
	oldOperatorAPIVersion := operatorv1alpha1.SchemeGroupVersion.String()
	newKubermaticAPIVersion := newv1.SchemeGroupVersion.String()

	// the order in which resources are migrated is important, as they are interlinked via owner references
	cloneTasks := []struct {
		kind          string
		cloner        cloneFn
		oldAPIVersion string
	}{
		{kind: "KubermaticConfiguration", cloner: cloneKubermaticConfigurationResourcesInCluster, oldAPIVersion: oldOperatorAPIVersion},
		{kind: "User", cloner: cloneUserResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Project", cloner: cloneProjectResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Cluster", cloner: cloneClusterResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Addon", cloner: cloneAddonResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "AddonConfig", cloner: cloneAddonConfigResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "AdmissionPlugin", cloner: cloneAdmissionPluginResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Alertmanager", cloner: cloneAlertmanagerResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "AllowedRegistry", cloner: cloneAllowedRegistryResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "ClusterTemplate", cloner: cloneClusterTemplateResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "ClusterTemplateInstance", cloner: cloneClusterTemplateInstanceResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "ConstraintTemplate", cloner: cloneConstraintTemplateResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Constraint", cloner: cloneConstraintResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "EtcdBackupConfig", cloner: cloneEtcdBackupConfigResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "EtcdRestore", cloner: cloneEtcdRestoreResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "ExternalCluster", cloner: cloneExternalClusterResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "KubermaticSetting", cloner: cloneKubermaticSettingResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "MLAAdminSetting", cloner: cloneMLAAdminSettingResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Preset", cloner: clonePresetResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "RuleGroup", cloner: cloneRuleGroupResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "Seed", cloner: cloneSeedResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "UserProjectBinding", cloner: cloneUserProjectBindingResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
		{kind: "UserSSHKey", cloner: cloneUserSSHKeyResourcesInCluster, oldAPIVersion: oldKubermaticAPIVersion},
	}

	// clone objects, but skip owner references; during this we will
	// build up a cache of UIDs

	logger.Info("Duplicating resources into new API group…")

	for _, task := range cloneTasks {
		logger.Debugf("Duplicating %s objects…", task.kind)

		cloned, err := task.cloner(ctx, logger, client)
		if err != nil {
			return fmt.Errorf("failed to clone %s: %w", task.kind, err)
		}

		logger.Infof("Duplicated %d %s objects.", cloned, task.kind)
	}

	// and now adjust the owner refs
	type ownerRefTask struct {
		kind          string
		oldAPIVersion string
		newAPIVersion string
		namespaces    []string
		drop          bool
	}

	ownerTasks := []ownerRefTask{}
	for _, t := range cloneTasks {
		ownerTasks = append(ownerTasks, ownerRefTask{
			kind:          t.kind,
			oldAPIVersion: t.oldAPIVersion,
			newAPIVersion: newKubermaticAPIVersion,
		})
	}

	// in addition to k8c.io resources, some global resources also have owner refs
	// pointing to k8c.io resources; those resources must also be fixed
	ownerTasks = append(ownerTasks, ownerRefTask{
		kind:          "ClusterRole",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
	}, ownerRefTask{
		kind:          "ClusterRoleBinding",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
	}, ownerRefTask{
		kind:          "Namespace",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
	}, ownerRefTask{
		kind:          "Service",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "Secret",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "ConfigMap",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "Deployment",
		oldAPIVersion: "apps/v1",
		newAPIVersion: "apps/v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "ServiceAccount",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "Role",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "RoleBinding",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "PodDisruptionBudget",
		oldAPIVersion: "policy/v1beta1",
		newAPIVersion: "policy/v1beta1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "Ingress",
		oldAPIVersion: "networking.k8s.io/v1",
		newAPIVersion: "networking.k8s.io/v1",
		namespaces:    []string{kubermaticNamespace},
	}, ownerRefTask{
		kind:          "CronJob",
		oldAPIVersion: "batch/v1",
		newAPIVersion: "batch/v1",
		namespaces:    []string{metav1.NamespaceSystem},
	}, ownerRefTask{
		kind:          "Job",
		oldAPIVersion: "batch/v1",
		newAPIVersion: "batch/v1",
		namespaces:    []string{metav1.NamespaceSystem},
	})

	// older KKP versions set explicit owner refs even on resources living in the
	// cluster namespace; those are dropped in 2.20 and affect pretty much all resources
	// in the cluster namespace
	clusterNamespaces, err := getClusterNamespaces(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to determine usercluster namespaces: %w", err)
	}

	ownerTasks = append(ownerTasks, ownerRefTask{
		kind:          "Role",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "RoleBinding",
		oldAPIVersion: "rbac.authorization.k8s.io/v1",
		newAPIVersion: "rbac.authorization.k8s.io/v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "Service",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "ServiceAccount",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "Secret",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "ConfigMap",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "Deployment",
		oldAPIVersion: "apps/v1",
		newAPIVersion: "apps/v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "StatefulSet", // not etcd, etcd gets special treatment
		oldAPIVersion: "apps/v1",
		newAPIVersion: "apps/v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "CronJob",
		oldAPIVersion: "batch/v1",
		newAPIVersion: "batch/v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "PodDisruptionBudget",
		oldAPIVersion: "policy/v1beta1",
		newAPIVersion: "policy/v1beta1",
		namespaces:    clusterNamespaces,
		drop:          true,
	}, ownerRefTask{
		kind:          "PersistentVolumeClaim",
		oldAPIVersion: "v1",
		newAPIVersion: "v1",
		namespaces:    clusterNamespaces,
		drop:          true,
	})

	logger.Info("Adjusting owner references…")

	for _, task := range ownerTasks {
		logger.Debugf("Adjusting owner references for %s objects…", task.kind)

		err := migrateOwnerReferencesForKind(ctx, logger, client, task.namespaces, task.oldAPIVersion, task.kind, task.newAPIVersion, task.drop)
		if err != nil {
			return fmt.Errorf("failed to update %s objects: %w", task.kind, err)
		}
	}

	// statefulsets are special because of the ownerRef in the PVC claim templates;
	// since the PVC claim template cannot be edited, we must delete the etcd STS
	// and recreate it
	logger.Info("Adjusting owner references for etcd rings…")

	err = migrateOwnerReferencesForEtcd(ctx, logger, client, clusterNamespaces, etcdTimeout)
	if err != nil {
		return fmt.Errorf("failed to update etcd rings: %w", err)
	}

	return nil
}

func getClusterNamespaces(ctx context.Context, client ctrlruntimeclient.Client) ([]string, error) {
	clusterList := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list Clusters: %w", err)
	}

	namespaces := []string{}
	for _, cluster := range clusterList.Items {
		namespaces = append(namespaces, cluster.Status.NamespaceName)
	}

	return namespaces, nil
}

// uidCache is a primitive runtime cache for the UIDs of
// objects created via ensureObject. It is used to fill in
// the UID for owner and object references.
var uidCache = map[string]types.UID{}

func getUIDCacheKey(kind, namespace, name string) string {
	if isNamespacedKind(kind) {
		return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	}

	return fmt.Sprintf("%s/%s", kind, name)
}

func getUIDCacheKeyForObject(obj ctrlruntimeclient.Object, client ctrlruntimeclient.Client) string {
	gvk, err := apiutil.GVKForObject(obj, client.Scheme())
	if err != nil {
		panic(err)
	}

	return getUIDCacheKey(gvk.Kind, obj.GetNamespace(), obj.GetName())
}

func ensureObject(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, cacheUID bool) error {
	oldVersion := ""

	existingObj := &unstructured.Unstructured{}
	existingObj.SetAPIVersion(obj.GetObjectKind().GroupVersionKind().GroupVersion().String())
	existingObj.SetKind(obj.GetObjectKind().GroupVersionKind().Kind)

	// Some kube-apiservers, like GKE, like to fail with a HTTP 500 "please try again later"
	// when attempting to create an object that already exists, probably only when both old
	// and new CRDs exist at the same time and GKE gets confused.
	// To prevent this, we carefully check the existence first.
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), existingObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check for object existence: %w", err)
		}

		if err := client.Create(ctx, obj); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create object: %w", err)
			}
		}
	} else {
		oldVersion = existingObj.GetResourceVersion()

		// forcefully overwrite any differences
		obj.SetResourceVersion(existingObj.GetResourceVersion())
		obj.SetGeneration(existingObj.GetGeneration())

		if err := client.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to update object: %w", err)
		}

		// if there was no actual change, reset the oldVersion so the
		// wait loop below triggers on the first .Get()
		if obj.GetResourceVersion() == oldVersion {
			oldVersion = ""
		}
	}

	// re-fetch the object to
	//    1. fill in its UID so we can cache it
	//    2. get the ResourceVersion, so we can later update any subresources like "status"
	// Get()ing an object fills in the APIVersion and Kind too, but only if the client
	// is backed by a cache (e.g. ctrlruntime's delegatingClient). Since the clients our
	// seedClientGetter provides do not have caches attached, this won't work.
	if err := wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj); err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		return obj.GetResourceVersion() != oldVersion, nil
	}); err != nil {
		return err
	}

	if cacheUID {
		uidCache[getUIDCacheKeyForObject(obj, client)] = obj.GetUID()
	}

	return nil
}

func convertObjectMeta(om metav1.ObjectMeta) metav1.ObjectMeta {
	// owner references are migrated later, once all objects are cloned
	om = *om.DeepCopy()
	om.OwnerReferences = []metav1.OwnerReference{}
	om.UID = ""
	om.Generation = 0
	om.ResourceVersion = ""
	om.CreationTimestamp = metav1.Time{}

	// normally, due to the preflight checks, no object can be in deletion;
	// during development this is possible however, so we still ensure that
	// no deletion info is copied by accident
	om.DeletionTimestamp = nil
	om.DeletionGracePeriodSeconds = nil

	// normalize/rename finalizers
	for i, finalizer := range om.Finalizers {
		finalizer = strings.ReplaceAll(finalizer, "operator.kubermatic.io/", "kubermatic.k8c.io/")
		finalizer = strings.ReplaceAll(finalizer, "kubermatic.io/", "kubermatic.k8c.io/")

		om.Finalizers[i] = finalizer
	}

	return om
}

func migrateOwnerReferencesForKind(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespaces []string, oldAPIVersion string, kind string, newAPIVersion string, dropReferences bool) error {
	list := &unstructured.UnstructuredList{}
	list.SetAPIVersion(oldAPIVersion)
	list.SetKind(kind)

	if err := client.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	selectedNamespaces := sets.NewString(namespaces...)

	for _, oldObj := range list.Items {
		// ignore object in unselected namespace
		if namespaces != nil && !selectedNamespaces.Has(oldObj.GetNamespace()) {
			continue
		}

		// ignore etcd statefulsets, they are processed by migrateOwnerReferencesForEtcd()
		if oldObj.GetName() == resources.EtcdStatefulSetName && kind == "StatefulSet" {
			continue
		}

		ownerRefs := oldObj.GetOwnerReferences()
		if len(ownerRefs) == 0 {
			continue
		}

		newOwnerRefs := migrateOwnerReferences(ownerRefs, oldObj.GetNamespace(), dropReferences)

		if !reflect.DeepEqual(ownerRefs, newOwnerRefs) {
			// fetch the corresponding new object
			newObj := &unstructured.Unstructured{}
			newObj.SetAPIVersion(newAPIVersion)
			newObj.SetKind(kind)
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(&oldObj), newObj); err != nil {
				return fmt.Errorf("failed to get cloned object: %w", err)
			}

			resourceName := ctrlruntimeclient.ObjectKeyFromObject(newObj).String()
			if newObj.GetNamespace() == "" {
				resourceName = newObj.GetName()
			}

			logger.WithField("resource", resourceName).Debugf("Updating %s…", kind)

			// update its owner refs
			newObj.SetOwnerReferences(newOwnerRefs)
			if err := client.Update(ctx, newObj); err != nil {
				return fmt.Errorf("failed to update object: %w", err)
			}
		}
	}

	return nil
}

// migrateOwnerReferencesForEtcd migrated etcd sets. This StatefulSet is special, because
// it not only contains ownerRefs n its own object meta, but the PVC template also has
// an ownerRef. The PVC template is immutable however, so we need to completely recreate
// the StatefulSet and pray to god that no volumes are affected.
func migrateOwnerReferencesForEtcd(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, namespaces []string, timeout time.Duration) error {
	list := &appsv1.StatefulSetList{}
	if err := client.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	selectedNamespaces := sets.NewString(namespaces...)

	for _, sts := range list.Items {
		// ignore object in unselected namespace
		if namespaces != nil && !selectedNamespaces.Has(sts.GetNamespace()) {
			continue
		}

		// ignore everything but etcd statefulsets
		if sts.GetName() != resources.EtcdStatefulSetName {
			continue
		}

		stsOwnerRefs := sts.GetOwnerReferences()

		pvcOwnerRefs := []metav1.OwnerReference{}
		if tpls := sts.Spec.VolumeClaimTemplates; len(tpls) > 0 {
			pvcOwnerRefs = tpls[0].OwnerReferences
		}

		if len(stsOwnerRefs) == 0 && len(pvcOwnerRefs) == 0 {
			continue
		}

		newSTSOwnerRefs := migrateOwnerReferences(stsOwnerRefs, sts.GetNamespace(), true)
		newPVCOwnerRefs := migrateOwnerReferences(pvcOwnerRefs, sts.GetNamespace(), true)

		if !reflect.DeepEqual(stsOwnerRefs, newSTSOwnerRefs) || !reflect.DeepEqual(pvcOwnerRefs, newPVCOwnerRefs) {
			stsLogger := logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&sts))
			stsLogger.Debug("Recreating StatefulSet…")

			// delete the sts
			if err := client.Delete(ctx, &sts); err != nil {
				return fmt.Errorf("failed to delete object: %w", err)
			}

			// recreate it
			newSTS := appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       sts.GetNamespace(),
					Name:            sts.GetName(),
					OwnerReferences: newSTSOwnerRefs,
				},
				Spec: *sts.Spec.DeepCopy(),
			}

			if len(newSTS.Spec.VolumeClaimTemplates) > 0 {
				newSTS.Spec.VolumeClaimTemplates[0].OwnerReferences = newPVCOwnerRefs
			}

			if err := client.Create(ctx, &newSTS); err != nil {
				return fmt.Errorf("failed to create object: %w", err)
			}

			// if enabled, wait for the etcd pods to get ready again; this slows down the
			// migration, but lowers the performance impact of the migration
			if timeout > 0 {
				if err := waitForEtcdReady(ctx, stsLogger, client, ctrlruntimeclient.ObjectKeyFromObject(&sts), timeout); err != nil {
					return fmt.Errorf("failed to wait for etcd: %w", err)
				}
			}
		}
	}

	return nil
}

func waitForEtcdReady(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, key types.NamespacedName, timeout time.Duration) error {
	logger.Debugf("Waiting up to %v for etcd to become ready…", timeout)

	err := wait.Poll(1*time.Second, timeout, func() (done bool, err error) {
		sts := &appsv1.StatefulSet{}
		if err := client.Get(ctx, key, sts); err != nil {
			return false, err
		}

		return sts.Status.ReadyReplicas == *sts.Spec.Replicas, nil
	})

	// etcd being "slow" is not an error per-se; treating this as a warning only allows the
	// user to choose a very small timeout (5s) only to slow down the migration a bit, without
	// blocking for each user cluster.
	if errors.Is(err, wait.ErrWaitTimeout) {
		logger.Warn("StatefulSet did not become ready within the timeout.")
		return nil
	}

	return err
}

func migrateOwnerReferences(ownerRefs []metav1.OwnerReference, namespace string, dropReferences bool) []metav1.OwnerReference {
	result := []metav1.OwnerReference{}

	for _, ref := range ownerRefs {
		if dropReferences && (ref.APIVersion == oldAPIGroupVersion || ref.APIVersion == newAPIGroupVersion) {
			continue
		}

		newRef := ref.DeepCopy()

		if newRef.APIVersion == oldAPIGroupVersion {
			newRef.APIVersion = newAPIGroupVersion

			cacheKey := getUIDCacheKey(newRef.Kind, namespace, newRef.Name)
			uid, exists := uidCache[cacheKey]
			if !exists {
				panic(fmt.Sprintf("Cannot find UID for %s in cache. Make sure to create %s first.", cacheKey, newRef.Kind))
			}

			newRef.UID = uid
		}

		result = append(result, *newRef)
	}

	return result
}

func migrateObjectReference(objectRef corev1.ObjectReference, namespace string) corev1.ObjectReference {
	newRef := *objectRef.DeepCopy()

	if newRef.APIVersion == oldAPIGroupVersion {
		newRef.APIVersion = newAPIGroupVersion

		if newRef.Namespace != "" {
			namespace = newRef.Namespace
		}

		cacheKey := getUIDCacheKey(newRef.Kind, namespace, newRef.Name)
		uid, exists := uidCache[cacheKey]
		if !exists {
			panic(fmt.Sprintf("Cannot find UID for %s in cache. Make sure to create %s first.", cacheKey, newRef.Kind))
		}

		newRef.UID = uid
	}

	return newRef
}

func cloneKubermaticConfigurationResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &operatorv1alpha1.KubermaticConfigurationList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		versions, err := convertKubermaticVersioningConfiguration(oldObject.Spec.Versions.Kubernetes)
		if err != nil {
			return 0, fmt.Errorf("KubermaticConfiguration %s is invalid: %w", oldObject.Name, err)
		}

		newObject := newv1.KubermaticConfiguration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "KubermaticConfiguration",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.KubermaticConfigurationSpec{
				CABundle:        oldObject.Spec.CABundle,
				ImagePullSecret: oldObject.Spec.ImagePullSecret,
				Auth:            newv1.KubermaticAuthConfiguration(oldObject.Spec.Auth),
				FeatureGates:    map[string]bool{},
				UI:              newv1.KubermaticUIConfiguration(oldObject.Spec.UI),
				API:             newv1.KubermaticAPIConfiguration(oldObject.Spec.API),
				SeedController: newv1.KubermaticSeedControllerConfiguration{
					DockerRepository:          oldObject.Spec.SeedController.DockerRepository,
					BackupStoreContainer:      oldObject.Spec.SeedController.BackupStoreContainer,
					BackupDeleteContainer:     oldObject.Spec.SeedController.BackupDeleteContainer,
					BackupCleanupContainer:    oldObject.Spec.SeedController.BackupCleanupContainer,
					MaximumParallelReconciles: oldObject.Spec.SeedController.MaximumParallelReconciles,
					PProfEndpoint:             oldObject.Spec.SeedController.PProfEndpoint,
					Resources:                 oldObject.Spec.SeedController.Resources,
					DebugLog:                  oldObject.Spec.SeedController.DebugLog,
					Replicas:                  oldObject.Spec.SeedController.Replicas,
				},
				MasterController: newv1.KubermaticMasterControllerConfiguration{
					DockerRepository: oldObject.Spec.MasterController.DockerRepository,
					ProjectsMigrator: newv1.KubermaticProjectsMigratorConfiguration(oldObject.Spec.MasterController.ProjectsMigrator),
					PProfEndpoint:    oldObject.Spec.MasterController.PProfEndpoint,
					Resources:        oldObject.Spec.MasterController.Resources,
					DebugLog:         oldObject.Spec.MasterController.DebugLog,
					Replicas:         oldObject.Spec.MasterController.Replicas,
				},
				UserCluster: newv1.KubermaticUserClusterConfiguration{
					KubermaticDockerRepository:          oldObject.Spec.UserCluster.KubermaticDockerRepository,
					DNATControllerDockerRepository:      oldObject.Spec.UserCluster.DNATControllerDockerRepository,
					EtcdLauncherDockerRepository:        oldObject.Spec.UserCluster.EtcdLauncherDockerRepository,
					OverwriteRegistry:                   oldObject.Spec.UserCluster.OverwriteRegistry,
					Addons:                              newv1.KubermaticAddonsConfiguration(oldObject.Spec.UserCluster.Addons.Kubernetes),
					NodePortRange:                       oldObject.Spec.UserCluster.NodePortRange,
					Monitoring:                          newv1.KubermaticUserClusterMonitoringConfiguration(oldObject.Spec.UserCluster.Monitoring),
					DisableAPIServerEndpointReconciling: oldObject.Spec.UserCluster.DisableAPIServerEndpointReconciling,
					EtcdVolumeSize:                      oldObject.Spec.UserCluster.EtcdVolumeSize,
					APIServerReplicas:                   oldObject.Spec.UserCluster.APIServerReplicas,
					MachineController:                   newv1.MachineControllerConfiguration(oldObject.Spec.UserCluster.MachineController),
				},
				ExposeStrategy: oldObject.Spec.ExposeStrategy,
				Ingress:        newv1.KubermaticIngressConfiguration(oldObject.Spec.Ingress),
				Versions:       *versions,
				VerticalPodAutoscaler: newv1.KubermaticVPAConfiguration{
					Recommender:         newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.Recommender),
					Updater:             newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.Updater),
					AdmissionController: newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.AdmissionController),
				},
				Proxy: newv1.KubermaticProxyConfiguration(oldObject.Spec.Proxy),
			},
		}

		for _, feature := range oldObject.Spec.FeatureGates.List() {
			newObject.Spec.FeatureGates[feature] = true
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating KubermaticConfiguration…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertKubermaticVersioningConfiguration(old operatorv1alpha1.KubermaticVersioningConfiguration) (*newv1.KubermaticVersioningConfiguration, error) {
	result := newv1.KubermaticVersioningConfiguration{}

	if old.Default != nil {
		parsed, err := semver.NewSemver(old.Default.String())
		if err != nil {
			return nil, fmt.Errorf("invalid default version %q: %w", old.Default.String(), err)
		}

		result.Default = parsed
	}

	for _, v := range old.Versions {
		result.Versions = append(result.Versions, semver.Semver(v.String()))
	}

	for _, u := range old.Updates {
		result.Updates = append(result.Updates, newv1.Update(u))
	}

	for _, i := range old.ProviderIncompatibilities {
		result.ProviderIncompatibilities = append(result.ProviderIncompatibilities, newv1.Incompatibility{
			Provider:  i.Provider,
			Version:   i.Version,
			Condition: newv1.ConditionType(i.Condition),
			Operation: newv1.OperationType(i.Operation),
		})
	}

	return &result, nil
}

func convertHealthStatus(oldHs kubermaticv1.HealthStatus) newv1.HealthStatus {
	switch oldHs {
	case kubermaticv1.HealthStatusUp:
		return newv1.HealthStatusUp
	case kubermaticv1.HealthStatusProvisioning:
		return newv1.HealthStatusProvisioning
	}

	return newv1.HealthStatusDown
}

func convertHealthStatusPtr(oldHs *kubermaticv1.HealthStatus) *newv1.HealthStatus {
	var status *newv1.HealthStatus
	if oldHs != nil {
		status1 := convertHealthStatus(*oldHs)
		status = &status1
	}
	return status
}

func cloneClusterResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Cluster",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Address:    newv1.ClusterAddress(oldObject.Address),
			Spec:       convertClusterSpec(oldObject.Spec),
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating Cluster…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.ClusterStatus{
			NamespaceName:          oldObject.Status.NamespaceName,
			CloudMigrationRevision: oldObject.Status.CloudMigrationRevision,
			LastUpdated:            oldObject.Status.LastUpdated,
			UserName:               oldObject.Status.UserName,
			UserEmail:              oldObject.Status.UserEmail,
			InheritedLabels:        oldObject.Status.InheritedLabels,
			ExtendedHealth: newv1.ExtendedClusterHealth{
				Apiserver:                    convertHealthStatus(oldObject.Status.ExtendedHealth.Apiserver),
				Scheduler:                    convertHealthStatus(oldObject.Status.ExtendedHealth.Scheduler),
				Controller:                   convertHealthStatus(oldObject.Status.ExtendedHealth.Controller),
				MachineController:            convertHealthStatus(oldObject.Status.ExtendedHealth.MachineController),
				Etcd:                         convertHealthStatus(oldObject.Status.ExtendedHealth.Etcd),
				OpenVPN:                      convertHealthStatus(oldObject.Status.ExtendedHealth.OpenVPN),
				CloudProviderInfrastructure:  convertHealthStatus(oldObject.Status.ExtendedHealth.CloudProviderInfrastructure),
				UserClusterControllerManager: convertHealthStatus(oldObject.Status.ExtendedHealth.UserClusterControllerManager),
				GatekeeperController:         convertHealthStatusPtr(oldObject.Status.ExtendedHealth.GatekeeperController),
				GatekeeperAudit:              convertHealthStatusPtr(oldObject.Status.ExtendedHealth.GatekeeperAudit),
				Monitoring:                   convertHealthStatusPtr(oldObject.Status.ExtendedHealth.Monitoring),
				Logging:                      convertHealthStatusPtr(oldObject.Status.ExtendedHealth.Logging),
				AlertmanagerConfig:           convertHealthStatusPtr(oldObject.Status.ExtendedHealth.AlertmanagerConfig),
				MLAGateway:                   convertHealthStatusPtr(oldObject.Status.ExtendedHealth.MLAGateway),
			},
		}

		if oldObject.Status.LastProviderReconciliation != nil {
			newObject.Status.LastProviderReconciliation = *oldObject.Status.LastProviderReconciliation
		}

		for _, condition := range oldObject.Status.Conditions {
			if newObject.Status.Conditions == nil {
				newObject.Status.Conditions = map[newv1.ClusterConditionType]newv1.ClusterCondition{}
			}

			conditionType := newv1.ClusterConditionType(condition.Type)
			newObject.Status.Conditions[conditionType] = newv1.ClusterCondition{
				Status:             condition.Status,
				KubermaticVersion:  condition.KubermaticVersion,
				LastHeartbeatTime:  condition.LastHeartbeatTime,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			}
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertAzureLoadBalancerSKU(old kubermaticv1.LBSKU) newv1.LBSKU {
	if old == "" {
		return newv1.AzureBasicLBSKU
	}

	return newv1.LBSKU(old)
}

func convertClusterSpec(old kubermaticv1.ClusterSpec) newv1.ClusterSpec {
	proxyMode := old.ClusterNetwork.ProxyMode
	if proxyMode == "" {
		proxyMode = resources.IPVSProxyMode
	}

	result := newv1.ClusterSpec{
		Cloud: newv1.CloudSpec{
			DatacenterName: old.Cloud.DatacenterName,
			ProviderName:   old.Cloud.ProviderName,

			// AWS, Azure, VSphere, Openstack, and Kubevirt need special treatment further down
			Alibaba:      (*newv1.AlibabaCloudSpec)(old.Cloud.Alibaba),
			Anexia:       (*newv1.AnexiaCloudSpec)(old.Cloud.Anexia),
			BringYourOwn: (*newv1.BringYourOwnCloudSpec)(old.Cloud.BringYourOwn),
			Digitalocean: (*newv1.DigitaloceanCloudSpec)(old.Cloud.Digitalocean),
			Fake:         (*newv1.FakeCloudSpec)(old.Cloud.Fake),
			GCP:          (*newv1.GCPCloudSpec)(old.Cloud.GCP),
			Hetzner:      (*newv1.HetznerCloudSpec)(old.Cloud.Hetzner),
			Packet:       (*newv1.PacketCloudSpec)(old.Cloud.Packet),
			Nutanix:      convertNutanixCloudSpec(old.Cloud.Nutanix),
		},
		ClusterNetwork: newv1.ClusterNetworkingConfig{
			Pods:                     newv1.NetworkRanges(old.ClusterNetwork.Pods),
			Services:                 newv1.NetworkRanges(old.ClusterNetwork.Services),
			DNSDomain:                old.ClusterNetwork.DNSDomain,
			ProxyMode:                proxyMode,
			IPVS:                     (*newv1.IPVSConfiguration)(old.ClusterNetwork.IPVS),
			NodeLocalDNSCacheEnabled: old.ClusterNetwork.NodeLocalDNSCacheEnabled,
			KonnectivityEnabled:      old.ClusterNetwork.KonnectivityEnabled,
		},
		Version:                              old.Version,
		HumanReadableName:                    old.HumanReadableName,
		ExposeStrategy:                       newv1.ExposeStrategy(old.ExposeStrategy),
		Pause:                                old.Pause,
		PauseReason:                          old.PauseReason,
		DebugLog:                             old.DebugLog,
		ComponentsOverride:                   convertComponentSettings(old.ComponentsOverride),
		OIDC:                                 newv1.OIDCSettings(old.OIDC),
		Features:                             old.Features,
		UpdateWindow:                         (*newv1.UpdateWindow)(old.UpdateWindow),
		UsePodSecurityPolicyAdmissionPlugin:  old.UsePodSecurityPolicyAdmissionPlugin,
		UsePodNodeSelectorAdmissionPlugin:    old.UsePodNodeSelectorAdmissionPlugin,
		UseEventRateLimitAdmissionPlugin:     old.UseEventRateLimitAdmissionPlugin,
		EnableUserSSHKeyAgent:                old.EnableUserSSHKeyAgent,
		EnableOperatingSystemManager:         old.EnableOperatingSystemManager,
		PodNodeSelectorAdmissionPluginConfig: old.PodNodeSelectorAdmissionPluginConfig,
		AdmissionPlugins:                     old.AdmissionPlugins,
		OPAIntegration:                       (*newv1.OPAIntegrationSettings)(old.OPAIntegration),
		ServiceAccount:                       (*newv1.ServiceAccountSettings)(old.ServiceAccount),
		MLA:                                  (*newv1.MLASettings)(old.MLA),
		ContainerRuntime:                     old.ContainerRuntime,
	}

	// borrowed from the ClusterSpec defaulting, because not all existing templates
	// might have the CIDRs set
	if len(result.ClusterNetwork.Services.CIDRBlocks) == 0 {
		if result.Cloud.Kubevirt != nil {
			// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
			// thus we have to avoid network collision
			result.ClusterNetwork.Services.CIDRBlocks = []string{"10.241.0.0/20"}
		} else {
			result.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20"}
		}
	}

	if len(result.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		if result.Cloud.Kubevirt != nil {
			result.ClusterNetwork.Pods.CIDRBlocks = []string{"172.26.0.0/16"}
		} else {
			result.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
	}

	if result.ExposeStrategy == "" {
		result.ExposeStrategy = defaults.DefaultExposeStrategy
	}

	if old := old.Cloud.AWS; old != nil {
		result.Cloud.AWS = &newv1.AWSCloudSpec{
			CredentialsReference:    old.CredentialsReference,
			AccessKeyID:             old.AccessKeyID,
			SecretAccessKey:         old.SecretAccessKey,
			AssumeRoleARN:           old.AssumeRoleARN,
			AssumeRoleExternalID:    old.AssumeRoleExternalID,
			VPCID:                   old.VPCID,
			ControlPlaneRoleARN:     old.ControlPlaneRoleARN,
			RouteTableID:            old.RouteTableID,
			InstanceProfileName:     old.InstanceProfileName,
			SecurityGroupID:         old.SecurityGroupID,
			NodePortsAllowedIPRange: old.NodePortsAllowedIPRange,
		}
	}

	if old := old.Cloud.Azure; old != nil {
		result.Cloud.Azure = &newv1.AzureCloudSpec{
			CredentialsReference:    old.CredentialsReference,
			TenantID:                old.TenantID,
			SubscriptionID:          old.SubscriptionID,
			ClientID:                old.ClientID,
			ClientSecret:            old.ClientSecret,
			ResourceGroup:           old.ResourceGroup,
			VNetResourceGroup:       old.VNetResourceGroup,
			VNetName:                old.VNetName,
			SubnetName:              old.SubnetName,
			RouteTableName:          old.RouteTableName,
			SecurityGroup:           old.SecurityGroup,
			NodePortsAllowedIPRange: old.NodePortsAllowedIPRange,
			AssignAvailabilitySet:   old.AssignAvailabilitySet,
			AvailabilitySet:         old.AvailabilitySet,
			LoadBalancerSKU:         convertAzureLoadBalancerSKU(old.LoadBalancerSKU),
		}
	}

	if old := old.Cloud.VSphere; old != nil {
		result.Cloud.VSphere = &newv1.VSphereCloudSpec{
			CredentialsReference: old.CredentialsReference,
			Username:             old.Username,
			Password:             old.Password,
			VMNetName:            old.VMNetName,
			Folder:               old.Folder,
			Datastore:            old.Datastore,
			DatastoreCluster:     old.DatastoreCluster,
			StoragePolicy:        old.StoragePolicy,
			ResourcePool:         old.ResourcePool,
			InfraManagementUser:  newv1.VSphereCredentials(old.InfraManagementUser),
		}
	}

	if old := old.Cloud.Kubevirt; old != nil {
		result.Cloud.Kubevirt = &newv1.KubevirtCloudSpec{
			CredentialsReference: old.CredentialsReference,
			Kubeconfig:           old.Kubeconfig,
			CSIKubeconfig:        "",
		}
	}

	if old := old.Cloud.Openstack; old != nil {
		result.Cloud.Openstack = &newv1.OpenstackCloudSpec{
			CredentialsReference:        old.CredentialsReference,
			Username:                    old.Username,
			Password:                    old.Password,
			Project:                     old.GetProject(),
			ProjectID:                   old.GetProjectId(),
			Domain:                      old.Domain,
			ApplicationCredentialID:     old.ApplicationCredentialID,
			ApplicationCredentialSecret: old.ApplicationCredentialSecret,
			UseToken:                    old.UseToken,
			Token:                       old.Token,
			Network:                     old.Network,
			SecurityGroups:              old.SecurityGroups,
			NodePortsAllowedIPRange:     old.NodePortsAllowedIPRange,
			FloatingIPPool:              old.FloatingIPPool,
			RouterID:                    old.RouterID,
			SubnetID:                    old.SubnetID,
			UseOctavia:                  old.UseOctavia,
		}
	}

	if old := old.CNIPlugin; old != nil {
		result.CNIPlugin = &newv1.CNIPluginSettings{
			Type:    newv1.CNIPluginType(old.Type),
			Version: old.Version,
		}
	} else {
		result.CNIPlugin = &newv1.CNIPluginSettings{
			Type:    newv1.CNIPluginTypeCanal,
			Version: cni.CanalCNILastUnspecifiedVersion,
		}
	}

	if old := old.AuditLogging; old != nil {
		result.AuditLogging = &newv1.AuditLoggingSettings{
			Enabled:      old.Enabled,
			PolicyPreset: newv1.AuditPolicyPreset(old.PolicyPreset),
		}
	}

	if old := old.EventRateLimitConfig; old != nil {
		result.EventRateLimitConfig = &newv1.EventRateLimitConfig{}
		if item := old.Server; item != nil {
			result.EventRateLimitConfig.Server = &newv1.EventRateLimitConfigItem{
				QPS:       item.QPS,
				Burst:     item.Burst,
				CacheSize: item.CacheSize,
			}
		}

		if item := old.Namespace; item != nil {
			result.EventRateLimitConfig.Namespace = &newv1.EventRateLimitConfigItem{
				QPS:       item.QPS,
				Burst:     item.Burst,
				CacheSize: item.CacheSize,
			}
		}

		if item := old.User; item != nil {
			result.EventRateLimitConfig.User = &newv1.EventRateLimitConfigItem{
				QPS:       item.QPS,
				Burst:     item.Burst,
				CacheSize: item.CacheSize,
			}
		}

		if item := old.SourceAndObject; item != nil {
			result.EventRateLimitConfig.SourceAndObject = &newv1.EventRateLimitConfigItem{
				QPS:       item.QPS,
				Burst:     item.Burst,
				CacheSize: item.CacheSize,
			}
		}
	}

	for _, network := range old.MachineNetworks {
		result.MachineNetworks = append(result.MachineNetworks, newv1.MachineNetworkingConfig(network))
	}

	return result
}

func convertNutanixCloudSpec(old *kubermaticv1.NutanixCloudSpec) *newv1.NutanixCloudSpec {
	if old == nil {
		return nil
	}
	return &newv1.NutanixCloudSpec{
		CredentialsReference: old.CredentialsReference,
		ClusterName:          old.ClusterName,
		ProjectName:          old.ProjectName,
		ProxyURL:             old.ProxyURL,
		Username:             old.Username,
		Password:             old.Password,
		CSI:                  nil,
	}
}

func cloneAddonResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AddonList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Addon objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Addon{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Addon",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AddonSpec{
				Name:      oldObject.Spec.Name,
				Cluster:   migrateObjectReference(oldObject.Spec.Cluster, ""),
				IsDefault: oldObject.Spec.IsDefault,
			},
		}

		if oldObject.Spec.Variables.Raw != nil {
			newObject.Spec.Variables = oldObject.Spec.Variables.DeepCopy()
		}

		for _, t := range oldObject.Spec.RequiredResourceTypes {
			newObject.Spec.RequiredResourceTypes = append(newObject.Spec.RequiredResourceTypes, newv1.GroupVersionKind(t))
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating Addon…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		for _, condition := range oldObject.Status.Conditions {
			if newObject.Status.Conditions == nil {
				newObject.Status.Conditions = map[newv1.AddonConditionType]newv1.AddonCondition{}
			}

			newObject.Status.Conditions[newv1.AddonConditionType(condition.Type)] = newv1.AddonCondition{
				Status:             condition.Status,
				LastHeartbeatTime:  condition.LastHeartbeatTime,
				LastTransitionTime: condition.LastTransitionTime,
			}
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAddonConfigResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AddonConfigList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AddonConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "AddonConfig",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AddonConfigSpec{
				ShortDescription: oldObject.Spec.ShortDescription,
				Description:      oldObject.Spec.Description,
				Logo:             oldObject.Spec.Logo,
				LogoFormat:       oldObject.Spec.LogoFormat,
				Controls:         []newv1.AddonFormControl{},
			},
		}

		for _, ctrl := range oldObject.Spec.Controls {
			newObject.Spec.Controls = append(newObject.Spec.Controls, newv1.AddonFormControl(ctrl))
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating AddonConfig…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAdmissionPluginResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AdmissionPluginList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AdmissionPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "AdmissionPlugin",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AdmissionPluginSpec{
				PluginName:  oldObject.Spec.PluginName,
				FromVersion: oldObject.Spec.FromVersion,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating AdmissionPlugin…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAlertmanagerResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AlertmanagerList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Alertmanager{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Alertmanager",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AlertmanagerSpec{
				ConfigSecret: oldObject.Spec.DeepCopy().ConfigSecret,
			},
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating Alertmanager…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.AlertmanagerStatus{
			ConfigStatus: newv1.AlertmanagerConfigurationStatus{
				LastUpdated:  oldObject.Status.ConfigStatus.LastUpdated,
				Status:       oldObject.Status.ConfigStatus.Status,
				ErrorMessage: oldObject.Status.ConfigStatus.ErrorMessage,
			},
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAllowedRegistryResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AllowedRegistryList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AllowedRegistry{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "AllowedRegistry",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AllowedRegistrySpec{
				RegistryPrefix: oldObject.Spec.RegistryPrefix,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating AllowedRegistry…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneClusterTemplateResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterTemplateList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ClusterTemplate{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "ClusterTemplate",
			},
			ObjectMeta:             convertObjectMeta(oldObject.ObjectMeta),
			Credential:             oldObject.Credential,
			ClusterLabels:          oldObject.ClusterLabels,
			InheritedClusterLabels: oldObject.InheritedClusterLabels,
			Spec:                   convertClusterSpec(oldObject.Spec),
			UserSSHKeys:            []newv1.ClusterTemplateSSHKey{},
		}

		for _, key := range oldObject.UserSSHKeys {
			newObject.UserSSHKeys = append(newObject.UserSSHKeys, newv1.ClusterTemplateSSHKey(key))
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating ClusterTemplate…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneClusterTemplateInstanceResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterTemplateInstanceList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ClusterTemplateInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "ClusterTemplateInstance",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ClusterTemplateInstanceSpec{
				ProjectID:           oldObject.Spec.ProjectID,
				ClusterTemplateID:   oldObject.Spec.ClusterTemplateID,
				ClusterTemplateName: oldObject.Spec.ClusterTemplateName,
				Replicas:            oldObject.Spec.Replicas,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating ClusterTemplateInstance…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneConstraintTemplateResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ConstraintTemplateList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ConstraintTemplate{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "ConstraintTemplate",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ConstraintTemplateSpec{
				CRD:      oldObject.Spec.CRD,
				Selector: newv1.ConstraintTemplateSelector(oldObject.Spec.Selector),
				Targets:  oldObject.Spec.Targets,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating ConstraintTemplate…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneConstraintResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ConstraintList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Constraint{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Constraint",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ConstraintSpec{
				ConstraintType: oldObject.Spec.ConstraintType,
				Disabled:       oldObject.Spec.Disabled,
				Match: newv1.Match{
					Scope:              oldObject.Spec.Match.Scope,
					Namespaces:         oldObject.Spec.Match.Namespaces,
					ExcludedNamespaces: oldObject.Spec.Match.ExcludedNamespaces,
					LabelSelector:      oldObject.Spec.Match.LabelSelector,
					NamespaceSelector:  oldObject.Spec.Match.NamespaceSelector,
				},
				Parameters: newv1.Parameters(oldObject.Spec.Parameters),
				Selector:   newv1.ConstraintSelector(oldObject.Spec.Selector),
			},
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating Constraint…")

		for _, kind := range oldObject.Spec.Match.Kinds {
			newObject.Spec.Match.Kinds = append(newObject.Spec.Match.Kinds, newv1.Kind(kind))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneEtcdBackupConfigResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.EtcdBackupConfigList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.EtcdBackupConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "EtcdBackupConfig",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.EtcdBackupConfigSpec{
				Name:        oldObject.Spec.Name,
				Schedule:    oldObject.Spec.Schedule,
				Keep:        oldObject.Spec.Keep,
				Cluster:     migrateObjectReference(oldObject.Spec.Cluster, ""),
				Destination: oldObject.Spec.Destination,
			},
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating EtcdBackupConfig…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.EtcdBackupConfigStatus{
			CleanupRunning: oldObject.Status.CleanupRunning,
		}

		for _, condition := range oldObject.Status.Conditions {
			if newObject.Status.Conditions == nil {
				newObject.Status.Conditions = map[newv1.EtcdBackupConfigConditionType]newv1.EtcdBackupConfigCondition{}
			}

			heartbeat := condition.LastHeartbeatTime
			if heartbeat.IsZero() {
				heartbeat = metav1.Now()
			}

			conditionType := newv1.EtcdBackupConfigConditionType(condition.Type)
			newObject.Status.Conditions[conditionType] = newv1.EtcdBackupConfigCondition{
				Status:             condition.Status,
				LastHeartbeatTime:  heartbeat,
				LastTransitionTime: condition.LastTransitionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			}
		}

		for _, backup := range oldObject.Status.CurrentBackups {
			newBackup := newv1.BackupStatus{
				BackupName:    backup.BackupName,
				JobName:       backup.JobName,
				BackupPhase:   newv1.BackupStatusPhase(backup.BackupPhase),
				BackupMessage: backup.BackupMessage,
				DeleteJobName: backup.DeleteJobName,
				DeletePhase:   newv1.BackupStatusPhase(backup.DeletePhase),
				DeleteMessage: backup.DeleteMessage,
			}

			if backup.ScheduledTime != nil {
				newBackup.ScheduledTime = *backup.ScheduledTime
			}
			if backup.BackupStartTime != nil {
				newBackup.BackupStartTime = *backup.BackupStartTime
			}
			if backup.BackupFinishedTime != nil {
				newBackup.BackupFinishedTime = *backup.BackupFinishedTime
			}
			if backup.DeleteStartTime != nil {
				newBackup.DeleteStartTime = *backup.DeleteStartTime
			}
			if backup.DeleteFinishedTime != nil {
				newBackup.DeleteFinishedTime = *backup.DeleteFinishedTime
			}

			newObject.Status.CurrentBackups = append(newObject.Status.CurrentBackups, newBackup)
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneEtcdRestoreResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.EtcdRestoreList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.EtcdRestore{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "EtcdRestore",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.EtcdRestoreSpec{
				Name:                            oldObject.Spec.Name,
				BackupDownloadCredentialsSecret: oldObject.Spec.BackupDownloadCredentialsSecret,
				BackupName:                      oldObject.Spec.BackupName,
				Cluster:                         migrateObjectReference(oldObject.Spec.Cluster, ""),
				Destination:                     oldObject.Spec.Destination,
			},
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating EtcdRestore…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.EtcdRestoreStatus{
			Phase: newv1.EtcdRestorePhase(oldObject.Status.Phase),
		}

		if oldObject.Status.RestoreTime != nil {
			newObject.Status.RestoreTime = *oldObject.Status.RestoreTime
		}

		if newObject.Status.Phase == "" {
			newObject.Status.Phase = newv1.EtcdRestorePhaseStarted
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneExternalClusterResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ExternalClusterList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ExternalCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "ExternalCluster",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ExternalClusterSpec{
				HumanReadableName:   oldObject.Spec.HumanReadableName,
				KubeconfigReference: oldObject.Spec.KubeconfigReference,
			},
		}

		if oldObject.Spec.CloudSpec != nil {
			newObject.Spec.CloudSpec = &newv1.ExternalClusterCloudSpec{
				GKE: (*newv1.ExternalClusterGKECloudSpec)(oldObject.Spec.CloudSpec.GKE),
				EKS: (*newv1.ExternalClusterEKSCloudSpec)(oldObject.Spec.CloudSpec.EKS),
				AKS: (*newv1.ExternalClusterAKSCloudSpec)(oldObject.Spec.CloudSpec.AKS),
			}
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating ExternalCluster…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneKubermaticSettingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.KubermaticSettingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.KubermaticSetting{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "KubermaticSetting",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SettingSpec{
				CustomLinks:                      newv1.CustomLinks{},
				CleanupOptions:                   newv1.CleanupOptions(oldObject.Spec.CleanupOptions),
				DefaultNodeCount:                 oldObject.Spec.DefaultNodeCount,
				DisplayDemoInfo:                  oldObject.Spec.DisplayDemoInfo,
				DisplayAPIDocs:                   oldObject.Spec.DisplayAPIDocs,
				DisplayTermsOfService:            oldObject.Spec.DisplayTermsOfService,
				EnableDashboard:                  oldObject.Spec.EnableDashboard,
				EnableOIDCKubeconfig:             oldObject.Spec.EnableOIDCKubeconfig,
				UserProjectsLimit:                oldObject.Spec.UserProjectsLimit,
				RestrictProjectCreation:          oldObject.Spec.RestrictProjectCreation,
				EnableExternalClusterImport:      oldObject.Spec.EnableExternalClusterImport,
				OpaOptions:                       newv1.OpaOptions(oldObject.Spec.OpaOptions),
				MlaOptions:                       newv1.MlaOptions(oldObject.Spec.MlaOptions),
				MlaAlertmanagerPrefix:            oldObject.Spec.MlaAlertmanagerPrefix,
				MlaGrafanaPrefix:                 oldObject.Spec.MlaGrafanaPrefix,
				MachineDeploymentVMResourceQuota: newv1.MachineDeploymentVMResourceQuota(oldObject.Spec.MachineDeploymentVMResourceQuota),
			},
		}

		for _, link := range oldObject.Spec.CustomLinks {
			newObject.Spec.CustomLinks = append(newObject.Spec.CustomLinks, newv1.CustomLink(link))
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating KubermaticSetting…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneMLAAdminSettingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.MLAAdminSettingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.MLAAdminSetting{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "MLAAdminSetting",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.MLAAdminSettingSpec{
				ClusterName: oldObject.Spec.ClusterName,
			},
		}

		if oldObject.Spec.MonitoringRateLimits != nil {
			newObject.Spec.MonitoringRateLimits = &newv1.MonitoringRateLimitSettings{
				IngestionRate:      oldObject.Spec.MonitoringRateLimits.IngestionRate,
				IngestionBurstSize: oldObject.Spec.MonitoringRateLimits.IngestionBurstSize,
				MaxSeriesPerMetric: oldObject.Spec.MonitoringRateLimits.MaxSeriesPerMetric,
				MaxSeriesTotal:     oldObject.Spec.MonitoringRateLimits.MaxSeriesTotal,
				QueryRate:          oldObject.Spec.MonitoringRateLimits.QueryRate,
				QueryBurstSize:     oldObject.Spec.MonitoringRateLimits.QueryBurstSize,
				MaxSamplesPerQuery: oldObject.Spec.MonitoringRateLimits.MaxSamplesPerQuery,
				MaxSeriesPerQuery:  oldObject.Spec.MonitoringRateLimits.MaxSeriesPerQuery,
			}
		}

		if oldObject.Spec.LoggingRateLimits != nil {
			newObject.Spec.LoggingRateLimits = &newv1.LoggingRateLimitSettings{
				IngestionRate:      oldObject.Spec.LoggingRateLimits.IngestionRate,
				IngestionBurstSize: oldObject.Spec.LoggingRateLimits.IngestionBurstSize,
				QueryRate:          oldObject.Spec.LoggingRateLimits.QueryRate,
				QueryBurstSize:     oldObject.Spec.LoggingRateLimits.QueryBurstSize,
			}
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating MLAAdminSetting…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func clonePresetResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.PresetList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Preset{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Preset",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.PresetSpec{
				Enabled:        oldObject.Spec.Enabled,
				RequiredEmails: oldObject.Spec.RequiredEmails,
			},
		}

		// Very old KKP versions supported the "RequiredEmailDomain" (singular) field and
		// would transparently translate this into the RequiredEmails (plural) field. This
		// CRD migration we're doing right now is _the_ perfect time to end the deprecation,
		// migrate once and for all and remove the RequiredEmailDomain field.
		if oldObject.Spec.RequiredEmailDomain != "" {
			if oldObject.Spec.RequiredEmails == nil {
				oldObject.Spec.RequiredEmails = []string{}
			}

			oldObject.Spec.RequiredEmails = append(oldObject.Spec.RequiredEmails, oldObject.Spec.RequiredEmailDomain)
		}

		oldSpec := oldObject.Spec

		if oldSpec.AWS != nil {
			newObject.Spec.AWS = &newv1.AWS{
				ProviderPreset:       newv1.ProviderPreset(oldSpec.AWS.ProviderPreset),
				AccessKeyID:          oldSpec.AWS.AccessKeyID,
				SecretAccessKey:      oldSpec.AWS.SecretAccessKey,
				AssumeRoleARN:        oldSpec.AWS.AssumeRoleARN,
				AssumeRoleExternalID: oldSpec.AWS.AssumeRoleExternalID,
				VPCID:                oldSpec.AWS.VPCID,
				RouteTableID:         oldSpec.AWS.RouteTableID,
				InstanceProfileName:  oldSpec.AWS.InstanceProfileName,
				SecurityGroupID:      oldSpec.AWS.SecurityGroupID,
				ControlPlaneRoleARN:  oldSpec.AWS.ControlPlaneRoleARN,
			}
		}

		if oldSpec.Alibaba != nil {
			newObject.Spec.Alibaba = &newv1.Alibaba{
				ProviderPreset:  newv1.ProviderPreset(oldSpec.Alibaba.ProviderPreset),
				AccessKeyID:     oldSpec.Alibaba.AccessKeyID,
				AccessKeySecret: oldSpec.Alibaba.AccessKeySecret,
			}
		}

		if oldSpec.Anexia != nil {
			newObject.Spec.Anexia = &newv1.Anexia{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Anexia.ProviderPreset),
				Token:          oldSpec.Anexia.Token,
			}
		}

		if oldSpec.Azure != nil {
			newObject.Spec.Azure = &newv1.Azure{
				ProviderPreset:    newv1.ProviderPreset(oldSpec.Azure.ProviderPreset),
				TenantID:          oldSpec.Azure.TenantID,
				SubscriptionID:    oldSpec.Azure.SubscriptionID,
				ClientID:          oldSpec.Azure.ClientID,
				ClientSecret:      oldSpec.Azure.ClientSecret,
				ResourceGroup:     oldSpec.Azure.ResourceGroup,
				VNetResourceGroup: oldSpec.Azure.VNetResourceGroup,
				VNetName:          oldSpec.Azure.VNetName,
				SubnetName:        oldSpec.Azure.SubnetName,
				RouteTableName:    oldSpec.Azure.RouteTableName,
				SecurityGroup:     oldSpec.Azure.SecurityGroup,
				LoadBalancerSKU:   convertAzureLoadBalancerSKU(oldSpec.Azure.LoadBalancerSKU),
			}
		}

		if oldSpec.Digitalocean != nil {
			newObject.Spec.Digitalocean = &newv1.Digitalocean{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Digitalocean.ProviderPreset),
				Token:          oldSpec.Digitalocean.Token,
			}
		}

		if oldSpec.Fake != nil {
			newObject.Spec.Fake = &newv1.Fake{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Fake.ProviderPreset),
				Token:          oldSpec.Fake.Token,
			}
		}

		if oldSpec.GCP != nil {
			newObject.Spec.GCP = &newv1.GCP{
				ProviderPreset: newv1.ProviderPreset(oldSpec.GCP.ProviderPreset),
				Network:        oldSpec.GCP.Network,
				Subnetwork:     oldSpec.GCP.Subnetwork,
				ServiceAccount: oldSpec.GCP.ServiceAccount,
			}
		}

		if oldSpec.Hetzner != nil {
			newObject.Spec.Hetzner = &newv1.Hetzner{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Hetzner.ProviderPreset),
				Token:          oldSpec.Hetzner.Token,
				Network:        oldSpec.Hetzner.Network,
			}
		}

		if oldSpec.Kubevirt != nil {
			newObject.Spec.Kubevirt = &newv1.Kubevirt{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Kubevirt.ProviderPreset),
				Kubeconfig:     oldSpec.Kubevirt.Kubeconfig,
			}
		}

		if oldSpec.Nutanix != nil {
			newObject.Spec.Nutanix = &newv1.Nutanix{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Nutanix.ProviderPreset),
				ProxyURL:       oldSpec.Nutanix.ProxyURL,
				Username:       oldSpec.Nutanix.Username,
				Password:       oldSpec.Nutanix.Password,
				ClusterName:    oldSpec.Nutanix.ClusterName,
				ProjectName:    oldSpec.Nutanix.ProjectName,
			}
		}

		if oldSpec.Openstack != nil {
			newObject.Spec.Openstack = &newv1.Openstack{
				ProviderPreset:              newv1.ProviderPreset(oldSpec.Openstack.ProviderPreset),
				UseToken:                    oldSpec.Openstack.UseToken,
				ApplicationCredentialID:     oldSpec.Openstack.ApplicationCredentialID,
				ApplicationCredentialSecret: oldSpec.Openstack.ApplicationCredentialSecret,
				Username:                    oldSpec.Openstack.Username,
				Password:                    oldSpec.Openstack.Password,
				Project:                     oldSpec.Openstack.GetProject(),
				ProjectID:                   oldSpec.Openstack.GetProjectId(),
				Domain:                      oldSpec.Openstack.Domain,
				Network:                     oldSpec.Openstack.Network,
				SecurityGroups:              oldSpec.Openstack.SecurityGroups,
				FloatingIPPool:              oldSpec.Openstack.FloatingIPPool,
				RouterID:                    oldSpec.Openstack.RouterID,
				SubnetID:                    oldSpec.Openstack.SubnetID,
			}
		}

		if oldSpec.Packet != nil {
			newObject.Spec.Packet = &newv1.Packet{
				ProviderPreset: newv1.ProviderPreset(oldSpec.Packet.ProviderPreset),
				APIKey:         oldSpec.Packet.APIKey,
				ProjectID:      oldSpec.Packet.ProjectID,
				BillingCycle:   oldSpec.Packet.BillingCycle,
			}
		}

		if oldSpec.VSphere != nil {
			newObject.Spec.VSphere = &newv1.VSphere{
				ProviderPreset:   newv1.ProviderPreset(oldSpec.VSphere.ProviderPreset),
				Username:         oldSpec.VSphere.Username,
				Password:         oldSpec.VSphere.Password,
				VMNetName:        oldSpec.VSphere.VMNetName,
				Datastore:        oldSpec.VSphere.Datastore,
				DatastoreCluster: oldSpec.VSphere.DatastoreCluster,
				ResourcePool:     oldSpec.VSphere.ResourcePool,
			}
		}

		if oldSpec.GKE != nil {
			newObject.Spec.GKE = &newv1.GKE{
				ProviderPreset: newv1.ProviderPreset(oldSpec.GKE.ProviderPreset),
				ServiceAccount: oldSpec.GKE.ServiceAccount,
			}
		}

		if oldSpec.EKS != nil {
			newObject.Spec.EKS = &newv1.EKS{
				ProviderPreset:  newv1.ProviderPreset(oldSpec.EKS.ProviderPreset),
				AccessKeyID:     oldSpec.EKS.AccessKeyID,
				SecretAccessKey: oldSpec.EKS.SecretAccessKey,
				Region:          oldSpec.EKS.Region,
			}
		}

		if oldSpec.AKS != nil {
			newObject.Spec.AKS = &newv1.AKS{
				ProviderPreset: newv1.ProviderPreset(oldSpec.AKS.ProviderPreset),
				TenantID:       oldSpec.AKS.TenantID,
				SubscriptionID: oldSpec.AKS.SubscriptionID,
				ClientID:       oldSpec.AKS.ClientID,
				ClientSecret:   oldSpec.AKS.ClientSecret,
			}
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating Preset…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneProjectResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ProjectList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Project objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Project",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ProjectSpec{
				Name: oldObject.Spec.Name,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating Project…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.ProjectStatus{
			Phase: newv1.ProjectPhase(oldObject.Status.Phase),
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneRuleGroupResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.RuleGroupList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list RuleGroup objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.RuleGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "RuleGroup",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.RuleGroupSpec{
				RuleGroupType: newv1.RuleGroupType(oldObject.Spec.RuleGroupType),
				Data:          oldObject.Spec.Data,
				Cluster:       migrateObjectReference(oldObject.Spec.Cluster, ""),
			},
		}

		logger.WithField("resource", ctrlruntimeclient.ObjectKeyFromObject(&oldObject)).Debug("Duplicating RuleGroup…")

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneSeedResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.SeedList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Seed objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Seed{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "Seed",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SeedSpec{
				Country:          oldObject.Spec.Country,
				Location:         oldObject.Spec.Location,
				Kubeconfig:       migrateObjectReference(oldObject.Spec.Kubeconfig, oldObject.Namespace),
				Datacenters:      map[string]newv1.Datacenter{},
				SeedDNSOverwrite: oldObject.Spec.SeedDNSOverwrite,
				NodeportProxy: newv1.NodeportProxyConfig{
					Disable:      oldObject.Spec.NodeportProxy.Disable,
					Annotations:  oldObject.Spec.NodeportProxy.Annotations,
					Envoy:        convertNodeportProxyComponentEnvoy(oldObject.Spec.NodeportProxy.Envoy),
					EnvoyManager: convertNodeportProxyComponent(oldObject.Spec.NodeportProxy.EnvoyManager),
					Updater:      convertNodeportProxyComponent(oldObject.Spec.NodeportProxy.Updater),
				},
				ExposeStrategy:           newv1.ExposeStrategy(oldObject.Spec.ExposeStrategy),
				DefaultComponentSettings: convertComponentSettings(oldObject.Spec.DefaultComponentSettings),
				DefaultClusterTemplate:   oldObject.Spec.DefaultClusterTemplate,
			},
		}

		for name, oldDC := range oldObject.Spec.Datacenters {
			newObject.Spec.Datacenters[name] = convertDatacenter(oldDC)
		}

		if oldObject.Spec.ProxySettings != nil {
			newObject.Spec.ProxySettings = &newv1.ProxySettings{
				HTTPProxy: (*newv1.ProxyValue)(oldObject.Spec.ProxySettings.HTTPProxy),
				NoProxy:   (*newv1.ProxyValue)(oldObject.Spec.ProxySettings.NoProxy),
			}
		}

		if oldObject.Spec.MLA != nil {
			newObject.Spec.MLA = &newv1.SeedMLASettings{
				UserClusterMLAEnabled: oldObject.Spec.MLA.UserClusterMLAEnabled,
			}
		}

		if oldObject.Spec.Metering != nil {
			newObject.Spec.Metering = &newv1.MeteringConfiguration{
				Enabled:          oldObject.Spec.Metering.Enabled,
				StorageClassName: oldObject.Spec.Metering.StorageClassName,
				StorageSize:      oldObject.Spec.Metering.StorageSize,
			}
		}

		if oldObject.Spec.EtcdBackupRestore != nil {
			// due to the preflight checks, this can only happen if someone edits
			// the Seed right after the preflight checks and before the cloning phase
			if oldObject.Spec.EtcdBackupRestore.DefaultDestination == nil {
				return 0, fmt.Errorf("a default destination must be set for the EtcdBackupRestore configuration in Seed %s", oldObject.Name)
			}

			destinations := make(map[string]*newv1.BackupDestination)
			for name, destination := range oldObject.Spec.EtcdBackupRestore.Destinations {
				destinations[name] = &newv1.BackupDestination{
					Endpoint:    destination.Endpoint,
					BucketName:  destination.BucketName,
					Credentials: destination.Credentials,
				}
			}
			newObject.Spec.EtcdBackupRestore = &newv1.EtcdBackupRestore{
				Destinations:       destinations,
				DefaultDestination: *oldObject.Spec.EtcdBackupRestore.DefaultDestination,
			}
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating Seed…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertComponentSettings(oldSettings kubermaticv1.ComponentSettings) newv1.ComponentSettings {
	newSettings := newv1.ComponentSettings{
		Apiserver: newv1.APIServerSettings{
			DeploymentSettings:          convertDeploymentSettings(oldSettings.Apiserver.DeploymentSettings),
			EndpointReconcilingDisabled: oldSettings.Apiserver.EndpointReconcilingDisabled,
			NodePortRange:               oldSettings.Apiserver.NodePortRange,
		},
		ControllerManager: convertControllerSettings(oldSettings.ControllerManager),
		Scheduler:         convertControllerSettings(oldSettings.Scheduler),
		Etcd: newv1.EtcdStatefulSetSettings{
			ClusterSize:  oldSettings.Etcd.ClusterSize,
			StorageClass: oldSettings.Etcd.StorageClass,
			DiskSize:     oldSettings.Etcd.DiskSize,
			Resources:    oldSettings.Etcd.Resources.DeepCopy(),
			Tolerations:  oldSettings.Etcd.Tolerations,
		},
		Prometheus: newv1.StatefulSetSettings{
			Resources: oldSettings.Prometheus.Resources.DeepCopy(),
		},
		NodePortProxyEnvoy: newv1.NodeportProxyComponent{
			DockerRepository: oldSettings.NodePortProxyEnvoy.DockerRepository,
			Resources:        oldSettings.NodePortProxyEnvoy.Resources,
		},
	}

	return newSettings
}

func convertControllerSettings(oldSettings kubermaticv1.ControllerSettings) newv1.ControllerSettings {
	return newv1.ControllerSettings{
		DeploymentSettings: convertDeploymentSettings(oldSettings.DeploymentSettings),
		LeaderElectionSettings: newv1.LeaderElectionSettings{
			LeaseDurationSeconds: oldSettings.LeaderElectionSettings.LeaseDurationSeconds,
			RenewDeadlineSeconds: oldSettings.LeaderElectionSettings.RenewDeadlineSeconds,
			RetryPeriodSeconds:   oldSettings.LeaderElectionSettings.RetryPeriodSeconds,
		},
	}
}

func convertDeploymentSettings(oldSettings kubermaticv1.DeploymentSettings) newv1.DeploymentSettings {
	return newv1.DeploymentSettings{
		Replicas:    oldSettings.Replicas,
		Resources:   oldSettings.Resources.DeepCopy(),
		Tolerations: oldSettings.Tolerations,
	}
}

func convertNodeportProxyComponentEnvoy(oldComponent kubermaticv1.NodeportProxyComponent) newv1.NodePortProxyComponentEnvoy {
	return newv1.NodePortProxyComponentEnvoy{
		NodeportProxyComponent: newv1.NodeportProxyComponent{
			DockerRepository: oldComponent.DockerRepository,
			Resources:        *oldComponent.Resources.DeepCopy(),
		},
	}
}

func convertNodeportProxyComponent(oldComponent kubermaticv1.NodeportProxyComponent) newv1.NodeportProxyComponent {
	return newv1.NodeportProxyComponent{
		DockerRepository: oldComponent.DockerRepository,
		Resources:        *oldComponent.Resources.DeepCopy(),
	}
}

func convertDatacenter(oldDC kubermaticv1.Datacenter) newv1.Datacenter {
	newDC := newv1.Datacenter{
		Country:  oldDC.Country,
		Location: oldDC.Location,
		Spec: newv1.DatacenterSpec{
			EnforceAuditLogging:            oldDC.Spec.EnforceAuditLogging,
			EnforcePodSecurityPolicy:       oldDC.Spec.EnforcePodSecurityPolicy,
			RequiredEmails:                 oldDC.Spec.RequiredEmailDomains,
			ProviderReconciliationInterval: oldDC.Spec.ProviderReconciliationInterval,
		},
	}

	// migrate from the deprecated flag to finally get rid of RequiredEmailDomain
	if oldDC.Spec.RequiredEmailDomain != "" {
		if newDC.Spec.RequiredEmails == nil {
			newDC.Spec.RequiredEmails = []string{}
		}

		newDC.Spec.RequiredEmails = append(newDC.Spec.RequiredEmails, oldDC.Spec.RequiredEmailDomain)
	}

	if oldDC.Node != nil {
		newDC.Node = &newv1.NodeSettings{
			ProxySettings: newv1.ProxySettings{
				HTTPProxy: (*newv1.ProxyValue)(oldDC.Node.HTTPProxy),
				NoProxy:   (*newv1.ProxyValue)(oldDC.Node.NoProxy),
			},
			InsecureRegistries: oldDC.Node.InsecureRegistries,
			RegistryMirrors:    oldDC.Node.RegistryMirrors,
			PauseImage:         oldDC.Node.PauseImage,
		}
	}

	oldSpec := oldDC.Spec

	if oldSpec.AWS != nil {
		newDC.Spec.AWS = &newv1.DatacenterSpecAWS{
			Region: oldSpec.AWS.Region,
			Images: newv1.ImageList(oldSpec.AWS.Images),
		}
	}

	if oldSpec.Alibaba != nil {
		newDC.Spec.Alibaba = &newv1.DatacenterSpecAlibaba{
			Region: oldSpec.Alibaba.Region,
		}
	}

	if oldSpec.Anexia != nil {
		newDC.Spec.Anexia = &newv1.DatacenterSpecAnexia{
			LocationID: oldSpec.Anexia.LocationID,
		}
	}

	if oldSpec.Azure != nil {
		newDC.Spec.Azure = &newv1.DatacenterSpecAzure{
			Location: oldSpec.Azure.Location,
		}
	}

	if oldSpec.Digitalocean != nil {
		newDC.Spec.Digitalocean = &newv1.DatacenterSpecDigitalocean{
			Region: oldSpec.Digitalocean.Region,
		}
	}

	if oldSpec.BringYourOwn != nil {
		newDC.Spec.BringYourOwn = &newv1.DatacenterSpecBringYourOwn{}
	}

	if oldSpec.Fake != nil {
		newDC.Spec.Fake = &newv1.DatacenterSpecFake{
			FakeProperty: oldSpec.Fake.FakeProperty,
		}
	}

	if oldSpec.GCP != nil {
		newDC.Spec.GCP = &newv1.DatacenterSpecGCP{
			Region:       oldSpec.GCP.Region,
			ZoneSuffixes: oldSpec.GCP.ZoneSuffixes,
			Regional:     oldSpec.GCP.Regional,
		}
	}

	if oldSpec.Hetzner != nil {
		newDC.Spec.Hetzner = &newv1.DatacenterSpecHetzner{
			Datacenter: oldSpec.Hetzner.Datacenter,
			Location:   oldSpec.Hetzner.Location,
			Network:    oldSpec.Hetzner.Network,
		}
	}

	if oldSpec.Kubevirt != nil {
		newDC.Spec.Kubevirt = &newv1.DatacenterSpecKubevirt{
			DNSPolicy: oldSpec.Kubevirt.DNSPolicy,
			DNSConfig: oldSpec.Kubevirt.DNSConfig.DeepCopy(),
		}
	}

	if oldSpec.Nutanix != nil {
		newDC.Spec.Nutanix = &newv1.DatacenterSpecNutanix{
			Endpoint:      oldSpec.Nutanix.Endpoint,
			Port:          oldSpec.Nutanix.Port,
			AllowInsecure: oldSpec.Nutanix.AllowInsecure,
			Images:        newv1.ImageList(oldSpec.Nutanix.Images),
		}
	}

	if oldSpec.Openstack != nil {
		newDC.Spec.Openstack = &newv1.DatacenterSpecOpenstack{
			AuthURL:              oldSpec.Openstack.AuthURL,
			AvailabilityZone:     oldSpec.Openstack.AvailabilityZone,
			Region:               oldSpec.Openstack.Region,
			IgnoreVolumeAZ:       oldSpec.Openstack.IgnoreVolumeAZ,
			EnforceFloatingIP:    oldSpec.Openstack.EnforceFloatingIP,
			DNSServers:           oldSpec.Openstack.DNSServers,
			Images:               newv1.ImageList(oldSpec.Openstack.Images),
			ManageSecurityGroups: oldSpec.Openstack.ManageSecurityGroups,
			UseOctavia:           oldSpec.Openstack.UseOctavia,
			TrustDevicePath:      oldSpec.Openstack.TrustDevicePath,
			NodeSizeRequirements: newv1.OpenstackNodeSizeRequirements{
				MinimumVCPUs:  oldSpec.Openstack.NodeSizeRequirements.MinimumVCPUs,
				MinimumMemory: oldSpec.Openstack.NodeSizeRequirements.MinimumMemory,
			},
			EnabledFlavors: oldSpec.Openstack.EnabledFlavors,
		}
	}

	if oldSpec.Packet != nil {
		newDC.Spec.Packet = &newv1.DatacenterSpecPacket{
			Facilities: oldSpec.Packet.Facilities,
		}
	}

	if oldSpec.VSphere != nil {
		newDC.Spec.VSphere = &newv1.DatacenterSpecVSphere{
			Endpoint:             oldSpec.VSphere.Endpoint,
			AllowInsecure:        oldSpec.VSphere.AllowInsecure,
			DefaultDatastore:     oldSpec.VSphere.DefaultDatastore,
			Datacenter:           oldSpec.VSphere.Datacenter,
			Cluster:              oldSpec.VSphere.Cluster,
			DefaultStoragePolicy: oldSpec.VSphere.DefaultStoragePolicy,
			RootPath:             oldSpec.VSphere.RootPath,
			Templates:            newv1.ImageList(oldSpec.VSphere.Templates),
		}

		if iu := oldSpec.VSphere.InfraManagementUser; iu != nil {
			newDC.Spec.VSphere.InfraManagementUser = &newv1.VSphereCredentials{
				Username: iu.Username,
				Password: iu.Password,
			}
		}
	}

	return newDC
}

func cloneUserResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list User objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.User{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "User",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.UserSpec{
				ID:                     oldObject.Spec.ID,
				Name:                   oldObject.Spec.Name,
				Email:                  oldObject.Spec.Email,
				IsAdmin:                oldObject.Spec.IsAdmin,
				InvalidTokensReference: oldObject.Spec.TokenBlackListReference,
			},
		}

		if s := oldObject.Spec.Settings; s != nil {
			newObject.Spec.Settings = &newv1.UserSettings{
				SelectedTheme:              s.SelectedTheme,
				ItemsPerPage:               s.ItemsPerPage,
				SelectedProjectID:          s.SelectedProjectID,
				SelectProjectTableView:     s.SelectProjectTableView,
				CollapseSidenav:            s.CollapseSidenav,
				DisplayAllProjectsForAdmin: s.DisplayAllProjectsForAdmin,
				LastSeenChangelogVersion:   s.LastSeenChangelogVersion,
			}
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating User…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		newObject.Status = newv1.UserStatus{}
		if oldObject.Status.LastSeen != nil {
			newObject.Status.LastSeen = *oldObject.Status.LastSeen
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneUserProjectBindingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserProjectBindingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list UserProjectBinding objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.UserProjectBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "UserProjectBinding",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.UserProjectBindingSpec{
				UserEmail: oldObject.Spec.UserEmail,
				ProjectID: oldObject.Spec.ProjectID,
				Group:     oldObject.Spec.Group,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating UserProjectBinding…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneUserSSHKeyResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserSSHKeyList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list UserSSHKey objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.UserSSHKey{
			TypeMeta: metav1.TypeMeta{
				APIVersion: newv1.SchemeGroupVersion.String(),
				Kind:       "UserSSHKey",
			},
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SSHKeySpec{
				Owner:       oldObject.Spec.Owner,
				Name:        oldObject.Spec.Name,
				Fingerprint: oldObject.Spec.Fingerprint,
				PublicKey:   oldObject.Spec.PublicKey,
				Clusters:    oldObject.Spec.Clusters,
			},
		}

		logger.WithField("resource", oldObject.GetName()).Debug("Duplicating UserSSHKey…")

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}
