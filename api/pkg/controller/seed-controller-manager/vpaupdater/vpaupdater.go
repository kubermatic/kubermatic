package vpaupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubermatic/kubermatic/api/pkg/controller/util"
)

const (
	ControllerName = "kubermatic_vpaupdater_controller"
	UpdatedByVPA   = "updated-by-vpa"
)

// Reconciler stores necessary components that are required to manage in-cluster Add-On's
type Reconciler struct {
	log        *zap.SugaredLogger
	workerName string
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

// Add creates a new Addon controller that is responsible for
// managing in-cluster addons
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		log:        log,
		Client:     client,
		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	objTypes := []runtime.Object{
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
		&autoscalingv1beta2.VerticalPodAutoscaler{},
	}
	for _, t := range objTypes {
		if err := c.Watch(&source.Kind{Type: t}, util.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		// If it's not a NotFound err, return it
		if !kerrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	log = r.log.With("cluster", cluster.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		kubermaticv1.ClusterConditionVPAUpdaterControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
			"%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	vpaList := &autoscalingv1beta2.VerticalPodAutoscalerList{}
	if err := r.Client.List(ctx, vpaList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
		return fmt.Errorf("failed to list VPAs: %v", err)
	}

	errors := []error{}
	for _, vpa := range vpaList.Items {
		switch vpa.Spec.TargetRef.Kind {
		case "StatefulSet":
			if err := r.updateStatefulSet(ctx, cluster.Status.NamespaceName, vpa.Spec.TargetRef.Name, vpa); err != nil {
				errors = append(errors, fmt.Errorf("failed to update the statefulset %s: %v", vpa.Spec.TargetRef.Name, err))
			}
		case "Deployment":
			if err := r.updateDeployment(ctx, cluster.Status.NamespaceName, vpa.Spec.TargetRef.Name, vpa); err != nil {
				errors = append(errors, fmt.Errorf("failed to update the deployment %s: %v", vpa.Spec.TargetRef.Name, err))
			}
		default:
			errors = append(errors, fmt.Errorf("encoutered unknown object kind %q", vpa.Spec.TargetRef.Kind))
		}
	}

	return utilerrors.NewAggregate(errors)
}

func (r *Reconciler) updateStatefulSet(ctx context.Context, namespaceName, statefulSetName string, vpa autoscalingv1beta2.VerticalPodAutoscaler) error {
	// TODO(xmudrii): Check which check we need.
	// VPA recommendations are not available for recently created resources and we want to skip those resources.
	if vpa.Status.Recommendation == nil || vpa.Status.Recommendation.ContainerRecommendations == nil {
		return nil
	}

	statefulSet := &appsv1.StatefulSet{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: statefulSetName, Namespace: namespaceName}, statefulSet); err != nil {
		return fmt.Errorf("failed to get statefulset %s/%s: %v", namespaceName, statefulSetName, err)
	}

	// TODO(xmudrii): Reconsider this logic.
	var lastUpdated *time.Time
	if a, ok := statefulSet.Annotations[UpdatedByVPA]; ok {
		req := []resources.Requirements{}
		err := json.Unmarshal([]byte(a), &req)
		if err != nil {
			return err
		}
		if len(req) > 0 {
			t, err := time.Parse(time.RFC3339, req[0].LastUpdated)
			if err != nil {
				return err
			}
			lastUpdated = &t
		}
	}

	requirements, updated := updatePodSpec(&statefulSet.Spec.Template.Spec, vpa.Status.Recommendation.ContainerRecommendations, lastUpdated)
	if updated {
		if statefulSet.ObjectMeta.Annotations == nil {
			statefulSet.ObjectMeta.Annotations = map[string]string{}
		}
		rb, err := json.Marshal(requirements)
		if err != nil {
			return fmt.Errorf("failed to marshal resource requirementes for statefulset %s/%s: %v", namespaceName, statefulSetName, err)
		}
		statefulSet.ObjectMeta.Annotations[UpdatedByVPA] = string(rb)

		if err := r.Client.Update(ctx, statefulSet); err != nil {
			return fmt.Errorf("failed to update statefulset %s/%s: %v", namespaceName, statefulSetName, err)
		}
	}

	return nil
}

func (r *Reconciler) updateDeployment(ctx context.Context, namespaceName, deploymentName string, vpa autoscalingv1beta2.VerticalPodAutoscaler) error {
	// TODO(xmudrii): Check which check we need.
	// VPA recommendations are not available for recently created resources and we want to skip those resources.
	if vpa.Status.Recommendation == nil || vpa.Status.Recommendation.ContainerRecommendations == nil {
		return nil
	}

	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespaceName}, deployment); err != nil {
		return fmt.Errorf("failed to get deployment %s/%s: %v", namespaceName, deploymentName, err)
	}

	// TODO(xmudrii): Reconsider this logic.
	var lastUpdated *time.Time
	if a, ok := deployment.Annotations[UpdatedByVPA]; ok {
		req := []resources.Requirements{}
		err := json.Unmarshal([]byte(a), &req)
		if err != nil {
			return err
		}
		if len(req) > 0 {
			t, err := time.Parse(time.RFC3339, req[0].LastUpdated)
			if err != nil {
				return err
			}
			lastUpdated = &t
		}
	}

	requirements, updated := updatePodSpec(&deployment.Spec.Template.Spec, vpa.Status.Recommendation.ContainerRecommendations, lastUpdated)
	if updated {
		if deployment.ObjectMeta.Annotations == nil {
			deployment.ObjectMeta.Annotations = map[string]string{}
		}
		rb, err := json.Marshal(requirements)
		if err != nil {
			return fmt.Errorf("failed to marshal resource requirementes for deployment %s/%s: %v", namespaceName, deploymentName, err)
		}
		deployment.ObjectMeta.Annotations[UpdatedByVPA] = string(rb)

		if err := r.Client.Update(ctx, deployment); err != nil {
			return fmt.Errorf("failed to update deployment %s/%s: %v", namespaceName, deploymentName, err)
		}
	}

	return nil
}

