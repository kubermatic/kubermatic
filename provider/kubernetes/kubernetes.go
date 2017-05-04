package kubernetes

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/rbac"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/fields"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

const (
	updateRetries = 5
)

type kubernetesProvider struct {
	tprClient                          extensions.Clientset
	kuberntesClient                    *kubernetes.Clientset
	minAPIServerPort, maxAPIServerPort int

	mu         sync.Mutex
	cps        map[string]provider.CloudProvider
	workerName string
	config     *rest.Config
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *rest.Config,
	cps map[string]provider.CloudProvider,
	workerName string,
	minAPIServerPort, maxAPIServerPort int,
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
		cps:              cps,
		kuberntesClient:  client,
		tprClient:        trpClient,
		workerName:       workerName,
		config:           clientConfig,
		minAPIServerPort: minAPIServerPort,
		maxAPIServerPort: maxAPIServerPort,
	}
}

func (p *kubernetesProvider) GetFreeNodePort() (int, error) {
	for {
		port := rand.IntnRange(p.minAPIServerPort, p.maxAPIServerPort)
		sel := labels.NewSelector()
		portString := strconv.Itoa(port)
		req, err := labels.NewRequirement("node-port", selection.Equals, []string{portString})
		if err != nil {
			return 0, err
		}
		sel = sel.Add(*req)
		nsList, err := p.kuberntesClient.Namespaces().List(metav1.ListOptions{LabelSelector: sel.String()})
		if err != nil {
			return 0, err
		}
		if len(nsList.Items) == 0 {
			return port, nil
		}
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
	nodePort, err := p.GetFreeNodePort()
	if err != nil {
		return nil, err
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
		Address: &api.ClusterAddress{
			NodePort: nodePort,
		},
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
// @TODO Remove with https://github.com/kubermatic/api/issues/220
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
	nodePort, err := p.GetFreeNodePort()
	if err != nil {
		return nil, err
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
		Address: &api.ClusterAddress{
			NodePort: nodePort,
		},
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
// @TODO Remove with https://github.com/kubermatic/api/issues/220
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
// @TODO Remove with https://github.com/kubermatic/api/issues/220
func (p *kubernetesProvider) ApplyCloudProvider(c *api.Cluster, ns *apiv1.Namespace) error {
	if c.Spec.Cloud.GetAWS() == nil {
		return nil
	}

	err := p.kuberntesClient.ExtensionsV1beta1Client.Deployments(ns.Name).Delete("controller-manager", &metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("could not delete controller manager deployment for new aws deployment: %v", err)
	}
	err = p.kuberntesClient.ExtensionsV1beta1Client.Deployments(ns.Name).Delete("apiserver-v5", &metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("could not delete apiserver deployment for new aws deployment: %v", err)
	}

	c.Status.Phase = api.PendingClusterStatusPhase

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
