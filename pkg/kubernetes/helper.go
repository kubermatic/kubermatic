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

package kubernetes

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/sdk/v2/semver"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/machine-controller/sdk/providerconfig"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// RevisionAnnotation is the revision annotation of a deployment's replica sets which records its rollout sequence.
	RevisionAnnotation = "deployment.kubernetes.io/revision"
	// NodeControlPlaneLabel is the label on kubernetes control plane nodes.
	NodeControlPlaneLabel = "node-role.kubernetes.io/control-plane"
)

var tokenValidator = regexp.MustCompile(`[bcdfghjklmnpqrstvwxz2456789]{6}\.[bcdfghjklmnpqrstvwxz2456789]{16}`)

// HasFinalizer tells if a object has all the given finalizers.
func HasFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).HasAll(names...)
}

func HasAnyFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).HasAny(names...)
}

// HasOnlyFinalizer tells if an object has only the given finalizer(s).
func HasOnlyFinalizer(o metav1.Object, names ...string) bool {
	return sets.New(o.GetFinalizers()...).Equal(sets.New(names...))
}

// HasFinalizerSuperset tells if the given finalizer(s) are a superset
// of the actual finalizers.
func HasFinalizerSuperset(o metav1.Object, names ...string) bool {
	return sets.New(names...).IsSuperset(sets.New(o.GetFinalizers()...))
}

// RemoveFinalizer removes the given finalizers from the object.
func RemoveFinalizer(obj metav1.Object, toRemove ...string) {
	set := sets.New(obj.GetFinalizers()...)
	set.Delete(toRemove...)
	obj.SetFinalizers(sets.List(set))
}

func TryRemoveFinalizer(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, finalizers ...string) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(obj)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the object
		if err := client.Get(ctx, key, obj); err != nil {
			// finalizer removal normally happens during object cleanup, so if
			// the object is gone already, that is absolutely fine
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		original := obj.DeepCopyObject().(ctrlruntimeclient.Object)

		// modify it
		previous := sets.New(obj.GetFinalizers()...)
		RemoveFinalizer(obj, finalizers...)
		current := sets.New(obj.GetFinalizers()...)

		// save some work
		if previous.Equal(current) {
			return nil
		}

		// update the object
		return client.Patch(ctx, obj, ctrlruntimeclient.MergeFromWithOptions(original, ctrlruntimeclient.MergeFromWithOptimisticLock{}))
	})

	if err != nil {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to remove finalizers %v from %s %s: %w", finalizers, kind, key, err)
	}

	return nil
}

// AddFinalizer will add the given finalizer to the object. It uses a StringSet to avoid duplicates.
func AddFinalizer(obj metav1.Object, finalizers ...string) {
	set := sets.New(obj.GetFinalizers()...)
	set.Insert(finalizers...)
	obj.SetFinalizers(sets.List(set))
}

func TryAddFinalizer(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, finalizers ...string) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(obj)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the object
		if err := client.Get(ctx, key, obj); err != nil {
			return err
		}

		// cannot add new finalizers to deleted objects
		if obj.GetDeletionTimestamp() != nil {
			return nil
		}

		original := obj.DeepCopyObject().(ctrlruntimeclient.Object)

		// modify it
		previous := sets.New(obj.GetFinalizers()...)
		AddFinalizer(obj, finalizers...)
		current := sets.New(obj.GetFinalizers()...)

		// save some work
		if previous.Equal(current) {
			return nil
		}

		// update the object
		return client.Patch(ctx, obj, ctrlruntimeclient.MergeFromWithOptions(original, ctrlruntimeclient.MergeFromWithOptimisticLock{}))
	})

	if err != nil {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to add finalizers %v to %s %s: %w", finalizers, kind, key, err)
	}

	return nil
}

// GenerateToken generates a new, random token that can be used
// as an admin and kubelet token.
func GenerateToken() string {
	return fmt.Sprintf("%s.%s", rand.String(6), rand.String(16))
}

// ValidateKubernetesToken checks if a given token is syntactically correct.
func ValidateKubernetesToken(token string) error {
	if !tokenValidator.MatchString(token) {
		return fmt.Errorf("token is malformed, must match %s", tokenValidator.String())
	}

	return nil
}

func ValidateSecretKeySelector(selector *providerconfig.GlobalSecretKeySelector, key string) error {
	if selector == nil || selector.Name == "" || selector.Namespace == "" || key == "" {
		return fmt.Errorf("%q cannot be empty", key)
	}
	return nil
}

