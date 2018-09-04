package resources

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbacv1lister "k8s.io/client-go/listers/rbac/v1"
)

// EnsureRole will create the role with the passed create function & create or update it if necessary.
// To check if its necessary it will do a lookup of the resource at the lister & compare the existing Role with the created one
func EnsureRole(data *TemplateData, create RoleCreator, roleLister rbacv1lister.RoleNamespaceLister, roleClient rbacv1client.RoleInterface) error {
	var existing *rbacv1.Role
	role, err := create(data, nil)
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if existing, err = roleLister.Get(role.Name); err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		if _, err = roleClient.Create(role); err != nil {
			return fmt.Errorf("failed to create Role %s: %v", role.Name, err)
		}
		return nil
	}

	role, err = create(data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Role: %v", err)
	}

	if DeepEqual(role, existing) {
		return nil
	}

	if _, err = roleClient.Update(role); err != nil {
		return fmt.Errorf("failed to update Role %s: %v", role.Name, err)
	}

	return nil
}
