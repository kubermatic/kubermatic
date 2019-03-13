package serviceaccount

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kubermatic_serviceaccount_projectbinding_controller"

	labelGroup = "group"
)

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
	glog.V(4).Infof("Reconciling SA binding for %s", saName)

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

	project := &kubermaticv1.Project{}
	if err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: projectName}, project); err != nil {
		return err
	}

	binding := &kubermaticv1.UserProjectBinding{}
	bindingName := genBindingName(projectName, sa.Spec.ID)
	if err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: bindingName}, binding); err != nil {
		if kerrors.IsNotFound(err) {
			return r.createBinding(sa, project)
		}
		return err
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) createBinding(user *kubermaticv1.User, project *kubermaticv1.Project) error {
	group, ok := user.Labels[labelGroup]
	if !ok {
		return fmt.Errorf("label %s not found", labelGroup)
	}
	binding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        user.GetUID(),
					Name:       user.Name,
				},
			},
			Name:   genBindingName(project.Name, user.Spec.ID),
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: project.Name},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: project.Name,
			UserEmail: user.Spec.Email,
			Group:     group,
		},
	}

	if err := r.Create(r.ctx, binding); err != nil {
		return err
	}

	return nil
}

func genBindingName(projectName, userID string) string {
	return fmt.Sprintf("sa-%s-%s", projectName, userID)
}
