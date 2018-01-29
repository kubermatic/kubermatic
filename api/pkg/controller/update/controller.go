package update

import (
	"fmt"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	seedcrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	kubernetesProvider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientkubernetes "k8s.io/client-go/kubernetes"
)

// Interface is a interface for a update controller
type Interface interface {
	Sync(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)
}

//SeedClientProvider offers functions to get resources of a seed-cluster
type SeedClientProvider interface {
	GetClient(dc string) (clientkubernetes.Interface, error)
	GetCRDClient(dc string) (seedcrdclient.Interface, error)
	GetInformerGroup(dc string) (*seedinformer.Group, error)
}

// New returns a update controller
func New(
	clientProvider SeedClientProvider,
	masterResourcesPath string,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
) Interface {
	return &controller{
		clientProvider:      clientProvider,
		masterResourcesPath: masterResourcesPath,
		versions:            versions,
		updates:             updates,
	}
}

// Controller represents an update controller
type controller struct {
	clientProvider      SeedClientProvider
	masterResourcesPath string
	versions            map[string]*apiv1.MasterVersion
	updates             []apiv1.MasterUpdate
}

// Sync determines the current update state, and advances to the next phase as required
func (u *controller) Sync(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	v, found := u.versions[c.Spec.MasterVersion]
	if !found {
		return nil, fmt.Errorf("unknown target master version %q", c.Spec.MasterVersion)
	}

	switch c.Status.MasterUpdatePhase {
	case kubermaticv1.StartMasterUpdatePhase:
		return u.updateDeployment(c, []string{v.EtcdOperatorDeploymentYaml}, v, kubermaticv1.EtcdOperatorUpdatePhase)
	case kubermaticv1.EtcdOperatorUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"etcd-operator"}, kubermaticv1.StartMasterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateEtcdCluster(c, []string{v.EtcdClusterYaml}, v, kubermaticv1.EtcdClusterUpdatePhase)
	case kubermaticv1.EtcdClusterUpdatePhase:
		c, ready, err := u.waitForEtcdCluster(c, []string{"etcd-cluster"}, kubermaticv1.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateDeployment(c, []string{v.ApiserverDeploymentYaml}, v, kubermaticv1.APIServerMasterUpdatePhase)
	case kubermaticv1.APIServerMasterUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"apiserver"}, kubermaticv1.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateDeployment(c, []string{v.ControllerDeploymentYaml, v.SchedulerDeploymentYaml, v.AddonManagerDeploymentYaml, v.NodeControllerDeploymentYaml}, v, kubermaticv1.ControllersMasterUpdatePhase)
	case kubermaticv1.ControllersMasterUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"controller-manager", "scheduler", "addon-manager", "node-controller"}, kubermaticv1.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}

		c.Status.MasterUpdatePhase = ""
		c.Status.Phase = kubermaticv1.LaunchingClusterStatusPhase
		c.Status.LastTransitionTime = metav1.Now()
		return c, nil
	}

	// this should never happen: we don't know the phase
	glog.Errorf("Unknown cluster %q update phase: %v", c.Name, c.Status.MasterUpdatePhase)
	c.Status.MasterUpdatePhase = ""
	c.Status.Phase = kubermaticv1.RunningClusterStatusPhase
	c.Status.LastTransitionTime = metav1.Now()
	return c, nil
}

