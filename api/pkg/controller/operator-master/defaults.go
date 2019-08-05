package operatormaster

import (
	"fmt"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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

	r.defaultImage(&config.Spec.API.Image, "quay.io/kubermatic/api", "api.image", logger)
	r.defaultImage(&config.Spec.UI.Image, "quay.io/kubermatic/ui-v2:v1.3.0", "ui.image", logger)
	r.defaultImage(&config.Spec.MasterController.Image, "quay.io/kubermatic/api", "masterController.image", logger)
	r.defaultImage(&config.Spec.SeedController.Image, "quay.io/kubermatic/api", "seedController.image", logger)
	r.defaultImage(&config.Spec.SeedController.Addons.Kubernetes.Image, "quay.io/kubermatic/addons:v0.2.19", "seedController.addons.kubernetes.image", logger)
	r.defaultImage(&config.Spec.SeedController.Addons.Openshift.Image, "quay.io/kubermatic/openshift-addons:v0.9", "seedController.addons.openshift.image", logger)

	var err error
	if !equality.Semantic.DeepEqual(original, config) {
		err = r.Client.Update(r.ctx, config)
	}

	return false, err
}

func (r *Reconciler) defaultImage(img *string, defaultImage string, key string, logger *zap.SugaredLogger) {
	if *img == "" {
		*img = defaultImage
		logger.Debugf("Defaulting Docker image for %s to %s", key, defaultImage)
	}
}

// defaultFields is generating a new ObjectModifier that wraps an
// ObjectCreator and takes care of applying the default labels and
// annotations from this operator. These are then used to establish
// a weak ownership.
func (r *Reconciler) defaultFields(config *operatorv1alpha1.KubermaticConfiguration) reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			if o, ok := obj.(metav1.Object); ok {
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
			}

			return obj, nil
		}
	}
}
