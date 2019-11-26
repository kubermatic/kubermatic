package master

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/docker/distribution/reference"
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

	if err := r.defaultDockerRepo(&config.Spec.API.DockerRepository, resources.DefaultKubermaticImage, "api.dockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.UI.DockerRepository, resources.DefaultDashboardImage, "ui.dockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.MasterController.DockerRepository, resources.DefaultKubermaticImage, "masterController.dockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.SeedController.DockerRepository, resources.DefaultKubermaticImage, "seedController.dockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.UserCluster.KubermaticDockerRepository, resources.DefaultKubermaticImage, "userCluster.addons.kubermaticDockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.UserCluster.DNATControllerDockerRepository, resources.DefaultDNATControllerImage, "userCluster.addons.dnatControllerDockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.UserCluster.Addons.Kubernetes.DockerRepository, resources.DefaultKubernetesAddonImage, "userCluster.addons.kubernetes.dockerRepository", logger); err != nil {
		return false, err
	}

	if err := r.defaultDockerRepo(&config.Spec.UserCluster.Addons.Openshift.DockerRepository, resources.DefaultOpenshiftAddonImage, "userCluster.addons.openshift.dockerRepository", logger); err != nil {
		return false, err
	}

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

func (r *Reconciler) defaultDockerRepo(repo *string, defaultRepo string, key string, logger *zap.SugaredLogger) error {
	if *repo == "" {
		*repo = defaultRepo
		logger.Debugw("Defaulting Docker repository", "key", key, "repo", defaultRepo)

		return nil
	}

	ref, err := reference.Parse(*repo)
	if err != nil {
		return fmt.Errorf("invalid docker repository '%s' configured for %s: %v", *repo, key, err)
	}

	if _, ok := ref.(reference.Tagged); ok {
		return fmt.Errorf("it is not allowed to specify an image tag for the %s repository", key)
	}

	return nil
}