func (u *controller) updateDeployment(c *kubermaticv1.Cluster, yamlFiles []string, masterVersion *apiv1.MasterVersion, nextPhase kubermaticv1.MasterUpdatePhase) (*kubermaticv1.Cluster, error) {
	client, err := u.clientProvider.GetClient(c.Spec.SeedDatacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for dc %q: %v", c.Spec.SeedDatacenterName, err)
	}

	for _, yamlFile := range yamlFiles {
		dep, err := resources.LoadDeploymentFile(c, masterVersion, u.masterResourcesPath, yamlFile)
		if err != nil {
			return nil, err
		}

		ns := kubernetesProvider.NamespaceName(c.Name)
		_, err = client.ExtensionsV1beta1().Deployments(ns).Update(dep)
		if errors.IsNotFound(err) {
			glog.Errorf("expected an %s deployment, but didn't find any for cluster %v. Creating a new one.", dep.Name, c.Name)
			_, err = client.ExtensionsV1beta1().Deployments(ns).Create(dep)
			if err != nil {
				return nil, fmt.Errorf("failed to re-create deployment for %s: %v", dep.Name, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to update deployment for %s: %v", dep.Name, err)
		}
	}

	c.Status.MasterUpdatePhase = nextPhase
	return c, nil
}

func (u *controller) updateEtcdCluster(c *kubermaticv1.Cluster, yamlFiles []string, masterVersion *apiv1.MasterVersion, nextPhase kubermaticv1.MasterUpdatePhase) (*kubermaticv1.Cluster, error) {
	client, err := u.clientProvider.GetCRDClient(c.Spec.SeedDatacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for dc %q: %v", c.Spec.SeedDatacenterName, err)
	}

	for _, yamlFile := range yamlFiles {
		newEtcd, err := resources.LoadEtcdClusterFile(masterVersion, u.masterResourcesPath, yamlFile)
		if err != nil {
			return nil, err
		}
		ns := fmt.Sprintf("cluster-%s", c.Name)
		var oldEtcd *etcdoperatorv1beta2.EtcdCluster
		oldEtcd, err = client.EtcdV1beta2().EtcdClusters(ns).Get("etcd-cluster", metav1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("failed to get current etcd cluster for %s: %v", newEtcd.ObjectMeta.Name, err)
			glog.Error(err)
			return nil, err
		}

		oldEtcd.Spec.Version = newEtcd.Spec.Version
		_, err = client.EtcdV1beta2().EtcdClusters(ns).Update(oldEtcd)
		if err != nil {
			err = fmt.Errorf("failed to update etcd cluster for %s: %v", newEtcd.ObjectMeta.Name, err)
			glog.Error(err)
			return nil, err
		}
	}
	c.Status.MasterUpdatePhase = nextPhase
	return c, nil
}

func (u *controller) waitForEtcdCluster(c *kubermaticv1.Cluster, names []string, fallbackPhase kubermaticv1.MasterUpdatePhase) (*kubermaticv1.Cluster, bool, error) {
	informerGroup, err := u.clientProvider.GetInformerGroup(c.Spec.SeedDatacenterName)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get informer group for dc %q: %v", c.Spec.SeedDatacenterName, err)
	}

	ns := kubernetesProvider.NamespaceName(c.Name)
	for _, name := range names {
		etcd, err := informerGroup.EtcdClusterInformer.Lister().EtcdClusters(ns).Get(name)
		if err != nil {
			return nil, false, err
		}
		//Ensure the etcd quorum
		if etcd.Spec.Size/2+1 >= etcd.Status.Size {
			return nil, false, nil
		}
	}
	return c, true, nil
}

func (u *controller) waitForDeployments(c *kubermaticv1.Cluster, names []string, fallbackPhase kubermaticv1.MasterUpdatePhase) (*kubermaticv1.Cluster, bool, error) {
	informerGroup, err := u.clientProvider.GetInformerGroup(c.Spec.SeedDatacenterName)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get informer group for dc %q: %v", c.Spec.SeedDatacenterName, err)
	}

	ns := kubernetesProvider.NamespaceName(c.Name)
	for _, name := range names {
		dep, err := informerGroup.DeploymentInformer.Lister().Deployments(ns).Get(name)
		if err != nil {
			return nil, false, err
		}
		if !healthyDep(dep) {
			return nil, false, nil
		}
	}
	return c, true, nil
}

// healthyDep is true if >= 90% of the expected pods are ready
func healthyDep(dep *v1beta1.Deployment) bool {
	return float64(dep.Status.UpdatedReplicas) >= 0.9*float64(*dep.Spec.Replicas)
}
