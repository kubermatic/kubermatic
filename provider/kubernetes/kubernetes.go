package kubernetes

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/runtime/serializer"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

const (
	updateRetries = 5
)

type kubernetesProvider struct {
	client *client.Client

	mu     sync.Mutex
	cps    map[string]provider.CloudProvider
	dev    bool
	config *restclient.Config
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *restclient.Config,
	cps map[string]provider.CloudProvider,
	dev bool,
) provider.KubernetesProvider {

	client, err := client.New(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &kubernetesProvider{
		cps:    cps,
		client: client,
		dev:    dev,
		config: clientConfig,
	}
}

func (p *kubernetesProvider) NewCluster(user provider.User, cluster string, spec *api.ClusterSpec) (*api.Cluster, error) {
	// call cluster before lock is taken
	cs, err := p.Clusters(user)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// sanity checks for a fresh cluster
	switch {
	case cluster == "":
		return nil, kerrors.NewBadRequest("cluster name is required")
	case user.Name == "":
		return nil, kerrors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, kerrors.NewBadRequest("cluster humanReadableName is required")
	}

	for _, c := range cs {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, kerrors.NewAlreadyExists(rbac.Resource("cluster"), spec.HumanReadableName)
		}
	}

	ns := &kapi.Namespace{
		ObjectMeta: kapi.ObjectMeta{
			Name:        NamespaceName(user.Name, cluster),
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
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
		_ = p.client.Namespaces().Delete(NamespaceName(user.Name, cluster))
		return nil, err
	}

	return c, nil
}

func (p *kubernetesProvider) clusterAndNS(user provider.User, cluster string) (*api.Cluster, *kapi.Namespace, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ns, err := p.client.Namespaces().Get(NamespaceName(user.Name, cluster))
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
		var ns *kapi.Namespace
		c, ns, err = p.clusterAndNS(user, cluster)
		if err != nil {
			return nil, err
		}

		c.Spec.Cloud = cloud
		provName, prov, err := provider.ClusterCloudProvider(p.cps, c)
		if err != nil {
			return nil, err
		}

		err = prov.PrepareCloudSpec(c)
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

	nsList, err := p.client.Namespaces().List(kapi.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
		RoleLabelKey: ClusterRoleLabel,
		userLabelKey: LabelUser(user.Name),
	})), FieldSelector: fields.Everything()})
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

	return p.client.Namespaces().Delete(NamespaceName(user.Name, cluster))
}

func (p *kubernetesProvider) CreateAddon(user provider.User, cluster string, addonSpec api.ClusterAddonSpec) error {
	groupversion := unversioned.GroupVersion{
		Group:   "k8s.io",
		Version: "v1",
	}
	p.config.GroupVersion = &groupversion

	config := *p.config
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: kapi.Codecs}

	client, err := client.New(&config)
	if err != nil {
		log.Fatal(err)
	}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&api.ClusterAddon{},
				&api.ClusterAddonList{},
				&kapi.ListOptions{},
				&kapi.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(kapi.Scheme)

	// initialize third party resource if it does not exist
	tpr, err := client.Extensions().ThirdPartyResources().Get("addons.kubermatic.io")
	if err != nil {
		if kerrors.IsNotFound(err) {
			tpr := &extensions.ThirdPartyResource{
				ObjectMeta: kapi.ObjectMeta{
					Name:            "addons.kubermatic.io",
					ResourceVersion: "v1",
				},
				Versions: []extensions.APIVersion{
					{Name: "v1"},
				},
				Description: "Addon resource",
			}

			result, err := client.Extensions().ThirdPartyResources().Create(tpr)
			if err != nil {
				return err
			}
			glog.Infof("CREATED: %#v\nFROM: %#v\n", result, tpr)
		} else {
			return err
		}
	} else {
		glog.Infof("SKIPPING: already exists %#v\n", tpr)
	}

	var resultAddon api.ClusterAddon

	err = client.Get().
		Resource("addons").
		Namespace(kapi.NamespaceDefault).
		Name(addonSpec.Name).
		Do().Into(&resultAddon)

	if err != nil {
		if kerrors.IsNotFound(err) {
			// Create an instance of our TPR
			tprAddon := &api.ClusterAddon{
				Metadata: kapi.ObjectMeta{
					Name: "addon1",
				},
				Spec: addonSpec,
			}

			var result api.ClusterAddon
			request := client.Post()
			request.Resource("addons")
			request.Namespace(kapi.NamespaceDefault)
			request.Body(tprAddon)

			requestResult := request.Do()
			requestResult.Into(&result)

			if err != nil {
				return err
			}
			glog.Infof("CREATED: %#v\n", result)
		} else {
			return err
		}
	} else {
		glog.Infof("GET: %#v\n", resultAddon)
	}

	// Fetch a list of our addons
	addonList := api.ClusterAddonList{}
	err = client.Get().Resource("addons").Do().Into(&addonList)
	if err != nil {
		return err
	}
	glog.Infof("LIST: %#v\n", addonList)

	return nil
}
