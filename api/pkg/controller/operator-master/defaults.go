package operatormaster

import (
	"fmt"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

func (r *Reconciler) defaultConfiguration(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) (bool, error) {
	logger.Debug("Applying defaults to Kubermatic configuration")

	original := config.DeepCopy()

	if config.Spec.Namespace == "" {
		config.Spec.Namespace = config.Namespace
		logger.Debugf("Defaulting field namespace to %s", config.Spec.Namespace)
	}

	if config.Spec.ExposeStrategy == "" {
		config.Spec.ExposeStrategy = "NodePort"
		logger.Debugf("Defaulting field exposeStrategy to %s", config.Spec.ExposeStrategy)
	}

	auth := config.Spec.Auth

	if auth.TokenIssuer == "" {
		auth.TokenIssuer = fmt.Sprintf("https://%s/dex", config.Spec.Domain)
		logger.Debugf("Defaulting field auth.tokenIssuer to %s", auth.TokenIssuer)
	}

	if auth.ClientID == "" {
		auth.ClientID = "kubermatic"
		logger.Debugf("Defaulting field auth.clientID to %s", auth.ClientID)
	}

	if auth.IssuerClientID == "" {
		auth.IssuerClientID = fmt.Sprintf("%sIssuer", auth.ClientID)
		logger.Debugf("Defaulting field auth.issuerClientID to %s", auth.IssuerClientID)
	}

	if auth.IssuerRedirectURL == "" {
		auth.IssuerRedirectURL = fmt.Sprintf("https://%s/api/v1/kubeconfig", config.Spec.Domain)
		logger.Debugf("Defaulting field auth.issuerRedirectURL to %s", auth.IssuerRedirectURL)
	}

	config.Spec.Auth = auth

	kubermaticDefaultImage := fmt.Sprintf("quay.io/kubermatic/api:%s", resources.KUBERMATICCOMMIT)
	addonsDefaultImage := fmt.Sprintf("quay.io/kubermatic/addons:%s", resources.KUBERMATICCOMMIT)

	r.defaultImage(&config.Spec.API.Image, kubermaticDefaultImage, "api.image", logger)
	r.defaultImage(&config.Spec.UI.Image, "quay.io/kubermatic/ui-v2:v1.3.0", "ui.image", logger)
	r.defaultImage(&config.Spec.MasterController.Image, kubermaticDefaultImage, "masterController.image", logger)
	r.defaultImage(&config.Spec.SeedController.Image, kubermaticDefaultImage, "seedController.image", logger)
	r.defaultImage(&config.Spec.SeedController.Addons.Kubernetes.Image, addonsDefaultImage, "seedController.addons.kubernetes.image", logger)
	r.defaultImage(&config.Spec.SeedController.Addons.Openshift.Image, "quay.io/kubermatic/openshift-addons:v0.9", "seedController.addons.openshift.image", logger)

	var (
		err       error
		defaulted bool
	)

	if !equality.Semantic.DeepEqual(original, config) {
		err = r.Client.Update(r.ctx, config)
		defaulted = true
	}

	return defaulted, err
}

func (r *Reconciler) defaultImage(img *string, defaultImage string, key string, logger *zap.SugaredLogger) {
	if *img == "" {
		*img = defaultImage
		logger.Debugf("Defaulting Docker image for %s to %s", key, defaultImage)
	}
}

// applyOwnerLabels is generating a new ObjectModifier that wraps an ObjectCreator and takes care
// of applying the default labels and annotations from this operator. These are then used to
// establish a weak ownership.
func (r *Reconciler) applyOwnerLabels(config *operatorv1alpha1.KubermaticConfiguration) reconciling.ObjectModifier {
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

			identifier, err := cache.MetaNamespaceKeyFunc(config)
			if err != nil {
				return obj, fmt.Errorf("failed to determine KubermaticConfiguration string key: %v", err)
			}

			annotations[ConfigurationOwnerAnnotation] = identifier
			o.SetAnnotations(annotations)

			labels := o.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[ManagedByLabel] = ControllerName
			o.SetLabels(labels)

			return obj, nil
		}
	}
}

// volumeRevisionLabels scans volume mounts for pod templates for ConfigMaps and Secrets and
// will then put new labels for these mounts onto the pod template, causing restarts when
// the volumes changed.
func (r *Reconciler) volumeRevisionLabels() reconciling.ObjectModifier {
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

			volumeLabels, err := resources.VolumeRevisionLabels(r.ctx, r.Client, deployment.Namespace, deployment.Spec.Template.Spec.Volumes)
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
