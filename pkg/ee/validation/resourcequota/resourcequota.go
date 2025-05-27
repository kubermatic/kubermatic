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

package resourcequota

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	resourcequotadefaultcontroller "k8c.io/kubermatic/v2/pkg/ee/resource-quota/default-quota-controller"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ValidateCreate(ctx context.Context,
	obj runtime.Object,
	client ctrlruntimeclient.Client) error {
	incomingQuota, ok := obj.(*kubermaticv1.ResourceQuota)
	if !ok {
		return errors.New("object is not a Resource Quota")
	}
	if incomingQuota == nil {
		return nil
	}

	currentQuotaList := &kubermaticv1.ResourceQuotaList{}
	if err := client.List(ctx, currentQuotaList, &ctrlruntimeclient.ListOptions{}); err != nil {
		return fmt.Errorf("failed to list resource quotas: %w", err)
	}

	incomingSubject := incomingQuota.Spec.Subject
	for _, currentQuota := range currentQuotaList.Items {
		currentSubject := currentQuota.Spec.Subject
		if currentSubject.Name == incomingSubject.Name && currentSubject.Kind == incomingSubject.Kind {
			return fmt.Errorf("ResourceQuota: Subject's Name %q and Kind pair %q must be unique", incomingSubject.Name, incomingSubject.Kind)
		}
	}

	return nil
}

func ValidateUpdate(ctx context.Context,
	oldObj runtime.Object,
	newObj runtime.Object) error {
	oldQuota, ok := oldObj.(*kubermaticv1.ResourceQuota)
	if !ok {
		return errors.New("existing object is not a Resource Quota")
	}
	if oldQuota == nil {
		return nil
	}
	newQuota, ok := newObj.(*kubermaticv1.ResourceQuota)
	if !ok {
		return errors.New("updated object is not a Resource Quota")
	}
	if newQuota == nil {
		return nil
	}

	oldSubject := oldQuota.Spec.Subject
	newSubject := newQuota.Spec.Subject
	if oldSubject != newSubject {
		return fmt.Errorf("Operation not permitted: updating ResourceQuota Subject is not allowed!")
	}

	return nil
}

func ValidateDelete(ctx context.Context,
	obj runtime.Object,
	client ctrlruntimeclient.Client) error {
	resourceQuota, ok := obj.(*kubermaticv1.ResourceQuota)
	if !ok {
		return errors.New("object is not a Resource Quota")
	}
	if resourceQuota == nil || resourceQuota.Labels == nil {
		return nil
	}

	// Ignore if it's not a default resource quota.
	if val, ok := resourceQuota.Labels[resourcequotadefaultcontroller.DefaultProjectResourceQuotaKey]; !ok || val != resourcequotadefaultcontroller.DefaultProjectResourceQuotaValue {
		return nil
	}

	globalSettings := &kubermaticv1.KubermaticSetting{}
	if err := client.Get(ctx, types.NamespacedName{Name: kubermaticv1.GlobalSettingsName}, globalSettings); err != nil {
		if apierrors.IsNotFound(err) {
			// Since KubermaticSettings don't exist, no validation is required.
			return nil
		}
		return fmt.Errorf("failed to get global settings %q: %w", kubermaticv1.GlobalSettingsName, err)
	}

	if globalSettings.Spec.HasDefaultProjectResourceQuota() {
		return fmt.Errorf("Removing default Project Quota resource is not allowed. Either remove the default/global project quota configuration from KubermaticSettings or update the Project Quota resource to empty or desired values.")
	}
	return nil
}