func updatePodSpec(podSpec *corev1.PodSpec, containerRecommendations []autoscalingv1beta2.RecommendedContainerResources, lastUpdated *time.Time) ([]resources.Requirements, bool) {
	podsRequirements := []resources.Requirements{}
	updated := false
	for _, cr := range containerRecommendations {
		for i, c := range podSpec.Containers {
			if c.Name == cr.ContainerName {
				requirements := corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: *cr.Target.Memory(),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: *cr.UpperBound.Memory(),
					},
				}
				podRequirements := resources.Requirements{
					Name:        podSpec.Containers[i].Name,
					Requires:    &requirements,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
				podsRequirements = append(podsRequirements, podRequirements)

				scaleUp := false
				scaleDown := false

				var valOld float64
				if podSpec.Containers[i].Resources.Requests.Memory() != nil {
					valOld = float64(podSpec.Containers[i].Resources.Requests.Memory().MilliValue())
				}
				valNew := float64(requirements.Requests.Memory().MilliValue())
				if valNew >= valOld*1.15 {
					scaleUp = true
				} else if valNew <= valOld*0.7 {
					scaleDown = true
				}

				valOld = 0
				if podSpec.Containers[i].Resources.Limits.Memory() != nil {
					valOld = float64(podSpec.Containers[i].Resources.Limits.Memory().MilliValue())
				}
				valNew = float64(requirements.Limits.Memory().MilliValue())
				if valNew >= valOld*1.15 {
					scaleUp = true
				} else if valNew <= valOld*0.7 {
					scaleDown = true
				}

				if lastUpdated == nil {
					podSpec.Containers[i].Resources = requirements
					updated = true
					continue
				}

				if scaleUp && time.Since(*lastUpdated).Seconds() >= 30 {
					podSpec.Containers[i].Resources = requirements
					updated = true
				} else if scaleDown && time.Since(*lastUpdated).Minutes() >= 5 {
					podSpec.Containers[i].Resources = requirements
					updated = true
				}
			}
		}
	}
	return podsRequirements, updated
}
