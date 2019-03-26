package serviceaccount

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	serviceaccount "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "kubermatic_serviceaccount_projectbinding_controller"

func Add(mgr manager.Manager) error {
	r := &reconcileServiceAccountProjectBinding{Client: mgr.GetClient(), ctx: context.TODO()}
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to User
	err = c.Watch(&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// reconcileServiceAccountProjectBinding reconciles User objects
type reconcileServiceAccountProjectBinding struct {
	ctx context.Context
	client.Client
}

func (r *reconcileServiceAccountProjectBinding) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	resourceName := request.Name
	if !strings.HasPrefix(resourceName, "serviceaccount") {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.ensureServiceAccountProjectBinding(resourceName)
}

func (r *reconcileServiceAccountProjectBinding) ensureServiceAccountProjectBinding(saName string) error {
	sa := &kubermaticv1.User{}
	if err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: saName}, sa); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// get project name from Owner Reference or from label
	projectName := ""
	for _, owner := range sa.GetOwnerReferences() {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			projectName = owner.Name
			break
		}
	}
	if len(projectName) == 0 {
		projectName = sa.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return fmt.Errorf("unable to find owing project for the service account = %s", sa.Name)
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, projectName))
	if err != nil {
		return err
	}

	bindings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(r.ctx, &client.ListOptions{LabelSelector: labelSelector}, bindings); err != nil {
		return err
	}

	bindingExist := false
	for _, binding := range bindings.Items {
		if binding.Spec.ProjectID == projectName && strings.EqualFold(binding.Spec.UserEmail, sa.Spec.Email) {
			bindingExist = true
			break
		}
	}
	if !bindingExist {
		if err := r.createBinding(sa, projectName); err != nil {
			return err
		}
	}
	// remove labelGroup from sa
	if _, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]; ok {
		delete(sa.Labels, serviceaccount.ServiceAccountLabelGroup)
		if err := r.Update(r.ctx, sa); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) createBinding(sa *kubermaticv1.User, projectName string) error {
	group, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]
	if !ok {
		return fmt.Errorf("label %s not found for sa %s", serviceaccount.ServiceAccountLabelGroup, sa.Name)
	}

	binding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        sa.GetUID(),
					Name:       sa.Name,
				},
			},
			Name:   rand.String(10),
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: projectName,
			UserEmail: sa.Spec.Email,
			Group:     group,
		},
	}

	return r.Create(r.ctx, binding)
}
