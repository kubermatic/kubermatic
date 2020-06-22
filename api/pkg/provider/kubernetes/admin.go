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
	"strings"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewAdminProvider returns a admin provider
func NewAdminProvider(client kubermaticclientset.Interface, userLister kubermaticv1lister.UserLister) *AdminProvider {
	return &AdminProvider{
		client:     client,
		userLister: userLister,
	}
}

// AdminProvider manages admin resources
type AdminProvider struct {
	client     kubermaticclientset.Interface
	userLister kubermaticv1lister.UserLister
}

// GetAdmins return all users with admin rights
func (a *AdminProvider) GetAdmins(userInfo *provider.UserInfo) ([]kubermaticv1.User, error) {
	var adminList []kubermaticv1.User
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	users, err := a.userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Spec.IsAdmin {
			adminList = append(adminList, *user.DeepCopy())
		}
	}

	return adminList, nil
}

// SetAdmin set/clear admin rights
func (a *AdminProvider) SetAdmin(userInfo *provider.UserInfo, email string, isAdmin bool) (*kubermaticv1.User, error) {
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	if strings.EqualFold(userInfo.Email, email) {
		return nil, kerrors.NewBadRequest("can not change own privileges")
	}
	userList, err := a.client.KubermaticV1().Users().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, user := range userList.Items {
		if strings.EqualFold(user.Spec.Email, email) {
			user.Spec.IsAdmin = isAdmin
			return a.client.KubermaticV1().Users().Update(&user)
		}
	}
	return nil, fmt.Errorf("the given user %s was not found", email)
}
