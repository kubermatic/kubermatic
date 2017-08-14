package kubernetes

import (
	"fmt"
	"log"
	"strings"
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

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

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

func (p *kubernetesProvider) NewClusterWithCloud(user provider.User, spec *api.ClusterSpec, cloud *api.CloudSpec) (*api.Cluster, error) {
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

	dc, found := p.dcs[cloud.Region]
	if !found {
		return nil, errors.NewBadRequest("Unregistered datacenter")
	}

	c := &api.Cluster{
		Metadata: api.Metadata{
			User: user.Name,
			Name: clusterName,
		},
		Spec: api.ClusterSpec{
			HumanReadableName: spec.HumanReadableName,
			WorkerName:        spec.WorkerName,
			Cloud:             cloud,
		},
		Status: api.ClusterStatus{
			LastTransitionTime: time.Now(),
			Phase:              api.PendingClusterStatusPhase,
		},
		Address: &api.ClusterAddress{},
		Seed:    dc.Seed,
	}

	c.Spec.WorkerName = p.workerName

	prov, found := p.cps[cloud.Name]
	if !found {
		return nil, fmt.Errorf("Unable to find provider %s for cluster %s", c.Spec.Cloud.Name, c.Metadata.Name)
	}

	err = prov.InitializeCloudSpec(c)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot set %s cloud config for cluster %q: %v",
			cloud.Name,
			c.Metadata.Name,
			err,
		)
	}

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return nil, err
	}
	ns, err = p.kuberntesClient.Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	// ensure the NS is deleted when any error occurs
	defer func(prov provider.CloudProvider, c *api.Cluster, err error) {
		if err != nil {
			err = prov.CleanUp(c)
			if err != nil {
				glog.Errorf("failed to do cloud provider cleanup after failed cloud provider initialization for cluster %s: %v", c.Metadata.Name, err)
			}
			err = p.kuberntesClient.Namespaces().Delete(NamespaceName(c.Metadata.Name), &metav1.DeleteOptions{})
			if err != nil {
				glog.Errorf("failed to delete cluster after failed creation for cluster %s: %v", c.Metadata.Name, err)
			}
		}
	}(prov, c, err)

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
		provName, prov, err := provider.ClusterCloudProvider(p.cps, c)
		if err != nil {
			return nil, err
		}

		err = prov.InitializeCloudSpec(c)
		if err != nil {
			cleanupErr := prov.CleanUp(c)
			if cleanupErr != nil {
				glog.Errorf("failed to do cloud provider cleanup after failed cloud provider initialization for cluster %s: %v", c.Metadata.Name, err)
			}
			return nil, fmt.Errorf(
				"cannot set %s cloud config for cluster %q: %v",
				provName,
				c.Metadata.Name,
				err,
			)
		}

		err = p.ApplyCloudProvider(c, ns)
		if err != nil {
			cleanupErr := prov.CleanUp(c)
			if cleanupErr != nil {
				glog.Errorf("failed to do cloud provider cleanup after failed cloud provider initialization for cluster %s: %v", c.Metadata.Name, err)
			}

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
	if c.Spec.Cloud.GetAWS() == nil {
		return nil
	}

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
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return err
	}

	list, err := p.tprClient.ClusterAddons(NamespaceName(cluster)).List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		err = p.tprClient.ClusterAddons(NamespaceName(cluster)).Delete(item.Metadata.Name, nil)
		if err != nil {
			return err
		}
	}

	return p.kuberntesClient.Namespaces().Delete(NamespaceName(cluster), &metav1.DeleteOptions{})
}

func (p *kubernetesProvider) CreateAddon(user provider.User, cluster string, addonName string) (*extensions.ClusterAddon, error) {
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}

	// Create an instance of our TPR
	addon := &extensions.ClusterAddon{
		Metadata: metav1.ObjectMeta{
			Name: fmt.Sprintf("addon-%s-%s", strings.Replace(addonName, "/", "", -1), rand.String(4)),
		},
		Name:  addonName,
		Phase: extensions.PendingAddonStatusPhase,
	}

	return p.tprClient.ClusterAddons(fmt.Sprintf("cluster-%s", cluster)).Create(addon)
}

func (p *kubernetesProvider) CreateNode(user provider.User, cluster string, node *api.Node) (*extensions.ClNode, error) {
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}

	meta := metav1.ObjectMeta{
		Name:        node.Metadata.UID,
		Annotations: node.Metadata.Annotations,
	}

	if meta.Annotations == nil {
		meta.Annotations = map[string]string{
			"user": node.Metadata.User,
		}
	} else {
		meta.Annotations["user"] = node.Metadata.User
	}

	n := &extensions.ClNode{
		Metadata: meta,
		Status:   node.Status,
		Spec:     node.Spec,
	}

	// TODO: Use proper cluster generator
	return p.tprClient.Nodes(fmt.Sprintf("cluster-%s", cluster)).Create(n)
}

func (p *kubernetesProvider) DeleteNode(user provider.User, cluster string, node *api.Node) error {
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return err
	}

	return p.tprClient.Nodes(fmt.Sprintf("cluster-%s", cluster)).Delete(node.Metadata.Name, &metav1.DeleteOptions{})
}
