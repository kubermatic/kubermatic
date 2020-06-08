package kubernetes

import (
	"context"
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewAdminProvider returns a admin provider
func NewAdminProvider(client ctrlruntimeclient.Client) *AdminProvider {
	return &AdminProvider{
		client: client,
	}
}

// AdminProvider manages admin resources
type AdminProvider struct {
	client ctrlruntimeclient.Client
}

// GetAdmins return all users with admin rights
func (a *AdminProvider) GetAdmins(userInfo *provider.UserInfo) ([]kubermaticv1.User, error) {
	var adminList []kubermaticv1.User
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	users := &kubermaticv1.UserList{}
	if err := a.client.List(context.Background(), users); err != nil {
		return nil, err
	}

	for _, user := range users.Items {
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
	userList := &kubermaticv1.UserList{}
	if err := a.client.List(context.Background(), userList); err != nil {
		return nil, err
	}
	for _, user := range userList.Items {
		if strings.EqualFold(user.Spec.Email, email) {
			userCopy := user.DeepCopy()
			userCopy.Spec.IsAdmin = isAdmin
			if err := a.client.Update(context.Background(), userCopy); err != nil {
				return nil, err
			}
			return userCopy, nil
		}
	}
	return nil, fmt.Errorf("the given user %s was not found", email)
}
