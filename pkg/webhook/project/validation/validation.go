/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Project CRD.
type validator struct {
	client ctrlruntimeclient.Client
}

// NewValidator returns a new cluster validator.
func NewValidator(client ctrlruntimeclient.Client) *validator {
	return &validator{
		client: client,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	project, ok := obj.(*kubermaticv1.Project)
	if !ok {
		return errors.New("object is not a Project")
	}

	// Projects need to have a User owner, because based on it all the RBACs are created by the rbac-controller-manager
	return v.validateProjectOwner(ctx, project)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	newProject, ok := newObj.(*kubermaticv1.Project)
	if !ok {
		return errors.New("new object is not a Project")
	}

	// Projects need to have a User owner, because based on it all the RBACs are created by the rbac-controller-manager
	return v.validateProjectOwner(ctx, newProject)
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (v *validator) validateProjectOwner(ctx context.Context, project *kubermaticv1.Project) error {
	metaPath := field.NewPath("meta", "ownerReferences")

	if len(project.OwnerReferences) == 0 {
		return field.Required(metaPath, "User owner ref for project required")
	}

	userOwnerRefPresent := false
	for _, ownerRef := range project.OwnerReferences {
		if ownerRef.Kind != kubermaticv1.UserKindName {
			continue
		}
		userOwnerRefPresent = true

		var user kubermaticv1.User
		if err := v.client.Get(ctx, types.NamespacedName{Name: ownerRef.Name}, &user); err != nil {
			if kerrors.IsNotFound(err) {
				return field.Invalid(metaPath.Child("name"), ownerRef.Name, "no such user exists")
			}

			return field.InternalError(metaPath, fmt.Errorf("failed to get user: %w", err))
		}
	}

	if !userOwnerRefPresent {
		return field.Required(metaPath, "User owner ref for project required")
	}

	return nil
}
