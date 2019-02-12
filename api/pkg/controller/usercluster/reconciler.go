package usercluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	debugLevel = 4
)

// reconciler creates and deletes Kubernetes resources to achieve the desired state
type reconciler struct {
	ctx    context.Context
	client controllerclient.Client
}

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) Reconcile(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	if len(c.Spec.MachineNetworks) > 0 {
		if err := r.ensureInitializerConfiguration(c, data); err != nil {
			return err
		}
	}

	if err := r.ensureRoles(data); err != nil {
		return err
	}

	if err := r.ensureRoleBindings(data); err != nil {
		return err
	}

	if err := r.ensureConfigMaps(data); err != nil {
		return err
	}

	if err := r.ensureClusterRoles(c, data); err != nil {
		return err
	}

	if err := r.ensureClusterRoleBindings(c, data); err != nil {
		return err
	}

	if err := r.ensureMutatingWebhookConfigurations(c, data); err != nil {
		return err
	}

	if err := r.ensureCustomResourceDefinitions(c); err != nil {
		return err
	}

	if err := r.ensureAPIServices(); err != nil {
		return err
	}

	if err := r.ensureServices(data); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ensureRoles(data *resources.TemplateData) error {
	creators := []resources.RoleCreator{
		machinecontroller.EndpointReaderRole,
		machinecontroller.KubeSystemRole,
		machinecontroller.KubePublicRole,
		machinecontroller.ClusterInfoReaderRole,
	}

	for _, create := range creators {
		role, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}

		existing := &rbacv1.Role{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: role.Namespace, Name: role.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, role); err != nil {
					return fmt.Errorf("failed to create Role %s in namespace %s: %v", role.Name, role.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new Role %s in namespace %s", role.Name, role.Namespace)
				return nil
			}
			return err
		}

		role, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}
		if resources.DeepEqual(role, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, role); err != nil {
			return fmt.Errorf("failed to update Role %s in namespace %s: %v", role.Name, role.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated Role %s in namespace %s", role.Name, role.Namespace)
	}

	return nil
}

func (r *reconciler) ensureRoleBindings(data *resources.TemplateData) error {
	creators := []resources.RoleBindingCreator{
		machinecontroller.DefaultRoleBinding,
		machinecontroller.KubeSystemRoleBinding,
		machinecontroller.KubePublicRoleBinding,
		machinecontroller.ClusterInfoAnonymousRoleBinding,
		metricsserver.RolebindingAuthReader,
		scheduler.RoleBindingAuthDelegator,
		controllermanager.RoleBindingAuthDelegator,
	}

	for _, create := range creators {
		rb, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		existing := &rbacv1.RoleBinding{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: rb.Namespace, Name: rb.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, rb); err != nil {
					return fmt.Errorf("failed to create RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new RoleBinding %s in namespace %s", rb.Name, rb.Namespace)
				return nil
			}
			return err
		}

		rb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding : %v", err)
		}
		if equality.Semantic.DeepEqual(rb, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, rb); err != nil {
			return fmt.Errorf("failed to update RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated RoleBinding %s in namespace %s", rb.Name, rb.Namespace)
	}

	return nil
}

func (r *reconciler) ensureConfigMaps(data *resources.TemplateData) error {
	creators := []resources.ConfigMapCreator{
		openvpn.ClientConfigConfigMapCreator(data),
		machinecontroller.ClusterInfoConfigMapCreator(data),
	}

	for _, create := range creators {
		cm, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		existing := &corev1.ConfigMap{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: cm.Namespace, Name: cm.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, cm); err != nil {
					return fmt.Errorf("failed to create ConfigMap %s in namespace %s: %v", cm.Name, cm.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new ConfigMap %s in namespace %s", cm.Name, cm.Namespace)
				return nil
			}
			return err
		}

		cm, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}
		if equality.Semantic.DeepEqual(cm, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s in namespace %s: %v", cm.Name, cm.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated ConfigMap %s in namespace %s", cm.Name, cm.Namespace)
	}

	return nil
}

// GetUserClusterRoleCreators returns a list of GetUserClusterRoleCreators
func GetUserClusterRoleCreators(c *kubermaticv1.Cluster) []resources.ClusterRoleCreator {
	creators := []resources.ClusterRoleCreator{
		machinecontroller.ClusterRole,
		vpnsidecar.DnatControllerClusterRole,
		metricsserver.ClusterRole,
	}

	if len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRole)
	}

	return creators
}