// IsDeploymentRolloutComplete returns a bool saying whether the deployment completed and
// an error in case an unexpected condition arrives.
//
// based on:
// https://github.com/kubernetes/kubernetes/blob/252887e39f905389156d2bc9c5932688857588e4/staging/src/k8s.io/kubectl/pkg/polymorphichelpers/rollout_status.go#L59
func IsDeploymentRolloutComplete(deployment *appsv1.Deployment, revision int64) (bool, error) {
	if revision > 0 {
		deploymentRev, err := Revision(deployment)
		if err != nil {
			return false, fmt.Errorf("cannot get the revision of deployment %q: %w", deployment.Name, err)
		}
		if revision != deploymentRev {
			return false, fmt.Errorf("desired revision (%d) is different from the running revision (%d)", revision, deploymentRev)
		}
	}

	if deployment.Generation <= deployment.Status.ObservedGeneration {
		cond := GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
		if cond != nil && cond.Reason == "ProgressDeadlineExceeded" {
			return false, fmt.Errorf("deployment %q exceeded its progress deadline", deployment.Name)
		}

		desiredReplicas := int32(1)
		if deployment.Spec.Replicas != nil {
			desiredReplicas = *deployment.Spec.Replicas
		}

		logger := kubermaticlog.Logger.With(
			"deployment", deployment.Name,
			"desired", desiredReplicas,
			"updated", deployment.Status.UpdatedReplicas,
			"available", deployment.Status.AvailableReplicas,
		)

		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			logger.Debug("deployment rollout did not complete: not all replicas have been updated")
			return false, nil
		}
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			logger.Debugw("deployment rollout did not complete: old replicas are pending termination", "pending", deployment.Status.Replicas-deployment.Status.UpdatedReplicas)
			return false, nil
		}
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			logger.Debug("deployment rollout did not complete: not enough updated replicas available")
			return false, nil
		}

		return true, nil
	}

	return false, nil
}

