package kubernetes

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/rbac"
	"k8s.io/client-go/rest"
)

const (
	updateRetries = 5
)

type kubernetesProvider struct {
	tprClient       extensions.Clientset
	kuberntesClient *kubernetes.Clientset

	mu         sync.Mutex
	cps        map[string]provider.CloudProvider
	workerName string
	config     *rest.Config
	dcs        map[string]provider.DatacenterMeta
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *rest.Config,
	cps map[string]provider.CloudProvider,
	workerName string,
	dcs map[string]provider.DatacenterMeta,
) provider.KubernetesProvider {
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	trpClient, err := extensions.WrapClientsetWithExtensions(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &kubernetesProvider{
		cps:             cps,
		kuberntesClient: client,
		tprClient:       trpClient,
		workerName:      workerName,
		config:          clientConfig,
		dcs:             dcs,
	}
}

func (p *kubernetesProvider) NewClusterWithCloud(user provider.User, spec *api.ClusterSpec) (*api.Cluster, error) {
	var err error

	// call cluster before lock is taken
	cs, err := p.Clusters(user)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// sanity checks for a fresh cluster
	switch {
	case user.Name == "":
		return nil, errors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, errors.NewBadRequest("cluster humanReadableName is required")
	}

	clusterName := rand.String(9)

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, errors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        NamespaceName(clusterName),
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
	}

	dc, found := p.dcs[spec.Cloud.DatacenterName]
	if !found {
		return nil, errors.NewBadRequest("Unknown datacenter")
	}

	c := &api.Cluster{
		Metadata: api.Metadata{
			User: user.Name,
			Name: clusterName,
		},
		Spec: *spec,
		Status: api.ClusterStatus{
			LastTransitionTime: time.Now(),
			Phase:              api.PendingClusterStatusPhase,
		},
		Address: &api.ClusterAddress{},
		Seed:    dc.Seed,
	}

	c.Spec.WorkerName = p.workerName
	_, prov, err := provider.ClusterCloudProvider(p.cps, c)
	if err != nil {
		return nil, err
	}

	cloud, err := prov.Initialize(c.Spec.Cloud, clusterName)
	if err != nil {
		return nil, fmt.Errorf("could not initialize cloud provider: %v", err)
	}

	c.Spec.Cloud = cloud

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return nil, err
	}
	ns, err = p.kuberntesClient.Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	return UnmarshalCluster(p.cps, ns)
}

// Deprecated in favor of NewClusterWithCloud
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func (p *kubernetesProvider) NewCluster(user provider.User, spec *api.ClusterSpec) (*api.Cluster, error) {
	// call cluster before lock is taken
	cs, err := p.Clusters(user)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// sanity checks for a fresh cluster
	switch {
	case user.Name == "":
		return nil, errors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, errors.NewBadRequest("cluster humanReadableName is required")
	}

	clusterName := rand.String(9)

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, errors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        NamespaceName(clusterName),
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
	}

	c := &api.Cluster{
		Metadata: api.Metadata{
			User: user.Name,
			Name: clusterName,
		},
		Spec: *spec,
		Status: api.ClusterStatus{
			LastTransitionTime: time.Now(),
			Phase:              api.PendingClusterStatusPhase,
		},
		Address: &api.ClusterAddress{},
	}
	c.Spec.WorkerName = p.workerName

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return nil, err
	}

	ns, err = p.kuberntesClient.Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	c, err = UnmarshalCluster(p.cps, ns)

	if err != nil {
		delErr := p.kuberntesClient.Namespaces().Delete(NamespaceName(clusterName), &metav1.DeleteOptions{})
		if delErr != nil {
			glog.Errorf("failed to delete cluster after failed creation for cluster %s: %v", c.Metadata.Name, err)
		}
		return nil, err
	}

	return c, nil
}

func (p *kubernetesProvider) clusterAndNS(user provider.User, cluster string) (*api.Cluster, *apiv1.Namespace, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ns, err := p.kuberntesClient.Namespaces().Get(NamespaceName(cluster), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, errors.NewNotFound(rbac.Resource("cluster"), cluster)
		}
		return nil, nil, err
	}

	c, err := UnmarshalCluster(p.cps, ns)
	if err != nil {
		return nil, nil, err
	}

	_, isAdmin := user.Roles["admin"]
	if c.Metadata.User != user.Name && !isAdmin {
		// don't return Forbidden, not NotFound to obfuscate the existence
		return nil, nil, errors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	return c, ns, nil
}

func (p *kubernetesProvider) Cluster(user provider.User, cluster string) (*api.Cluster, error) {
	c, _, err := p.clusterAndNS(user, cluster)
	return c, err
}

