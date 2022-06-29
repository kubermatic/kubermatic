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

package reconciling

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// OwnerRefWrapper is responsible for wrapping a ObjectCreator function, solely to set the OwnerReference to the cluster object.
func OwnerRefWrapper(ref metav1.OwnerReference) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			obj.(metav1.Object).SetOwnerReferences([]metav1.OwnerReference{ref})
			return obj, nil
		}
	}
}

// ImagePullSecretsWrapper is generating a new ObjectModifier that wraps an ObjectCreator
// and takes care of adding the secret names provided to the ImagePullSecrets.
//
// TODO At the moment only Deployments are supported, but
// this can be extended to whatever Object carrying a PodSpec.
func ImagePullSecretsWrapper(secretNames ...string) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}
			if len(secretNames) == 0 {
				return obj, nil
			}
			switch o := obj.(type) {
			case *appsv1.Deployment:
				configureImagePullSecrets(&o.Spec.Template.Spec, secretNames)
				return o, nil
			default:
				return o, fmt.Errorf(`type %q is not supported by ImagePullSecretModifier`, o.GetObjectKind().GroupVersionKind())
			}
		}
	}
}

func configureImagePullSecrets(podSpec *corev1.PodSpec, secretNames []string) {
	// Only configure image pull secrets when provided in the configuration.
	currentSecretNames := sets.NewString()
	for _, ips := range podSpec.ImagePullSecrets {
		currentSecretNames.Insert(ips.Name)
	}
	for _, s := range secretNames {
		if !currentSecretNames.Has(s) {
			podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{Name: s})
		}
	}
}

// DefaultContainer defaults all Container attributes to the same values as they would get from the Kubernetes API.
func DefaultContainer(c *corev1.Container, procMountType *corev1.ProcMountType) {
	if c.ImagePullPolicy == "" {
		c.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if c.TerminationMessagePath == "" {
		c.TerminationMessagePath = corev1.TerminationMessagePathDefault
	}
	if c.TerminationMessagePolicy == "" {
		c.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	}

	for idx := range c.Env {
		if c.Env[idx].ValueFrom != nil && c.Env[idx].ValueFrom.FieldRef != nil {
			if c.Env[idx].ValueFrom.FieldRef.APIVersion == "" {
				c.Env[idx].ValueFrom.FieldRef.APIVersion = "v1"
			}
		}
	}

	// This attribute was added in 1.12
	if c.SecurityContext != nil {
		c.SecurityContext.ProcMount = procMountType
	}
}

// DefaultPodSpec defaults all Container attributes to the same values as they would get from the Kubernetes API.
// In addition, it sets default PodSpec values that KKP requires in all workloads, for example appropriate security settings.
// The following KKP-specific defaults are applied:
// - SecurityContext.SeccompProfile is set to be of type `RuntimeDefault` to enable seccomp isolation if not set.
func DefaultPodSpec(oldPodSpec, newPodSpec corev1.PodSpec) (corev1.PodSpec, error) {
	// make sure to keep the old procmount types in case a creator overrides the entire PodSpec
	initContainerProcMountType := map[string]*corev1.ProcMountType{}
	containerProcMountType := map[string]*corev1.ProcMountType{}
	for _, container := range oldPodSpec.InitContainers {
		if container.SecurityContext != nil {
			initContainerProcMountType[container.Name] = container.SecurityContext.ProcMount
		}
	}
	for _, container := range oldPodSpec.Containers {
		if container.SecurityContext != nil {
			containerProcMountType[container.Name] = container.SecurityContext.ProcMount
		}
	}

	for idx, container := range newPodSpec.InitContainers {
		DefaultContainer(&newPodSpec.InitContainers[idx], initContainerProcMountType[container.Name])
	}

	for idx, container := range newPodSpec.Containers {
		DefaultContainer(&newPodSpec.Containers[idx], containerProcMountType[container.Name])
	}

	for idx, vol := range newPodSpec.Volumes {
		if vol.VolumeSource.Secret != nil && vol.VolumeSource.Secret.DefaultMode == nil {
			newPodSpec.Volumes[idx].Secret.DefaultMode = utilpointer.Int32Ptr(corev1.SecretVolumeSourceDefaultMode)
		}
		if vol.VolumeSource.ConfigMap != nil && vol.VolumeSource.ConfigMap.DefaultMode == nil {
			newPodSpec.Volumes[idx].ConfigMap.DefaultMode = utilpointer.Int32Ptr(corev1.ConfigMapVolumeSourceDefaultMode)
		}
	}

	// set KKP specific defaults for every Pod created by it

	if newPodSpec.SecurityContext == nil {
		newPodSpec.SecurityContext = &corev1.PodSecurityContext{}
	}

	if newPodSpec.SecurityContext.SeccompProfile == nil {
		newPodSpec.SecurityContext.SeccompProfile = &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		}
	}

	return newPodSpec, nil
}

// DefaultDeployment defaults all Deployment attributes to the same values as they would get from the Kubernetes API.
// In addition, the Deployment's PodSpec template gets defaulted with KKP-specific values (see DefaultPodSpec for details).
func DefaultDeployment(oldObj, newObj *appsv1.Deployment) (ctrlruntimeclient.Object, error) {
	var err error

	if newObj.Spec.Strategy.Type == "" {
		newObj.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType

		if newObj.Spec.Strategy.RollingUpdate == nil {
			newObj.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 0,
				},
			}
		}
	}

	newObj.Spec.Template.Spec, err = DefaultPodSpec(oldObj.Spec.Template.Spec, newObj.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	return newObj, nil
}

// DefaultStatefulSet defaults all StatefulSet attributes to the same values as they would get from the Kubernetes API.
// In addition, the StatefulSet's PodSpec template gets defaulted with KKP-specific values (see DefaultPodSpec for details).
func DefaultStatefulSet(oldObj, newObj *appsv1.StatefulSet) (ctrlruntimeclient.Object, error) {
	var err error

	newObj.Spec.Template.Spec, err = DefaultPodSpec(oldObj.Spec.Template.Spec, newObj.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	return newObj, nil
}

// DefaultDaemonSet defaults all DaemonSet attributes to the same values as they would get from the Kubernetes API.
// In addition, the DaemonSet's PodSpec template gets defaulted with KKP-specific values (see DefaultPodSpec for details).
func DefaultDaemonSet(oldObj, newObj *appsv1.DaemonSet) (ctrlruntimeclient.Object, error) {
	var err error

	newObj.Spec.Template.Spec, err = DefaultPodSpec(oldObj.Spec.Template.Spec, newObj.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	return newObj, nil
}

// DefaultCronJob defaults all CronJob attributes to the same values as they would get from the Kubernetes API.
// In addition, the CronJob's PodSpec template gets defaulted with KKP-specific values (see DefaultPodSpec for details).
func DefaultCronJob(oldObj, newObj *batchv1beta1.CronJob) (ctrlruntimeclient.Object, error) {
	var err error

	newObj.Spec.JobTemplate.Spec.Template.Spec, err = DefaultPodSpec(oldObj.Spec.JobTemplate.Spec.Template.Spec, newObj.Spec.JobTemplate.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	return newObj, nil
}
