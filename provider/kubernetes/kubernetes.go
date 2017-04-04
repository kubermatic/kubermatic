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
	"k8s.io/client-go/kubernetes"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/rbac"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/pkg/util/rand"
	"k8s.io/client-go/rest"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

const (
	updateRetries = 5
)

type kubernetesProvider struct {
	tprClient extensions.Clientset
	client    *kubernetes.Clientset

	mu     sync.Mutex
	cps    map[string]provider.CloudProvider
	dev    bool
	config *rest.Config
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *rest.Config,
	cps map[string]provider.CloudProvider,
	dev bool,
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
		cps:       cps,
		client:    client,
		tprClient: trpClient,
		dev:       dev,
		config:    clientConfig,
	}
}

func (p *kubernetesProvider) GetFreeNodePort() (int, error) {
	for {
		port := rand.IntnRange(12000, 14767)
		sel := labels.NewSelector()
		portString := strconv.Itoa(port)
		req, err := labels.NewRequirement("node-port", selection.Equals, []string{portString})
		if err != nil {
			return 0, err
		}
		sel = sel.Add(*req)
		nsList, err := p.client.Namespaces().List(v1.ListOptions{LabelSelector: sel.String()})
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
		return nil, kerrors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, kerrors.NewBadRequest("cluster humanReadableName is required")
	}

	clusterName := rand.String(9)

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, kerrors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name:        NamespaceName(user.Name, clusterName),
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
			Dev:               spec.Dev,
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
	if p.dev {
		c.Spec.Dev = true
	}

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
	ns, err = p.client.Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	// ensure the NS is deleted when any error occurs
	defer func(ns *v1.Namespace, prov provider.CloudProvider, c *api.Cluster, err error) {
		if err != nil {
			_ = prov.CleanUp(c)
			_ = p.client.Namespaces().Delete(NamespaceName(user.Name, c.Metadata.Name), &v1.DeleteOptions{})
		}
	}(ns, prov, c, err)

	return UnmarshalCluster(p.cps, ns)
	if err != nil {
		return nil, err
	}

	return c, nil
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
		return nil, kerrors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, kerrors.NewBadRequest("cluster humanReadableName is required")
	}

	clusterName := rand.String(9)

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, kerrors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name:        NamespaceName(user.Name, clusterName),
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
	if p.dev {
		c.Spec.Dev = true
	}

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return nil, err
	}

	ns, err = p.client.Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	c, err = UnmarshalCluster(p.cps, ns)

	if err != nil {
		_ = p.client.Namespaces().Delete(NamespaceName(user.Name, clusterName), &v1.DeleteOptions{})
		return nil, err
	}

	return c, nil
}

func (p *kubernetesProvider) clusterAndNS(user provider.User, cluster string) (*api.Cluster, *v1.Namespace, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ns, err := p.client.Namespaces().Get(NamespaceName(user.Name, cluster), metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
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
		return nil, nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
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
		var ns *v1.Namespace
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
			_ = prov.CleanUp(c)
			return nil, fmt.Errorf(
				"cannot set %s cloud config for cluster %q: %v",
				provName,
				c.Metadata.Name,
				err,
			)
		}

		err = p.ApplyCloudProvider(c, ns)
		if err != nil {
			_ = prov.CleanUp(c)
			return nil, err
		}

		ns, err = MarshalCluster(p.cps, c, ns)
		if err != nil {
			return nil, err
		}

		ns, err = p.client.Namespaces().Update(ns)
		if err == nil {
			c, err = UnmarshalCluster(p.cps, ns)
			if err != nil {
				return nil, err
			}

			return c, nil
		}
		if !kerrors.IsConflict(err) {
			return nil, err
		}
	}
	return nil, err
}

// Deprecated at V2 of create cluster endpoint
// this is a super hack and dirty hack to load the AWS cloud config from the cluster controller's templates
// to create the config map by hand for now.
// @TODO Remove with https://github.com/kubermatic/api/issues/220
func (p *kubernetesProvider) ApplyCloudProvider(c *api.Cluster, ns *v1.Namespace) error {
	if c.Spec.Cloud.GetAWS() == nil {
		return nil
	}

	err := p.client.Deployments(ns.Name).Delete("controller-manager", &v1.DeleteOptions{})
	if err != nil {
		glog.Errorf("could not delete controller manager deployment for new aws deployment: %v", err)
	}
	err = p.client.Deployments(ns.Name).Delete("apiserver-v5", &v1.DeleteOptions{})
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

	nsList, err := p.client.Namespaces().List(v1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(l)).String(), FieldSelector: fields.Everything().String()})
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

	list, err := p.tprClient.ClusterAddons(NamespaceName(user.Name, cluster)).List(v1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		err = p.tprClient.ClusterAddons(NamespaceName(user.Name, cluster)).Delete(item.Metadata.Name, nil)
		if err != nil {
			return err
		}
	}

	return p.client.Namespaces().Delete(NamespaceName(user.Name, cluster), &v1.DeleteOptions{})
}

func (p *kubernetesProvider) CreateAddon(user provider.User, cluster string, addonName string) (*extensions.ClusterAddon, error) {
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}

	// Create an instance of our TPR
	addon := &extensions.ClusterAddon{
		Metadata: v1.ObjectMeta{
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

	meta := v1.ObjectMeta{
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

	// TODO: Use propper cluster generator
	return p.tprClient.Nodes(fmt.Sprintf("cluster-%s", cluster)).Create(n)
}