func (r *reconciler) ensureClusterRoles(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetUserClusterRoleCreators(c)

	for _, create := range creators {
		cRole, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		existing := &rbacv1.ClusterRole{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: cRole.Namespace, Name: cRole.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, cRole); err != nil {
					return fmt.Errorf("failed to create ClusterRole %s in namespace %s: %v", cRole.Name, cRole.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new ClusterRole %s in namespace %s", cRole.Name, cRole.Namespace)
				return nil
			}
			return err
		}

		cRole, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole : %v", err)
		}
		if equality.Semantic.DeepEqual(cRole, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, cRole); err != nil {
			return fmt.Errorf("failed to update ClusterRole %s in namespace %s: %v", cRole.Name, cRole.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated ClusterRole %s in namespace %s", cRole.Name, cRole.Namespace)
	}

	return nil
}

// GetUserClusterRoleBindingCreators returns a list of ClusterRoleBindingCreators which should be used to create user cluster role bindings.
func GetUserClusterRoleBindingCreators(c *kubermaticv1.Cluster) []resources.ClusterRoleBindingCreator {
	creators := []resources.ClusterRoleBindingCreator{
		machinecontroller.ClusterRoleBinding,
		machinecontroller.NodeBootstrapperClusterRoleBinding,
		machinecontroller.NodeSignerClusterRoleBinding,
		vpnsidecar.DnatControllerClusterRoleBinding,
		metricsserver.ClusterRoleBindingResourceReader,
		metricsserver.ClusterRoleBindingAuthDelegator,
		scheduler.ClusterRoleBindingAuthDelegator,
		controllermanager.ClusterRoleBindingAuthDelegator,
	}

	if len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRoleBinding)
	}

	return creators
}

func (r *reconciler) ensureClusterRoleBindings(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetUserClusterRoleBindingCreators(c)

	for _, create := range creators {
		crb, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		existing := &rbacv1.ClusterRoleBinding{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: crb.Namespace, Name: crb.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, crb); err != nil {
					return fmt.Errorf("failed to create ClusterRoleBinding %s in namespace %s: %v", crb.Name, crb.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new ClusterRoleBinding %s in namespace %s", crb.Name, crb.Namespace)
				return nil
			}
			return err
		}

		crb, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding : %v", err)
		}
		if equality.Semantic.DeepEqual(crb, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s in namespace %s: %v", crb.Name, crb.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated ClusterRoleBinding %s in namespace %s", crb.Name, crb.Namespace)
	}

	return nil
}

// GetUserClusterMutatingWebhookConfigurationCreators returns all UserClusterMutatingWebhookConfigurationCreators
func GetUserClusterMutatingWebhookConfigurationCreators() []resources.MutatingWebhookConfigurationCreator {
	return []resources.MutatingWebhookConfigurationCreator{
		machinecontroller.MutatingwebhookConfiguration,
	}
}

