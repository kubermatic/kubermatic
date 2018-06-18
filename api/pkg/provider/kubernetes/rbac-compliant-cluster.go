package kubernetes

import (
	"errors"
	"strings"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NewRBACCompliantClusterProvider returns a new cluster provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewRBACCompliantClusterProvider(
	createSeedImpersonatedClient kubermaticImpersonationClient,
	seedPrivilegedClient kubermaticclientset.Interface,
	userClusterConnProvider UserClusterConnectionProvider,
	clusterLister kubermaticv1lister.ClusterLister,
	workerName string) (*RBACCompliantClusterProvider, error) {
	return &RBACCompliantClusterProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		seedPrivilegedClient:         seedPrivilegedClient,
		userClusterConnProvider:      userClusterConnProvider,
		clusterLister:                clusterLister,
		workerName:                   workerName,
	}, nil
}

// RBACCompliantClusterProvider struct that holds required components in order to provide
// cluster provided that is RBAC compliant
type RBACCompliantClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient kubermaticImpersonationClient

	// userClusterConnProvider used for obtaining a connection to the client's cluster
	userClusterConnProvider UserClusterConnectionProvider

	// clusterLister provide access to local cache that stores cluster objects
	clusterLister kubermaticv1lister.ClusterLister

	// seedPrivilegedClient a privileged client connection used for creating addons only
	seedPrivilegedClient kubermaticclientset.Interface

	workerName string
}

// New creates a brand new cluster that is bound to the given project
func (p *RBACCompliantClusterProvider) New(project *kubermaticapiv1.Project, user *kubermaticapiv1.User, spec *kubermaticapiv1.ClusterSpec) (*kubermaticapiv1.Cluster, error) {
	if project == nil || user == nil || spec == nil {
		return nil, errors.New("project and/or user and/or spec is missing but required")
	}
	spec.HumanReadableName = strings.TrimSpace(spec.HumanReadableName)
	spec.WorkerName = p.workerName
	clusters, err := p.List(project)
	if err != nil {
		return nil, err
	}
	for _, c := range clusters {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, kerrors.NewAlreadyExists(schema.GroupResource{Group: kubermaticapiv1.SchemeGroupVersion.Group, Resource: kubermaticapiv1.ClusterResourceName}, spec.HumanReadableName)
		}
	}

	name := rand.String(10)
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kubermaticapiv1.WorkerNameLabelKey: p.workerName,
			},
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
		},
		Spec: *spec,
		Status: kubermaticapiv1.ClusterStatus{
			UserEmail:     user.Spec.Email,
			UserName:      user.Name,
			NamespaceName: NamespaceName(name),
		},
		Address: kubermaticapiv1.ClusterAddress{},
	}

	seedImpersonatedClient, err := p.createSeedImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}
	cluster, err = seedImpersonatedClient.Clusters().Create(cluster)
	if err != nil {
		return nil, err
	}

	// TODO: Make Addons to be part of the cluster specification
	//       For more details see: https://github.com/kubermatic/kubermatic/issues/1211
	//
	// TODO: Add RBAC Roles to `Addons` resources
	//       For more details see: https://github.com/kubermatic/kubermatic/issues/1181
	addons := []string{
		"canal",
		"dashboard",
		"dns",
		"heapster",
		"kube-proxy",
		"openvpn",
		"rbac",
	}
	gv := kubermaticapiv1.SchemeGroupVersion
	ownerRef := *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
	for _, addon := range addons {
		_, err = p.seedPrivilegedClient.KubermaticV1().Addons(cluster.Status.NamespaceName).Create(&kubermaticapiv1.Addon{
			ObjectMeta: metav1.ObjectMeta{
				Name:            addon,
				Namespace:       cluster.Status.NamespaceName,
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
			Spec: kubermaticapiv1.AddonSpec{
				Name: addon,
				Cluster: corev1.ObjectReference{
					Name:       cluster.Name,
					Namespace:  "",
					UID:        cluster.UID,
					APIVersion: cluster.APIVersion,
					Kind:       "Cluster",
				},
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// List gets all clusters that belong to the given project
//
// Note:
// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *RBACCompliantClusterProvider) List(project *kubermaticapiv1.Project) ([]*kubermaticapiv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}
	clusters, err := p.clusterLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	projectClusters := []*kubermaticapiv1.Cluster{}
	for _, cluster := range clusters {
		owners := cluster.GetOwnerReferences()
		for _, owner := range owners {
			if owner.APIVersion == kubermaticapiv1.SchemeGroupVersion.String() && owner.Kind == kubermaticapiv1.ProjectKindName && owner.Name == project.Name {
				projectClusters = append(projectClusters, cluster)
			}
		}
	}

	return projectClusters, nil
}

// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
func (p *RBACCompliantClusterProvider) Get(user *kubermaticapiv1.User, project *kubermaticapiv1.Project, clusterName string) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := p.createSeedImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}
	return seedImpersonatedClient.Clusters().Get(clusterName, metav1.GetOptions{})
}

// Delete deletes the given cluster
func (p *RBACCompliantClusterProvider) Delete(user *kubermaticapiv1.User, project *kubermaticapiv1.Project, clusterName string) error {
	seedImpersonatedClient, err := p.createSeedImpersonationClientWrapper(user, project)
	if err != nil {
		return err
	}

	return seedImpersonatedClient.Clusters().Delete(clusterName, &metav1.DeleteOptions{})
}

// Update updates a cluster
func (p *RBACCompliantClusterProvider) Update(user *kubermaticapiv1.User, project *kubermaticapiv1.Project, newCluster *kubermaticapiv1.Cluster) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := p.createSeedImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}

	return seedImpersonatedClient.Clusters().Update(newCluster)
}

// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
func (p *RBACCompliantClusterProvider) GetAdminKubeconfigForCustomerCluster(c *kubermaticapiv1.Cluster) (*clientcmdapi.Config, error) {
	b, err := p.userClusterConnProvider.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(b)
}

// GetMachineClientForCustomerCluster returns a client to interact with machine resources in the given cluster
//
// Note that the client you will get has admin privileges
func (p *RBACCompliantClusterProvider) GetMachineClientForCustomerCluster(c *kubermaticapiv1.Cluster) (machineclientset.Interface, error) {
	return p.userClusterConnProvider.GetMachineClient(c)
}

// GetClientForCustomerCluster returns a client to interact with the given cluster
//
// Note that the client you will get has admin privileges
func (p *RBACCompliantClusterProvider) GetClientForCustomerCluster(c *kubermaticapiv1.Cluster) (kubernetes.Interface, error) {
	return p.userClusterConnProvider.GetClient(c)
}

// createSeedImpersonationClientWrapper is a helper method that spits back kubermatic client that uses user impersonation
func (p *RBACCompliantClusterProvider) createSeedImpersonationClientWrapper(user *kubermaticapiv1.User, project *kubermaticapiv1.Project) (kubermaticclientv1.KubermaticV1Interface, error) {
	if user == nil || project == nil {
		return nil, errors.New("user and/or project is missing but required")
	}
	groupName, err := user.GroupForProject(project.Name)
	if err != nil {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, project.Name, err)
	}
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: user.Spec.Email,
		Groups:   []string{groupName},
	}
	return p.createSeedImpersonatedClient(impersonationCfg)
}
