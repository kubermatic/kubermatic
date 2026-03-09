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

package labelownercontroller

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// This controller sets the ResourceQuotas subject labels and owner reference.
const controllerName = "kkp-resource-quota-label-owner-controller"

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	recorder     events.EventRecorder
}

func Add(
	masterMgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	log = log.Named(controllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		recorder:     masterMgr.GetEventRecorder(controllerName),
	}

	_, err := builder.ControllerManagedBy(masterMgr).
		Named(controllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ResourceQuota{}).
		Watches(&kubermaticv1.Project{}, enqueueResourceQuotaForProject(r.masterClient), builder.WithPredicates(withProjectEventFilter())).
		Build(r)

	return err
}

func resourceQuotaLabelOwnerRefReconcilerFactory(rq *kubermaticv1.ResourceQuota) reconciling.NamedResourceQuotaReconcilerFactory {
	return func() (string, reconciling.ResourceQuotaReconciler) {
		return rq.Name, func(c *kubermaticv1.ResourceQuota) (*kubermaticv1.ResourceQuota, error) {
			// ensure labels and owner ref
			kuberneteshelper.EnsureLabels(c, map[string]string{
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: rq.Spec.Subject.Kind,
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: rq.Spec.Subject.Name,
			})
			c.OwnerReferences = rq.OwnerReferences

			return c, nil
		}
	}
}

// Reconcile reconciles the resource quota subject labels and owner references.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, resourceQuota); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, resourceQuota)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, resourceQuota *kubermaticv1.ResourceQuota) error {
	// skip if deleted
	if !resourceQuota.DeletionTimestamp.IsZero() {
		return nil
	}

	// set master labels and owner ref
	if strings.EqualFold(resourceQuota.Spec.Subject.Kind, kubermaticv1.ProjectSubjectKind) {
		err := ensureProjectOwnershipRef(ctx, r.masterClient, resourceQuota)
		if err != nil {
			return err
		}
	}
	resourceQuotaMasterReconcilerFactories := []reconciling.NamedResourceQuotaReconcilerFactory{
		resourceQuotaLabelOwnerRefReconcilerFactory(resourceQuota),
	}

	return reconciling.ReconcileResourceQuotas(ctx, resourceQuotaMasterReconcilerFactories, "", r.masterClient)
}

func ensureProjectOwnershipRef(ctx context.Context, client ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota) error {
	subjectName := resourceQuota.Spec.Subject.Name
	ownRefs := resourceQuota.OwnerReferences

	// check if reference already exists
	for _, owners := range ownRefs {
		if owners.Kind == kubermaticv1.ProjectKindName && owners.Name == subjectName {
			return nil
		}
	}

	// set project reference
	project := &kubermaticv1.Project{}
	key := types.NamespacedName{Name: subjectName}
	if err := client.Get(ctx, key, project); err != nil {
		return err
	}

	projectRef := resources.GetProjectRef(project)
	ownRefs = append(ownRefs, projectRef)
	resourceQuota.SetOwnerReferences(ownRefs)

	return nil
}

func enqueueResourceQuotaForProject(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		name := a.GetName()

		resourceQuotaList := &kubermaticv1.ResourceQuotaList{}
		if err := client.List(ctx, resourceQuotaList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list resourceQuotas: %w", err))
		}

		for _, rq := range resourceQuotaList.Items {
			if strings.EqualFold(rq.Spec.Subject.Name, name) && kubermaticv1.ProjectSubjectKind == rq.Spec.Subject.Kind {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      rq.Name,
					Namespace: rq.Namespace,
				}})
				break
			}
		}
		return requests
	})
}

func withProjectEventFilter() predicate.Predicate {
	return predicate.Funcs{
		// just handle create events, in other cases the controller should already set the labels/owner ref
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
