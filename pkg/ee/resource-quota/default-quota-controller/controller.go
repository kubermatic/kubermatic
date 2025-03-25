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

package defaultcontroller

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	k8cequality "k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	utilpredicate "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName                   = "kkp-master-default-resource-quota-controller"
	DefaultProjectResourceQuotaKey   = "kkp-default-resource-quota"
	DefaultProjectResourceQuotaValue = "true"
)

type reconciler struct {
	masterClient ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	reconciler := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		masterClient: mgr.GetClient(),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.KubermaticSetting{}, builder.WithPredicates(utilpredicate.ByName(kubermaticv1.GlobalSettingsName), withSettingsEventFilter())).
		// Watch for creation of Project; we need to make sure that we create default project quotas, if required.
		Watches(&kubermaticv1.Project{}, enqueueProjectQuotas(reconciler.masterClient), builder.WithPredicates(projectEventFilter())).
		Build(reconciler)

	return err
}

// Reconcile creates/updates/deletes default project resource quota based on the default resource quota setting in Kubermatic settings.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	setting := &kubermaticv1.KubermaticSetting{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, setting); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get global settings %q: %w", setting.Name, err)
	}

	err := r.reconcile(ctx, setting, log)
	if err != nil {
		r.recorder.Event(setting, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, setting *kubermaticv1.KubermaticSetting, log *zap.SugaredLogger) error {
	projects := &kubermaticv1.ProjectList{}
	if err := r.masterClient.List(ctx, projects); err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	resourceQuotas := &kubermaticv1.ResourceQuotaList{}
	if err := r.masterClient.List(ctx, resourceQuotas); err != nil {
		return fmt.Errorf("failed to list resource quotas: %w", err)
	}

	// Delete all default project quotas if user didn't specify any default project quota.
	if !setting.Spec.HasDefaultProjectResourceQuota() {
		return r.handleDeletion(ctx)
	}

	return r.synchronizeResourceQuotas(ctx, setting.Spec.DefaultProjectResourceQuota, projects, resourceQuotas)
}

func (r *reconciler) synchronizeResourceQuotas(ctx context.Context, defaultResourceQuota *kubermaticv1.DefaultProjectResourceQuota, projects *kubermaticv1.ProjectList, quotas *kubermaticv1.ResourceQuotaList) error {
	// We need to synchronize default resource quotas.
	var defaultQuotaFactories []reconciling.NamedResourceQuotaReconcilerFactory

	// Create a lookup for projects with resource quotas.
	resourceQuotaLookup := map[string]kubermaticv1.ResourceQuota{}
	for _, quota := range quotas.Items {
		if quota.Spec.Subject.Kind == kubermaticv1.ProjectSubjectKind {
			resourceQuotaLookup[quota.Spec.Subject.Name] = quota
		}
	}

	// Iterate over all the projects and synchronize projects with their default quotas.
	for _, project := range projects.Items {
		// Ignore projects that are queued for deletion.
		if project.DeletionTimestamp != nil {
			continue
		}

		quota, ok := resourceQuotaLookup[project.Name]
		if !ok {
			// Default resource quota doesn't exist.
			resourceQuota := genDefaultResourceQuota(defaultResourceQuota, &project)
			defaultQuotaFactories = append(defaultQuotaFactories, projectQuotaReconcilerFactory(resourceQuota))
			continue
		}

		// This is not a default quota and should be skipped.
		if val, ok := quota.Labels[DefaultProjectResourceQuotaKey]; !ok || val != DefaultProjectResourceQuotaValue {
			continue
		}

		// Quota already exists and we need to update it
		if !k8cequality.Semantic.DeepEqual(quota.Spec.Quota, defaultResourceQuota.Quota) {
			quota.Spec.Quota = defaultResourceQuota.Quota
			defaultQuotaFactories = append(defaultQuotaFactories, projectQuotaReconcilerFactory(&quota))
		}
	}

	// Create or Update the resource quotas.
	if err := reconciling.ReconcileResourceQuotas(ctx, defaultQuotaFactories, "", r.masterClient); err != nil {
		return fmt.Errorf("failed to reconcile ResourceQuotas: %w", err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context) error {
	req, err := labels.NewRequirement(DefaultProjectResourceQuotaKey, selection.Equals, []string{DefaultProjectResourceQuotaValue})
	if err != nil {
		return err
	}
	listOpts := ctrlruntimeclient.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req),
	}
	deleteAllOfOptions := &ctrlruntimeclient.DeleteAllOfOptions{
		ListOptions: listOpts,
	}

	if err = r.masterClient.DeleteAllOf(ctx, &kubermaticv1.ResourceQuota{}, deleteAllOfOptions); err != nil {
		return fmt.Errorf("failed to delete default ResourceQuotas: %w", err)
	}
	return nil
}

func genDefaultResourceQuota(defaultResourceQuota *kubermaticv1.DefaultProjectResourceQuota, project *kubermaticv1.Project) *kubermaticv1.ResourceQuota {
	quota := &kubermaticv1.ResourceQuota{}
	quota.Labels = map[string]string{
		DefaultProjectResourceQuotaKey: DefaultProjectResourceQuotaValue,
	}
	quota.Spec.Subject = kubermaticv1.Subject{
		Name: project.Name,
		Kind: kubermaticv1.ProjectSubjectKind,
	}
	quota.Spec.Quota = defaultResourceQuota.Quota
	quota.Name = buildNameFromSubject(quota.Spec.Subject)
	return quota
}

func buildNameFromSubject(subject kubermaticv1.Subject) string {
	return fmt.Sprintf("%s-%s", subject.Kind, subject.Name)
}

// just reconcile if the default project resource quota changed.
func withSettingsEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster, ok := e.ObjectOld.(*kubermaticv1.KubermaticSetting)
			if !ok {
				return false
			}
			newCluster, ok := e.ObjectNew.(*kubermaticv1.KubermaticSetting)
			if !ok {
				return false
			}
			return !reflect.DeepEqual(oldCluster.Spec.DefaultProjectResourceQuota, newCluster.Spec.DefaultProjectResourceQuota)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func projectQuotaReconcilerFactory(resourceQuota *kubermaticv1.ResourceQuota) reconciling.NamedResourceQuotaReconcilerFactory {
	return func() (string, reconciling.ResourceQuotaReconciler) {
		return resourceQuota.Name, func(existing *kubermaticv1.ResourceQuota) (*kubermaticv1.ResourceQuota, error) {
			existing.Spec = resourceQuota.Spec

			if resourceQuota.Labels != nil {
				if existing.Labels == nil {
					existing.Labels = map[string]string{}
				}
				for k, v := range resourceQuota.Labels {
					existing.Labels[k] = v
				}
			}

			if resourceQuota.Annotations != nil {
				if existing.Annotations == nil {
					existing.Annotations = map[string]string{}
				}
				for k, v := range resourceQuota.Annotations {
					existing.Annotations[k] = v
				}
			}

			return existing, nil
		}
	}
}

func enqueueProjectQuotas(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		globalSettings := &kubermaticv1.KubermaticSetting{}
		if err := client.Get(ctx, types.NamespacedName{Name: kubermaticv1.GlobalSettingsName}, globalSettings); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to get global settings %q: %w", kubermaticv1.GlobalSettingsName, err))
			return requests
		}

		if globalSettings.Spec.HasDefaultProjectResourceQuota() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: globalSettings.GetName(),
			}})
		}
		return requests
	})
}

func projectEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
