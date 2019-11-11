package common

import (
	"context"
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// OperatorName is used as the value for ManagedBy labels to establish
	// a weak ownership to reconciled resources.
	OperatorName = "kubermatic-operator"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// ConfigurationOwnerAnnotation is the annotation containing a resource's
	// owning configuration name and namespace.
	ConfigurationOwnerAnnotation = "operator.kubermatic.io/configuration"
)

func StringifyFeatureGates(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	features := make([]string, 0)
	for _, feature := range cfg.Spec.FeatureGates.List() {
		features = append(features, fmt.Sprintf("%s=true", feature))
	}

	return strings.Join(features, ",")
}

// OwnerLabelsModifierFactory is generating a new ObjectModifier that wraps an ObjectCreator
// and takes care of applying the default labels and annotations from this operator.
// These are then used to establish a weak ownership.
func OwnerLabelsModifierFactory(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			o, ok := obj.(metav1.Object)
			if !ok {
				return obj, nil
			}

			annotations := o.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			identifier, err := cache.MetaNamespaceKeyFunc(cfg)
			if err != nil {
				return obj, fmt.Errorf("failed to determine KubermaticConfiguration string key: %v", err)
			}

			annotations[ConfigurationOwnerAnnotation] = identifier
			o.SetAnnotations(annotations)

			labels := o.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[ManagedByLabel] = OperatorName
			o.SetLabels(labels)

			return obj, nil
		}
	}
}

// VolumeRevisionLabelsModifierFactory scans volume mounts for pod templates for ConfigMaps
// and Secrets and will then put new labels for these mounts onto the pod template, causing
// restarts when the volumes changed.
func VolumeRevisionLabelsModifierFactory(ctx context.Context, client ctrlruntimeclient.Client) reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				return obj, nil
			}

			volumeLabels, err := resources.VolumeRevisionLabels(ctx, client, deployment.Namespace, deployment.Spec.Template.Spec.Volumes)
			if err != nil {
				return obj, fmt.Errorf("failed to determine revision labels for volumes: %v", err)
			}

			// switch to a new map in case the deployment used the same map for selector.matchLabels and labels
			oldLabels := deployment.Spec.Template.Labels
			deployment.Spec.Template.Labels = volumeLabels

			for k, v := range oldLabels {
				deployment.Spec.Template.Labels[k] = v
			}

			return obj, nil
		}
	}
}

func createSecretData(s *corev1.Secret, data map[string]string) *corev1.Secret {
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}

	for k, v := range data {
		s.Data[k] = []byte(v)
	}

	return s
}
