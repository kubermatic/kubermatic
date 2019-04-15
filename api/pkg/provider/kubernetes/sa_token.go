package kubernetes

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	kubev1 "k8s.io/client-go/listers/core/v1"
)

const (
	kubermaticNamespace = "kubermatic"
	labelTokenName      = "token"
)

// NewServiceAccountProvider returns a service account provider
func NewServiceAccountTokenProvider(kubernetesImpersonationClient kubernetesImpersonationClient, tokenGenerator serviceaccount.TokenGenerator, tokenAuthenticator serviceaccount.TokenAuthenticator, secretLister kubev1.SecretLister) *ServiceAccountTokenProvider {
	return &ServiceAccountTokenProvider{
		kubernetesImpersonationClient: kubernetesImpersonationClient,
		tokenGenarator:                tokenGenerator,
		tokenAuthenticator:            tokenAuthenticator,
		secretLister:                  secretLister,
	}
}

// ServiceAccountProvider manages service account resources
type ServiceAccountTokenProvider struct {
	// kubernetesImpersonationClient is used as a ground for impersonation
	kubernetesImpersonationClient kubernetesImpersonationClient

	// tokenGenarator generates a token which will identify the given ServiceAccount
	tokenGenarator serviceaccount.TokenGenerator

	// tokenAuthenticator checks given token and transform it to custom claim object
	tokenAuthenticator serviceaccount.TokenAuthenticator

	secretLister kubev1.SecretLister
}

// Create creates a new token for service account
func (p *ServiceAccountTokenProvider) Create(userInfo *provider.UserInfo, sa *kubermaticv1.User, tokenName, projectID string) (*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if sa == nil {
		return nil, kerrors.NewBadRequest("service account cannot be nil")
	}

	secret := &v1.Secret{}
	secret.Name = fmt.Sprintf("sa-token-%s", rand.String(10))
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			UID:        sa.GetUID(),
			Name:       sa.Name,
		},
	}
	secret.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"name":                         tokenName,
	}
	secret.Data = make(map[string][]byte)
	token, err := p.tokenGenarator.Generate(serviceaccount.Claims(sa.Spec.Email, projectID, secret.Name))
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	secret.Data[labelTokenName] = []byte(token)
	secret.Type = "Opaque"

	kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	return kubernetesImpersonatedClient.CoreV1().Secrets(kubermaticNamespace).Create(secret)
}

// List  gets tokens for the given service account and project
func (p *ServiceAccountTokenProvider) List(userInfo *provider.UserInfo, project *kubermaticv1.Project, sa *kubermaticv1.User, options *provider.ServiceAccountTokenListOptions) ([]*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if sa == nil {
		return nil, kerrors.NewBadRequest("sa cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountTokenListOptions{}
	}
	saCopy := *sa
	saCopy.Name = addSAPrefix(saCopy.Name)

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, project.Name))
	if err != nil {
		return nil, err
	}
	allSecrets, err := p.secretLister.List(labelSelector)
	if err != nil {
		return nil, err
	}

	resultList := make([]*v1.Secret, 0)
	for _, secret := range allSecrets {
		if strings.HasPrefix(secret.Name, "sa-token") {
			for _, owner := range secret.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.UserKindName &&
					owner.Name == saCopy.Name && owner.UID == saCopy.UID {
					resultList = append(resultList, secret)
				}
			}
		}
	}

	// Note:
	// After we get the list of tokens we try to get at least one item using unprivileged account to see if the token have read access
	if len(resultList) > 0 {

		kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
		if err != nil {
			return nil, err
		}

		tokenToGet := resultList[0]
		_, err = kubernetesImpersonatedClient.CoreV1().Secrets(kubermaticNamespace).Get(tokenToGet.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

	}

	if len(options.TokenName) == 0 {
		return resultList, nil
	}

	filteredList := make([]*v1.Secret, 0)
	for _, token := range resultList {
		name, ok := token.Labels["name"]
		if ok {
			if name == options.TokenName {
				filteredList = append(filteredList, token)
				break
			}
		}
	}

	return filteredList, nil
}

func (p *ServiceAccountTokenProvider) GetTokenAuthenticator() serviceaccount.TokenAuthenticator {
	return p.tokenAuthenticator
}