// Revision returns the revision number of the input object.
func Revision(obj runtime.Object) (int64, error) {
	acc, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	v, ok := acc.GetAnnotations()[RevisionAnnotation]
	if !ok {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// GetDeploymentCondition returns the condition with the provided type.
func GetDeploymentCondition(status appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func ContainerFromString(containerSpec string) (*corev1.Container, error) {
	if len(strings.TrimSpace(containerSpec)) == 0 {
		return nil, nil
	}

	container := &corev1.Container{}
	manifestDecoder := yaml.NewYAMLToJSONDecoder(strings.NewReader(containerSpec))
	if err := manifestDecoder.Decode(container); err != nil {
		return nil, err
	}

	// Just because it's a valid corev1.Container does not mean
	// the APIServer will accept it, thus we do some additional
	// checks
	if container.Name == "" {
		return nil, fmt.Errorf("container must have a name")
	}
	if container.Image == "" {
		return nil, fmt.Errorf("container must have an image")
	}

	return container, nil
}

func SortOwnerReferences(refs []metav1.OwnerReference) {
	sort.Slice(refs, func(i, j int) bool {
		refA := refs[i]
		refB := refs[j]

		if refA.APIVersion != refB.APIVersion {
			return refA.APIVersion < refB.APIVersion
		}

		if refA.Kind != refB.Kind {
			return refA.Kind < refB.Kind
		}

		return refA.Name < refB.Name
	})
}

func HasOwnerReference(o metav1.Object, ref metav1.OwnerReference) bool {
	for _, r := range o.GetOwnerReferences() {
		if equalOwnerRefs(r, ref) {
			return true
		}
	}

	return false
}

// RemoveOwnerReference removes any reference that has the same
// APIVersion, Kind and Name.
func RemoveOwnerReferences(o metav1.Object, refToRemoves ...metav1.OwnerReference) {
	removeOwnerReference(o, equalOwnerRefs, refToRemoves...)
}

// RemoveOwnerReferenceKinds removes any reference with the same
// APIVersion and Kind, notably ignoring the name.
func RemoveOwnerReferenceKinds(o metav1.Object, refKindsToRemove ...metav1.OwnerReference) {
	removeOwnerReference(o, equalOwnerRefKinds, refKindsToRemove...)
}

func equalOwnerRefKinds(a, b metav1.OwnerReference) bool {
	return a.APIVersion == b.APIVersion && a.Kind == b.Kind
}

func equalOwnerRefs(a, b metav1.OwnerReference) bool {
	return equalOwnerRefKinds(a, b) && a.Name == b.Name
}

func removeOwnerReference(o metav1.Object, comparator func(a, b metav1.OwnerReference) bool, refs ...metav1.OwnerReference) {
	newRefs := []metav1.OwnerReference{}

	for _, r := range o.GetOwnerReferences() {
		valid := true
		for _, toRemove := range refs {
			if comparator(r, toRemove) {
				valid = false
				break
			}
		}

		if valid {
			newRefs = append(newRefs, r)
		}
	}

	o.SetOwnerReferences(newRefs)
}

// EnsureOwnerReference will add the given owner reference to the
// object if it doesn't exist yet. Other references with the same
// name can exist.
func EnsureOwnerReference(o metav1.Object, ref metav1.OwnerReference) {
	RemoveOwnerReferences(o, ref)

	refs := o.GetOwnerReferences()
	refs = append(refs, ref)
	o.SetOwnerReferences(refs)
}

// EnsureUniqueOwnerReference will remove any owner ref with the same
// APIVersion and Kind, and then add the given ref to the owner references.
// This ensures that only one ref with a given kind exists.
func EnsureUniqueOwnerReference(o metav1.Object, ref metav1.OwnerReference) {
	RemoveOwnerReferenceKinds(o, ref)

	refs := o.GetOwnerReferences()
	refs = append(refs, ref)
	o.SetOwnerReferences(refs)
}

func EnsureLabels(o metav1.Object, toEnsure map[string]string) {
	labels := maps.Clone(o.GetLabels())

	if labels == nil {
		labels = make(map[string]string)
	}
	for key, value := range toEnsure {
		labels[key] = value
	}
	o.SetLabels(labels)
}

func EnsureAnnotations(o metav1.Object, toEnsure map[string]string) {
	annotations := maps.Clone(o.GetAnnotations())

	if annotations == nil {
		annotations = make(map[string]string)
	}
	for key, value := range toEnsure {
		annotations[key] = value
	}
	o.SetAnnotations(annotations)
}

func EnsureAnnotationContains(o metav1.Object, annotation string, separator string, toEnsure ...string) {
	values := sets.New[string]()

	existing, ok := o.GetAnnotations()[annotation]
	if ok {
		parts := strings.Split(existing, separator)
		values.Insert(parts...)
	}

	values.Insert(toEnsure...)

	EnsureAnnotations(o, map[string]string{
		annotation: strings.Join(sets.List(values), separator),
	})
}

// HasAnnotationTrue checks whether the given object has the specified annotation
// set to "true" (case-insensitive).
func HasAnnotationTrue(obj metav1.Object, key string) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	val, exists := annotations[key]
	return exists && strings.EqualFold(val, "true")
}

type SeedClientMap map[string]ctrlruntimeclient.Client

type SeedVisitorFunc func(seedName string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error

func (m SeedClientMap) Each(ctx context.Context, log *zap.SugaredLogger, visitor SeedVisitorFunc) error {
	for seedName, seedClient := range m {
		err := visitor(seedName, seedClient, log.With("seed", seedName))
		if err != nil {
			return fmt.Errorf("failed processing Seed %s: %w", seedName, err)
		}
	}

	return nil
}

// IsNodeReady returns true if a node is ready; false otherwise.
func IsNodeReady(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func GetNodeGroupReadyCount(nodes *corev1.NodeList, providerNodeLabel, providerNodePoolName string) int32 {
	var readyReplicasCount int32
	for _, node := range nodes.Items {
		if node.Labels[providerNodeLabel] == providerNodePoolName && IsNodeReady(&node) {
			readyReplicasCount++
		}
	}

	return readyReplicasCount
}

func GetVersion(client *kubernetes.Clientset) (*ksemver.Semver, error) {
	version, err := client.ServerVersion()
	if err != nil {
		return nil, err
	}
	v, err := ksemver.NewSemver(version.GitVersion)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func GetClusterClient(ctx context.Context, cluster *kubermaticv1.ExternalCluster, masterClient ctrlruntimeclient.Client) (*kubernetes.Clientset, error) {
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, masterClient)
	rawKubeconfig, err := secretKeyGetter(cluster.Spec.KubeconfigReference, "kubeconfig")
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.Load([]byte(rawKubeconfig))
	if err != nil {
		return nil, err
	}
	clientConfig, err := getRestConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getRestConfig(cfg *clientcmdapi.Config) (*rest.Config, error) {
	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		"",
		&clientcmd.ConfigOverrides{},
		nil,
	)

	return iconfig.ClientConfig()
}

func GetContainerRuntime(ctx context.Context,
	clusterClient *kubernetes.Clientset,
) (string, error) {
	nodeReq, err := labels.NewRequirement(NodeControlPlaneLabel, selection.Exists, []string{})
	if err != nil {
		return "", fmt.Errorf("error creating selector requirement: %w", err)
	}
	selector := labels.NewSelector().Add(*nodeReq)

	controlPlaneNode, err := clusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		Limit:         1,
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", err
	}

	for len(controlPlaneNode.Items) > 0 {
		containerRuntimeVersion := controlPlaneNode.Items[0].Status.NodeInfo.ContainerRuntimeVersion
		containerRuntime, _, found := strings.Cut(containerRuntimeVersion, ":")
		if found {
			return containerRuntime, nil
		}
	}

	return "", fmt.Errorf("failed to fetch container runtime: no control plane nodes found with label %s", NodeControlPlaneLabel)
}

func ExternalClusterPausedChecker(ctx context.Context, externalClusterName string, masterClient ctrlruntimeclient.Client) (bool, error) {
	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := masterClient.Get(ctx, types.NamespacedName{Name: externalClusterName}, externalCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get external cluster %q: %w", externalClusterName, err)
	}

	return externalCluster.Spec.Pause, nil
}
