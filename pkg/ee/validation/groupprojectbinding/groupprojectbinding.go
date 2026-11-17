//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package groupprojectbinding

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	authenticationv1 "k8s.io/api/authentication/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// roleRank orders the project roles a GroupProjectBinding can grant. projectmanagers manage
// members but not clusters, so they rank between editors and owners: they may grant up to
// projectmanagers, never owners.
var roleRank = map[string]int{
	"viewers":         1,
	"editors":         2,
	"projectmanagers": 3,
	"owners":          4,
}

func ValidateCreate(ctx context.Context, binding *kubermaticv1.GroupProjectBinding, client ctrlruntimeclient.Client) error {
	if err := validateRequesterRole(ctx, client, binding.Spec.ProjectID, binding.Spec.Role); err != nil {
		return err
	}

	duplicate, err := hasDuplicateGroupProjectBinding(ctx, client, binding, binding.Name)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("group %q is already bound to project %q", binding.Spec.Group, binding.Spec.ProjectID)
	}

	return nil
}

func ValidateUpdate(ctx context.Context, oldGroupProjectBinding *kubermaticv1.GroupProjectBinding, newGroupProjectBinding *kubermaticv1.GroupProjectBinding, client ctrlruntimeclient.Client) error {
	if oldGroupProjectBinding.Spec.ProjectID != newGroupProjectBinding.Spec.ProjectID {
		return errors.New("attribute \"projectID\" cannot be updated for existing GroupProjectBinding resource")
	}

	// changing the group of a binding hands its role to other people, so it needs the same
	// rank as changing the role. The old role counts too: a member may not change a binding
	// above their own rank.
	if oldGroupProjectBinding.Spec.Role != newGroupProjectBinding.Spec.Role || oldGroupProjectBinding.Spec.Group != newGroupProjectBinding.Spec.Group {
		for _, role := range []string{oldGroupProjectBinding.Spec.Role, newGroupProjectBinding.Spec.Role} {
			if err := validateRequesterRole(ctx, client, newGroupProjectBinding.Spec.ProjectID, role); err != nil {
				return err
			}
		}
	}

	duplicate, err := hasDuplicateGroupProjectBinding(ctx, client, newGroupProjectBinding, newGroupProjectBinding.Name)
	if err != nil {
		return err
	}
	if duplicate {
		return fmt.Errorf("group %q is already bound to project %q", newGroupProjectBinding.Spec.Group, newGroupProjectBinding.Spec.ProjectID)
	}

	return nil
}

func hasDuplicateGroupProjectBinding(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding, excludeName string) (bool, error) {
	existingBindings := &kubermaticv1.GroupProjectBindingList{}
	if err := client.List(ctx, existingBindings); err != nil {
		return false, fmt.Errorf("failed to list GroupProjectBindings: %w", err)
	}

	for _, existing := range existingBindings.Items {
		if existing.Name == excludeName {
			continue
		}
		if existing.Spec.Group == binding.Spec.Group && existing.Spec.ProjectID == binding.Spec.ProjectID {
			return true, nil
		}
	}

	return false, nil
}

// validateRequesterRole rejects a role higher than the requester's own role in the project.
// The KKP API writes GroupProjectBindings for project members with an impersonated client.
// Its groups are "<role>-<projectID>" for a role from a UserProjectBinding and
// "<idp group>-<projectID>" for a role from a GroupProjectBinding. Requests without such a
// group do not come from a project member acting through the KKP API: KKP admins (the KKP
// API uses its own ServiceAccount for them), KKP controllers and master cluster admins.
// RBAC already decided that they may write GroupProjectBindings, so they are not limited here.
func validateRequesterRole(ctx context.Context, client ctrlruntimeclient.Client, projectID, role string) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("cannot determine the requesting user: %w", err)
	}

	own, isMember, err := requesterRank(ctx, client, req.UserInfo, projectID)
	if err != nil {
		return err
	}
	if !isMember {
		return nil
	}

	// a role this check cannot rank could be anything, so project members may not grant it
	requested, ok := roleRank[role]
	if !ok {
		return fmt.Errorf("unknown role %q", role)
	}

	if requested > own {
		return fmt.Errorf("role %q is higher than the role of %q in project %q", role, req.UserInfo.Username, projectID)
	}

	return nil
}

// requesterRank returns the highest project role rank the requester's groups carry, and
// whether any of the groups belongs to the project at all.
func requesterRank(ctx context.Context, client ctrlruntimeclient.Client, userInfo authenticationv1.UserInfo, projectID string) (int, bool, error) {
	highest := 0
	found := false

	var bindings *kubermaticv1.GroupProjectBindingList

	for _, group := range userInfo.Groups {
		prefix, ok := strings.CutSuffix(group, "-"+projectID)
		if !ok {
			continue
		}

		if rank, ok := roleRank[prefix]; ok {
			found = true
			highest = max(highest, rank)
			continue
		}

		if bindings == nil {
			bindings = &kubermaticv1.GroupProjectBindingList{}
			if err := client.List(ctx, bindings); err != nil {
				return 0, false, fmt.Errorf("failed to list GroupProjectBindings: %w", err)
			}
		}

		for _, binding := range bindings.Items {
			if binding.Spec.ProjectID != projectID || binding.Spec.Group != prefix {
				continue
			}
			found = true
			// an unknown role ranks lowest, so it never lets the requester grant more
			highest = max(highest, roleRank[binding.Spec.Role])
		}
	}

	return highest, found, nil
}
