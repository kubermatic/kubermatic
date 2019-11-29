package common

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the label containing the application's version.
	VersionLabel = "app.kubernetes.io/version"

	DockercfgSecretName                   = "dockercfg"
	DexCASecretName                       = "dex-ca"
	MasterFilesSecretName                 = "extra-files"
	SeedAdmissionWebhookName              = "kubermatic.io-seeds"
	SeedWebhookServingCASecretName        = "seed-webhook-ca"
	SeedWebhookServingCertSecretName      = "seed-webhook-cert"
	seedWebhookCommonName                 = "seed-webhook"
	seedWebhookServiceName                = "seed-webhook"
	IngressName                           = "kubermatic"
	MasterControllerManagerDeploymentName = "kubermatic-master-controller-manager"
	SeedControllerManagerDeploymentName   = "kubermatic-seed-controller-manager"
)

func NamespaceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedNamespaceCreatorGetter {
	return func() (string, reconciling.NamespaceCreator) {
		return cfg.Namespace, func(n *corev1.Namespace) (*corev1.Namespace, error) {
			if n.Labels == nil {
				n.Labels = map[string]string{}
			}

			n.Labels[NameLabel] = cfg.Namespace

			return n, nil
		}
	}
}

func DockercfgSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

func DexCASecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DexCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, map[string]string{
				"caBundle.pem": cfg.Spec.Auth.CABundle,
			}), nil
		}
	}
}

func MasterFilesSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return MasterFilesSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, cfg.Spec.MasterFiles), nil
		}
	}
}

func SeedWebhookServingCASecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	creator := certificates.GetCACreator(seedWebhookCommonName)

	return func() (string, reconciling.SecretCreator) {
		return SeedWebhookServingCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s, err := creator(s)
			if err != nil {
				return s, fmt.Errorf("failed to reconcile seed-webhook CA: %v", err)
			}

			return s, nil
		}
	}
}

func SeedWebhookServingCertSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	altNames := []string{
		fmt.Sprintf("%s.%s", seedWebhookCommonName, cfg.Namespace),
		fmt.Sprintf("%s.%s.svc", seedWebhookCommonName, cfg.Namespace),
	}

	caGetter := func() (*triple.KeyPair, error) {
		se := corev1.Secret{}
		key := types.NamespacedName{
			Namespace: cfg.Namespace,
			Name:      SeedWebhookServingCASecretName,
		}

		if err := client.Get(context.Background(), key, &se); err != nil {
			return nil, fmt.Errorf("CA certificate could not be retrieved: %v", err)
		}

		keypair, err := triple.ParseRSAKeyPair(se.Data[resources.CACertSecretKey], se.Data[resources.CAKeySecretKey])
		if err != nil {
			return nil, fmt.Errorf("CA certificate secret contains no valid key pair: %v", err)
		}

		return keypair, nil
	}

	return servingcerthelper.ServingCertSecretCreator(caGetter, SeedWebhookServingCertSecretName, seedWebhookCommonName, altNames, nil)
}

func seedAdmissionWebhookName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("kubermatic-seeds-%s", cfg.Namespace)
}

func SeedAdmissionWebhookCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return seedAdmissionWebhookName(cfg), func(hook *admissionregistrationv1beta1.ValidatingWebhookConfiguration) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
			policy := admissionregistrationv1beta1.Fail
			sideEffects := admissionregistrationv1beta1.SideEffectClassUnknown
			scope := admissionregistrationv1beta1.AllScopes

			ca, err := seedWebhookCABundle(cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find Seed Admission CA bundle: %v", err)
			}

			hook.Webhooks = []admissionregistrationv1beta1.ValidatingWebhook{
				{
					Name:                    "seeds.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &policy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1beta1.ServiceReference{
							Name:      seedWebhookServiceName,
							Namespace: cfg.Namespace,
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							NameLabel: cfg.Namespace,
						},
					},
					Rules: []admissionregistrationv1beta1.RuleWithOperations{
						{
							Rule: admissionregistrationv1beta1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"seeds"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1beta1.OperationType{
								admissionregistrationv1beta1.OperationAll,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func SeedAdmissionServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return seedWebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP

			if len(s.Spec.Ports) != 1 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "https"
			s.Spec.Ports[0].Port = 443
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8100)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			selector, err := determineWebhookServiceSelector(cfg, client)
			if err != nil {
				return s, fmt.Errorf("failed to determine SeedAdmissionWebhook target service: %v", err)
			}

			s.Spec.Selector = selector

			return s, nil
		}
	}
}

// On master clusters, point to the master-controller-manager, otherwise
// point to the seed-controller-manager. On combined master+seeds, the
// master has precedence.
func determineWebhookServiceSelector(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) (map[string]string, error) {
	deployment := appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      MasterControllerManagerDeploymentName,
		Namespace: cfg.Namespace,
	}

	err := client.Get(context.Background(), key, &deployment)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %v", key, err)
	}

	if err == nil {
		return map[string]string{
			NameLabel: MasterControllerManagerDeploymentName,
		}, nil
	}

	key = types.NamespacedName{
		Name:      SeedControllerManagerDeploymentName,
		Namespace: cfg.Namespace,
	}

	err = client.Get(context.Background(), key, &deployment)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %v", key, err)
	}

	if err == nil {
		return map[string]string{
			NameLabel: SeedControllerManagerDeploymentName,
		}, nil
	}

	return nil, fmt.Errorf("neither master- nor seed-controller-manager exist in namespace %s", cfg.Namespace)
}

func seedWebhookCABundle(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) ([]byte, error) {
	secret := corev1.Secret{}
	key := types.NamespacedName{
		Name:      SeedWebhookServingCASecretName,
		Namespace: cfg.Namespace,
	}

	err := client.Get(context.Background(), key, &secret)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve seed admission webhook CA Secret %s: %v", SeedWebhookServingCASecretName, err)
	}

	cert, ok := secret.Data[resources.CACertSecretKey]
	if !ok {
		return nil, fmt.Errorf("Secret %s does not contain CA certificate at key %s", SeedWebhookServingCASecretName, resources.CACertSecretKey)
	}

	return cert, nil
}
