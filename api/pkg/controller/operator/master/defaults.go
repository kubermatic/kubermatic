package master

import (
	"fmt"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/api/equality"
)

func (r *Reconciler) defaultConfiguration(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) (bool, error) {
	logger.Debug("Applying defaults to Kubermatic configuration")

	original := config.DeepCopy()

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