// Deprecated in favor of NewClusterWithCloud
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func (p *kubernetesProvider) SetCloud(user provider.User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error) {
	var err error
	for r := updateRetries; r >= 0; r-- {
		if r != updateRetries {
			time.Sleep(500 * time.Millisecond)
		}

		var c *api.Cluster
		var ns *apiv1.Namespace
		c, ns, err = p.clusterAndNS(user, cluster)
		if err != nil {
			return nil, err
		}
		c.Spec.Cloud = cloud
		_, cp, err := provider.ClusterCloudProvider(p.cps, c)
		if err != nil {
			return nil, err
		}
		cloud, err := cp.Initialize(cloud, c.Metadata.Name)
		if err != nil {
			cleanupErr := cp.CleanUp(cloud)
			if cleanupErr != nil {
				glog.Errorf("failed to cleanup after initialize: %v", cleanupErr)
			}
			return nil, err
		}
		c.Spec.Cloud = cloud

		err = p.ApplyCloudProvider(c, ns)
		if err != nil {
			return nil, err
		}

		ns, err = MarshalCluster(p.cps, c, ns)
		if err != nil {
			return nil, err
		}

		ns, err = p.kuberntesClient.Namespaces().Update(ns)
		if err == nil {
			c, err = UnmarshalCluster(p.cps, ns)
			if err != nil {
				return nil, err
			}

			return c, nil
		}
		if !errors.IsConflict(err) {
			return nil, err
		}
	}
	return nil, err
}

// Deprecated at V2 of create cluster endpoint
// this is a super hack and dirty hack to load the AWS cloud config from the cluster controller's templates
// to create the config map by hand for now.
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func (p *kubernetesProvider) ApplyCloudProvider(c *api.Cluster, ns *apiv1.Namespace) error {
	deps := []string{
		"controller-manager",
		"apiserver",
	}

	//We need to set the replica count to 0 before deleting the deployment.
	//This is a known issue in the apiserver, kubectl does the same...
	for _, name := range deps {
		dep, err := p.kuberntesClient.ExtensionsV1beta1Client.Deployments(ns.Name).Get(name, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get deployment %s/%s: %v", ns.Name, name, err)
		}
		replicas := int32(0)
		dep.Spec.Replicas = &replicas
		dep, err = p.kuberntesClient.ExtensionsV1beta1Client.Deployments(ns.Name).Update(dep)
		if err != nil {
			glog.Errorf("failed to update deployment %s/%s: %v", ns.Name, name, err)
		}
		err = p.kuberntesClient.ExtensionsV1beta1Client.Deployments(ns.Name).Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("failed to delete deployment %s/%s: %v", ns.Name, name, err)
		}
	}
	//Dirty hack to give the controller indexers time to sync
	time.Sleep(5 * time.Second)

	c.Status.Phase = api.PendingClusterStatusPhase
	c.Status.LastTransitionTime = time.Now()

	return nil
}

func (p *kubernetesProvider) Clusters(user provider.User) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	l := map[string]string{
		RoleLabelKey: ClusterRoleLabel,
	}

	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		l[userLabelKey] = LabelUser(user.Name)
	}

	nsList, err := p.kuberntesClient.Namespaces().List(metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(l)).String(), FieldSelector: fields.Everything().String()})
	if err != nil {
		return nil, err
	}

	cs := make([]*api.Cluster, 0, len(nsList.Items))
	for i := range nsList.Items {
		ns := nsList.Items[i]
		c, err := UnmarshalCluster(p.cps, &ns)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshaling namespace %s: %v", ns.Name, err))
			continue
		}

		cs = append(cs, c)
	}

	return cs, nil
}

func (p *kubernetesProvider) DeleteCluster(user provider.User, cluster string) error {
	// check permission by getting the cluster first
	c, err := p.Cluster(user, cluster)
	if err != nil {
		return err
	}

	_, cp, err := provider.ClusterCloudProvider(p.cps, c)
	if err != nil {
		return err
	}

	if c.Spec.Cloud != nil {
		err = cp.CleanUp(c.Spec.Cloud)
		if err != nil {
			return err
		}
	}

	return p.kuberntesClient.Namespaces().Delete(NamespaceName(cluster), &metav1.DeleteOptions{})
}

func (p *kubernetesProvider) UpgradeCluster(user provider.User, cluster, version string) error {
	c, ns, err := p.clusterAndNS(user, cluster)
	if err != nil {
		return err
	}

	c.Spec.MasterVersion = version
	c.Status.Phase = api.UpdatingMasterClusterStatusPhase

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return err
	}
	ns, err = p.kuberntesClient.Namespaces().Update(ns)

	return err
}
