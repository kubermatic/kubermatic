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

package resource_usage_controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	machinevalidation "k8c.io/kubermatic/v2/pkg/ee/validation/machine"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "resource_usage_controller"

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	clusterName     string
	caBundle        *certificates.CABundle
	recorder        record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterName string, caBundle *certificates.CABundle,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		clusterName:     clusterName,
		caBundle:        caBundle,
		recorder:        userMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
	}
	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	// Watch for changes to Machines
	if err = c.Watch(
		&source.Kind{Type: &clusterv1alpha1.Machine{}}, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(metav1.NamespaceSystem)); err != nil {
		return fmt.Errorf("failed to establish watch for Machines: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("resource", request)
	log.Debug("reconciling")

	machines := &clusterv1alpha1.MachineList{}
	if err := r.userClient.List(ctx, machines); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get machines: %w", err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err = r.seedClient.Get(ctx, types.NamespacedName{
		Name: r.clusterName,
	}, cluster); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	err = r.reconcile(ctx, cluster, machines)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ClusterResourceUsageReconcileFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, machines *clusterv1alpha1.MachineList) error {
	resourceUsage := kubermaticv1.NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	for _, machine := range machines.Items {
		resourceDetails, err := machinevalidation.GetMachineResourceUsage(ctx, r.userClient, &machine, r.caBundle.CertPool())
		if err != nil {
			return fmt.Errorf("error getting machine resource usage for machine %q: %w", machine.Name, err)
		}

		resourceUsage.CPU.Add(*resourceDetails.Cpu())
		resourceUsage.Memory.Add(*resourceDetails.Memory())
		resourceUsage.Storage.Add(*resourceDetails.Storage())
	}

	cluster.Status.ResourceUsage = resourceUsage

	return kubermaticv1helper.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ResourceUsage = resourceUsage
	})
}
