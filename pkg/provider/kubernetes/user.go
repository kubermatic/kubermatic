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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type blacklistToken struct {
	Token  string     `json:"token"`
	Expiry apiv1.Time `json:"expiry"`
}

// NewUserProvider returns a user provider.
func NewUserProvider(runtimeClient ctrlruntimeclient.Client) *UserProvider {
	return &UserProvider{
		runtimeClient: runtimeClient,
	}
}

// UserProvider manages user resources.
type UserProvider struct {
	runtimeClient ctrlruntimeclient.Client
}

var _ provider.UserProvider = &UserProvider{}

// UserByID returns a user by the given ID.
func (p *UserProvider) UserByID(ctx context.Context, id string) (*kubermaticv1.User, error) {
	user := &kubermaticv1.User{}
	if err := p.runtimeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: id}, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UserByEmail returns a user by the given email.
func (p *UserProvider) UserByEmail(ctx context.Context, email string) (*kubermaticv1.User, error) {
	users := &kubermaticv1.UserList{}
	if err := p.runtimeClient.List(ctx, users); err != nil {
		return nil, err
	}

	for _, user := range users.Items {
		if strings.EqualFold(user.Spec.Email, email) {
			return user.DeepCopy(), nil
		}
	}

	return nil, provider.ErrNotFound
}

// CreateUser creates a new user. If no user is found at all the created user is elected as the first admin.
//
// Note that:
// The name of the newly created resource will be unique and it is derived from the user's email address (sha256(email)
// This prevents creating multiple resources for the same user with the same email address.
//
// In the beginning I was considering to hex-encode the email address as it will produce a unique output because the email address in unique.
// The only issue I have found with this approach is that the length can get quite long quite fast.
// Thus decided to use sha256 as it produces fixed output and the hash collisions are very, very, very, very rare.

func (p *UserProvider) CreateUser(ctx context.Context, name, email string, groups []string) (*kubermaticv1.User, error) {
	if len(name) == 0 || len(email) == 0 {
		return nil, apierrors.NewBadRequest("Email, ID and Name cannot be empty when creating a new user resource")
	}

	if kubermaticv1helper.IsProjectServiceAccount(email) {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("cannot add a user with the given email %s as the name is reserved, please try a different email address", email))
	}

	user := &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%x", sha256.Sum256([]byte(email))),
		},
		Spec: kubermaticv1.UserSpec{
			Name:   name,
			Email:  email,
			Groups: groups,
		},
	}

	var userList kubermaticv1.UserList
	if err := p.runtimeClient.List(ctx, &userList); err != nil {
		return nil, err
	}

	// Elect the first user as admin
	if len(userList.Items) == 0 {
		user.Spec.IsAdmin = true
	}

	if err := p.runtimeClient.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser updates user.
func (p *UserProvider) UpdateUser(ctx context.Context, user *kubermaticv1.User) (*kubermaticv1.User, error) {
	// make sure the first patch doesn't override the status
	status := user.Status.DeepCopy()

	if err := p.runtimeClient.Update(ctx, user); err != nil {
		return nil, err
	}

	oldUser := user.DeepCopy()
	user.Status = *status
	if err := p.runtimeClient.Status().Patch(ctx, user, ctrlruntimeclient.MergeFrom(oldUser)); err != nil {
		return nil, err
	}

	return user, nil
}

func (p *UserProvider) InvalidateToken(ctx context.Context, user *kubermaticv1.User, token string, expiry apiv1.Time) error {
	if user == nil {
		return apierrors.NewBadRequest("user cannot be nil")
	}
	if token == "" {
		return apierrors.NewBadRequest("token cannot be empty")
	}

	secret, err := ensureTokenBlacklistSecret(ctx, p.runtimeClient, user)
	if err != nil {
		return err
	}
	tokenList, ok := secret.Data[resources.TokenBlacklist]
	if !ok {
		return fmt.Errorf("secret %s has no key %s", secret.Name, resources.TokenBlacklist)
	}

	blockedTokens := make([]blacklistToken, 0)
	if len(tokenList) > 0 {
		if err := json.Unmarshal(tokenList, &blockedTokens); err != nil {
			return err
		}
	}

	blockedTokens = append(blockedTokens, blacklistToken{
		Token:  token,
		Expiry: expiry,
	})

	blockedTokens = clearExpiredTokens(blockedTokens)

	tokenJSON, err := json.Marshal(&blockedTokens)
	if err != nil {
		return err
	}

	secret.Data = map[string][]byte{
		resources.TokenBlacklist: tokenJSON,
	}

	if err := p.runtimeClient.Update(ctx, secret); err != nil {
		return err
	}

	return nil
}

func (p *UserProvider) GetInvalidatedTokens(ctx context.Context, user *kubermaticv1.User) ([]string, error) {
	result := make([]string, 0)
	if user == nil {
		return nil, apierrors.NewBadRequest("user cannot be nil")
	}
	if user.Spec.InvalidTokensReference == nil {
		return result, nil
	}
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, p.runtimeClient)
	tokenList, err := secretKeyGetter(user.Spec.InvalidTokensReference, resources.TokenBlacklist)
	if err != nil {
		return nil, err
	}
	blockedTokens := make([]blacklistToken, 0)
	if len(tokenList) > 0 {
		if err := json.Unmarshal([]byte(tokenList), &blockedTokens); err != nil {
			return nil, err
		}
	}

	for _, token := range blockedTokens {
		result = append(result, token.Token)
	}

	return result, nil
}

func (p *UserProvider) List(ctx context.Context) ([]kubermaticv1.User, error) {
	ul := kubermaticv1.UserList{}
	if err := p.runtimeClient.List(ctx, &ul); err != nil {
		return nil, err
	}

	return ul.Items, nil
}

func ensureTokenBlacklistSecret(ctx context.Context, client ctrlruntimeclient.Client, user *kubermaticv1.User) (*corev1.Secret, error) {
	name := user.GetInvalidTokensReferenceSecretName()

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: name}
	existingSecret := &corev1.Secret{}
	if err := client.Get(ctx, namespacedName, existingSecret); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for secret %q: %w", name, err)
	}

	if existingSecret.Name == "" {
		existingSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.TokenBlacklist: {},
			},
		}

		if err := client.Create(ctx, existingSecret); err != nil {
			return nil, fmt.Errorf("failed to create token blacklist secret: %w", err)
		}
	}

	if user.Spec.InvalidTokensReference == nil {
		user.Spec.InvalidTokensReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      name,
				Namespace: resources.KubermaticNamespace,
			},
		}
		if err := client.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	return existingSecret, nil
}

func clearExpiredTokens(tokens []blacklistToken) []blacklistToken {
	blockedTokens := make([]blacklistToken, 0)

	for _, blockedToken := range tokens {
		if blockedToken.Expiry.After(time.Now()) {
			blockedTokens = append(blockedTokens, blockedToken)
		}
	}

	return blockedTokens
}
