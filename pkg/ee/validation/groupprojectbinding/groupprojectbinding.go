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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ValidateCreate(ctx context.Context, binding *kubermaticv1.GroupProjectBinding, client ctrlruntimeclient.Client) error {
	existingBindings := &kubermaticv1.GroupProjectBindingList{}
	if err := client.List(ctx, existingBindings); err != nil {
		return fmt.Errorf("failed to list GroupProjectBindings: %w", err)
	}

	for _, existing := range existingBindings.Items {
		if existing.Spec.Group == binding.Spec.Group && existing.Spec.ProjectID == binding.Spec.ProjectID {
			return fmt.Errorf("group %q is already bound to project %q", binding.Spec.Group, binding.Spec.ProjectID)
		}
	}

	return nil
}

func ValidateUpdate(oldGroupProjectBinding *kubermaticv1.GroupProjectBinding, newGroupProjectBinding *kubermaticv1.GroupProjectBinding) error {
	if oldGroupProjectBinding.Spec.ProjectID != newGroupProjectBinding.Spec.ProjectID {
		return errors.New("attribute \"projectID\" cannot be updated for existing GroupProjectBinding resource")
	}

	return nil
}
