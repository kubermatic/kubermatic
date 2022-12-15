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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	utilpredicate "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

func Add(mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	reconciler := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		masterClient: mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.KubermaticSetting{}}, &handler.EnqueueRequestForObject{},
		utilpredicate.ByName(kubermaticv1.GlobalSettingsName), withSettingsEventFilter()); err != nil {
		return fmt.Errorf("failed to create watch for kubermatic global settings: %w", err)
	}

	return nil
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
		log.Errorw("ReconcilingError", zap.Error(err))
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

	projectQuotas := pairProjectQuotas(projects, resourceQuotas)
	// if the setting is nil, remove all default resource quotas
	if setting.Spec.DefaultProjectResourceQuota == nil {
		if err := r.handleDeletion(ctx, projectQuotas); err != nil {
			return fmt.Errorf("error deleting default project quotas %w", err)
		}
		return nil
	}

	for _, pQuota := range projectQuotas {
		if err := r.ensureDefaultProjectQuota(ctx, setting, &pQuota.project, pQuota.quota); err != nil {
			return fmt.Errorf("error ensuring default project quotas: %w", err)
		}
	}

	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, quotas map[string]*projectQuota) error {
	for _, pQuota := range quotas {
		if pQuota.quota != nil {
			if err := r.masterClient.Delete(ctx, pQuota.quota); err != nil {
				return fmt.Errorf("error deleting default quota %q: %w", pQuota.quota.Name, err)
			}
		}
	}
	return nil
}

type projectQuota struct {
	project kubermaticv1.Project
	quota   *kubermaticv1.ResourceQuota
}

func pairProjectQuotas(projects *kubermaticv1.ProjectList, quotas *kubermaticv1.ResourceQuotaList) map[string]*projectQuota {
	projectQuotaMap := map[string]*projectQuota{}
	for _, project := range projects.Items {
		projectQuotaMap[project.Name] = &projectQuota{project: project}
	}

	for _, quota := range quotas.Items {
		if quota.Spec.Subject.Kind == kubermaticv1.ProjectSubjectKind {
			// prune projects with custom quotas
			if quota.Labels == nil || !(quota.Labels[DefaultProjectResourceQuotaKey] == DefaultProjectResourceQuotaValue) {
				delete(projectQuotaMap, quota.Spec.Subject.Name)
				continue
			}

			pQuota, ok := projectQuotaMap[quota.Spec.Subject.Name]
			if !ok {
				// skip if quota does not have project, should not happen but maybe in a race during deletion it could
				continue
			}
			pQuota.quota = &quota
		}
	}
	return projectQuotaMap
}

func (r *reconciler) ensureDefaultProjectQuota(ctx context.Context, settings *kubermaticv1.KubermaticSetting,
	project *kubermaticv1.Project, quota *kubermaticv1.ResourceQuota) error {
	// if missing, create
	if quota == nil {
		newQuota := genDefaultResourceQuota(settings, project)
		if err := r.masterClient.Create(ctx, newQuota); err != nil {
			return fmt.Errorf("error creating default resource quota: %w", err)
		}
	} else {
		quota.Spec.Quota = settings.Spec.DefaultProjectResourceQuota.Quota
		if err := r.masterClient.Update(ctx, quota); err != nil {
			return fmt.Errorf("error updating default resource quota: %w", err)
		}
	}
	return nil
}

func genDefaultResourceQuota(settings *kubermaticv1.KubermaticSetting, project *kubermaticv1.Project) *kubermaticv1.ResourceQuota {
	quota := &kubermaticv1.ResourceQuota{}
	quota.Labels = map[string]string{
		DefaultProjectResourceQuotaKey: DefaultProjectResourceQuotaValue,
	}
	quota.Spec.Subject = kubermaticv1.Subject{
		Name: project.Name,
		Kind: kubermaticv1.ProjectSubjectKind,
	}
	quota.Spec.Quota = settings.Spec.DefaultProjectResourceQuota.Quota
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
