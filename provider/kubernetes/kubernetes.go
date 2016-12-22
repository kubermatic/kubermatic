package kubernetes

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/rbac"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/pkg/util/rand"
	"k8s.io/client-go/rest"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

const (
	updateRetries = 5
)

type kubernetesProvider struct {
	tprClient rest.Interface
	client    *kubernetes.Clientset

	mu     sync.Mutex
	cps    map[string]provider.CloudProvider
	dev    bool
	config *rest.Config
}

//RegisterTprs register's our own ThirdPartyResources
func RegisterTprs(gv schema.GroupVersion) {
	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				gv,
				&api.ClusterAddon{},
				&api.ClusterAddonList{},
				&v1.ListOptions{},
				&v1.DeleteOptions{},
			)
			return nil
		})
	err := schemeBuilder.AddToScheme(kapi.Scheme)

	if err != nil {
		log.Fatal(err)
	}
}

//NewTprClient returns a client which is meant for our own ThirdPartyResources
func NewTprClient(clientConfig *rest.Config) rest.Interface {
	config := *clientConfig

	groupversion := schema.GroupVersion{
		Group:   "kubermatic.io",
		Version: "v1",
	}
	RegisterTprs(groupversion)

	config.GroupVersion = &groupversion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: kapi.Codecs}

	tprclient, err := rest.RESTClientFor(&config)
	if err != nil {
		log.Fatal(err)
	}

	return tprclient
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

	return &kubernetesProvider{
		cps:       cps,
		client:    client,
		tprClient: NewTprClient(clientConfig),
		dev:       dev,
		config:    clientConfig,
	}
}

func (p *kubernetesProvider) GetFreeNodePort() (int, error) {
	for {
		port := rand.IntnRange(30000, 32767)
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

	cluster := rand.String(9)

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, kerrors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name:        NamespaceName(user.Name, cluster),
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
			Name: cluster,
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
		_ = p.client.Namespaces().Delete(NamespaceName(user.Name, cluster), &v1.DeleteOptions{})
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

	if c.Metadata.User != user.Name {
		// don't return Forbidden, not NotFound to obfuscate the existence
		return nil, nil, kerrors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	return c, ns, nil
}

func (p *kubernetesProvider) Cluster(user provider.User, cluster string) (*api.Cluster, error) {
	c, _, err := p.clusterAndNS(user, cluster)
	return c, err
}

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
			return nil, fmt.Errorf(
				"cannot set %s cloud config for cluster %q: %v",
				provName,
				c.Metadata.Name,
				err,
			)
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

func (p *kubernetesProvider) Clusters(user provider.User) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nsList, err := p.client.Namespaces().List(v1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
		RoleLabelKey: ClusterRoleLabel,
		userLabelKey: LabelUser(user.Name),
	})).String(), FieldSelector: fields.Everything().String()})
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
	return p.client.Namespaces().Delete(NamespaceName(user.Name, cluster), &v1.DeleteOptions{})
}

func (p *kubernetesProvider) CreateAddon(user provider.User, cluster string, addonName string) (*api.ClusterAddon, error) {
	_, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}

	// Create an instance of our TPR
	addon := &api.ClusterAddon{
		Metadata: v1.ObjectMeta{
			Name: fmt.Sprintf("addon-%s-%s", addonName, rand.String(4)),
		},
		Name:  addonName,
		Phase: api.PendingAddonStatusPhase,
	}

	var result api.ClusterAddon
	err = p.tprClient.
		Post().
		Resource("clusteraddons").
		Namespace(fmt.Sprintf("cluster-%s", cluster)).
		Body(addon).
		Do().
		Into(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
