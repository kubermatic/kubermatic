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
	"fmt"
	"regexp"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

const (
	// RevisionAnnotation is the revision annotation of a deployment's replica sets which records its rollout sequence
	RevisionAnnotation = "deployment.kubernetes.io/revision"
)

var tokenValidator = regexp.MustCompile(`[bcdfghjklmnpqrstvwxz2456789]{6}\.[bcdfghjklmnpqrstvwxz2456789]{16}`)

// HasFinalizer tells if a object has all the given finalizers
func HasFinalizer(o metav1.Object, names ...string) bool {
	return sets.NewString(o.GetFinalizers()...).HasAll(names...)
}

func HasAnyFinalizer(o metav1.Object, names ...string) bool {
	return sets.NewString(o.GetFinalizers()...).HasAny(names...)
}

// HasOnlyFinalizer tells if an object has only the given finalizer
func HasOnlyFinalizer(o metav1.Object, name string) bool {
	set := sets.NewString(o.GetFinalizers()...)
	return set.Has(name) && set.Len() == 1
}

// RemoveFinalizer removes the given finalizers from the object
func RemoveFinalizer(obj metav1.Object, toRemove ...string) {
	set := sets.NewString(obj.GetFinalizers()...)
	set.Delete(toRemove...)
	obj.SetFinalizers(set.List())
}

// AddFinalizer will add the given finalizer to the object. It uses a StringSet to avoid duplicates
func AddFinalizer(obj metav1.Object, finalizers ...string) {
	set := sets.NewString(obj.GetFinalizers()...)
	set.Insert(finalizers...)
	obj.SetFinalizers(set.List())
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
	if selector.Name == "" || selector.Namespace == "" || key == "" {
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
			return false, fmt.Errorf("cannot get the revision of deployment %q: %v", deployment.Name, err)
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
		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			klog.Infof("Deployment %q rollout did not complete: %d out of %d new replicas have been updated...\n", deployment.Name, deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas)
			return false, nil
		}
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			klog.Infof("Deployment %q rollout did not complete: %d old replicas are pending termination...\n", deployment.Name, deployment.Status.Replicas-deployment.Status.UpdatedReplicas)
			return false, nil
		}
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			klog.Infof("Deployment %q rollout did not complete: %d of %d updated replicas are available...\n", deployment.Name, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas)
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