func (r *reconciler) ensureMutatingWebhookConfigurations(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetUserClusterMutatingWebhookConfigurationCreators()
	for _, create := range creators {
		mutatingWebhookConfiguration, err := create(c, data, nil)
		if err != nil {
			return fmt.Errorf("failed to build MutatingwebhookConfiguration: %v", err)
		}

		existing := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: mutatingWebhookConfiguration.Namespace, Name: mutatingWebhookConfiguration.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, mutatingWebhookConfiguration); err != nil {
					return fmt.Errorf("failed to create MutatingWebhookConfiguration %s in namespace %s: %v", mutatingWebhookConfiguration.Name, mutatingWebhookConfiguration.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new MutatingWebhookConfiguration %s in namespace %s", mutatingWebhookConfiguration.Name, mutatingWebhookConfiguration.Namespace)
				return nil
			}
			return err
		}

		mutatingWebhookConfiguration, err = create(c, data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build MutatingWebhookConfiguration : %v", err)
		}
		if equality.Semantic.DeepEqual(mutatingWebhookConfiguration, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, mutatingWebhookConfiguration); err != nil {
			return fmt.Errorf("failed to update MutatingWebhookConfiguration %s in namespace %s: %v", mutatingWebhookConfiguration.Name, mutatingWebhookConfiguration.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated MutatingWebhookConfiguration %s in namespace %s", mutatingWebhookConfiguration.Name, mutatingWebhookConfiguration.Namespace)
	}

	return nil
}

// GetCRDCreators reuturns a list of CRDCreateors
func GetCRDCreators() []resources.CRDCreateor {
	return []resources.CRDCreateor{
		machinecontroller.MachineCRD,
		machinecontroller.MachineSetCRD,
		machinecontroller.MachineDeploymentCRD,
		machinecontroller.ClusterCRD,
	}
}

func (r *reconciler) ensureCustomResourceDefinitions(c *kubermaticv1.Cluster) error {
	creators := GetCRDCreators()

	for _, create := range creators {
		crd, err := create(c.Spec.Version, nil)
		if err != nil {
			return fmt.Errorf("failed to build CustomResourceDefinitions: %v", err)
		}

		existing := &apiextensionsv1beta1.CustomResourceDefinition{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: crd.Namespace, Name: crd.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, crd); err != nil {
					return fmt.Errorf("failed to create CustomResourceDefinition %s in namespace %s: %v", crd.Name, crd.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new CustomResourceDefinition %s in namespace %s", crd.Name, crd.Namespace)
				return nil
			}
			return err
		}

		crd, err = create(c.Spec.Version, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build CustomResourceDefinition : %v", err)
		}
		if equality.Semantic.DeepEqual(crd, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, crd); err != nil {
			return fmt.Errorf("failed to update CustomResourceDefinition %s in namespace %s: %v", crd.Name, crd.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated CustomResourceDefinition %s in namespace %s", crd.Name, crd.Namespace)
	}

	return nil
}

// GetAPIServiceCreators returns a list of APIServiceCreator
func GetAPIServiceCreators() []resources.APIServiceCreator {
	return []resources.APIServiceCreator{
		metricsserver.APIService,
	}
}

func (r *reconciler) ensureAPIServices() error {
	creators := GetAPIServiceCreators()

	for _, create := range creators {
		apiService, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build APIService: %v", err)
		}

		existing := &apiregistrationv1beta1.APIService{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: apiService.Namespace, Name: apiService.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, apiService); err != nil {
					return fmt.Errorf("failed to create APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new APIService %s in namespace %s", apiService.Name, apiService.Namespace)
				return nil
			}
			return err
		}

		apiService, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build APIService : %v", err)
		}
		if equality.Semantic.DeepEqual(apiService, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, apiService); err != nil {
			return fmt.Errorf("failed to update APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated APIService %s in namespace %s", apiService.Name, apiService.Namespace)
	}

	return nil
}

// GetUserClusterServiceCreators returns a list of ServiceCreator's used for the user cluster
func GetUserClusterServiceCreators(data resources.ServiceDataProvider) []resources.ServiceCreator {
	return []resources.ServiceCreator{
		metricsserver.ExternalNameServiceCreator(data),
	}
}

func (r *reconciler) ensureServices(data *resources.TemplateData) error {
	creators := GetUserClusterServiceCreators(data)

	for _, create := range creators {
		service, err := create(&corev1.Service{})
		if err != nil {
			return fmt.Errorf("failed to build Service: %v", err)
		}

		existing := &corev1.Service{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: service.Namespace, Name: service.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, service); err != nil {
					return fmt.Errorf("failed to create Service %s in namespace %s: %v", service.Name, service.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new Service %s in namespace %s", service.Name, service.Namespace)
				return nil
			}
			return err
		}

		service, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Service : %v", err)
		}
		if equality.Semantic.DeepEqual(service, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, service); err != nil {
			return fmt.Errorf("failed to update Service %s in namespace %s: %v", service.Name, service.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated Service %s in namespace %s", service.Name, service.Namespace)
	}

	return nil
}

func (r *reconciler) ensureInitializerConfiguration(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []resources.InitializerConfigurationCreator{
		ipamcontroller.MachineIPAMInitializerConfiguration,
	}

	for _, create := range creators {
		initializerConfiguration, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build InitializerConfiguration: %v", err)
		}

		existing := &admissionv1alpha1.InitializerConfiguration{}
		err = r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: initializerConfiguration.Namespace, Name: initializerConfiguration.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(r.ctx, initializerConfiguration); err != nil {
					return fmt.Errorf("failed to create InitializerConfiguration %s in namespace %s: %v", initializerConfiguration.Name, initializerConfiguration.Namespace, err)
				}
				glog.V(debugLevel).Infof("created a new InitializerConfiguration %s in namespace %s", initializerConfiguration.Name, initializerConfiguration.Namespace)
				return nil
			}
			return err
		}

		initializerConfiguration, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build InitializerConfiguration : %v", err)
		}
		if equality.Semantic.DeepEqual(initializerConfiguration, existing) {
			continue
		}

		if err := r.client.Update(r.ctx, initializerConfiguration); err != nil {
			return fmt.Errorf("failed to update InitializerConfiguration %s in namespace %s: %v", initializerConfiguration.Name, initializerConfiguration.Namespace, err)
		}
		glog.V(debugLevel).Infof("updated InitializerConfiguration %s in namespace %s", initializerConfiguration.Name, initializerConfiguration.Namespace)
	}

	return nil
}
