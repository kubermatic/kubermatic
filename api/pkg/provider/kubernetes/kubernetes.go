package kubernetes

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubernetesProvider struct {
	crdClient       crdclient.Interface
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

	crdClient := crdclient.NewForConfigOrDie(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &kubernetesProvider{
		cps:             cps,
		kuberntesClient: client,
		crdClient:       crdClient,
		workerName:      workerName,
		config:          clientConfig,
		dcs:             dcs,
	}
}

func (p *kubernetesProvider) NewClusterWithCloud(user auth.User, spec *api.ClusterSpec) (*api.Cluster, error) {
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
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}
	}

	ns := &v1.Namespace{
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
	ns, err = p.kuberntesClient.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	return UnmarshalCluster(p.cps, ns)
}

func (p *kubernetesProvider) clusterAndNS(user auth.User, cluster string) (*api.Cluster, *v1.Namespace, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ns, err := p.kuberntesClient.CoreV1().Namespaces().Get(NamespaceName(cluster), metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil, errors.NewNotFound("cluster", cluster)
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
		return nil, nil, errors.NewNotFound("cluster", cluster)
	}

	return c, ns, nil
}

func (p *kubernetesProvider) Cluster(user auth.User, cluster string) (*api.Cluster, error) {
	c, _, err := p.clusterAndNS(user, cluster)
	return c, err
}

func (p *kubernetesProvider) Clusters(user auth.User) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	l := map[string]string{
		RoleLabelKey: ClusterRoleLabel,
	}

	if _, isAdmin := user.Roles["admin"]; !isAdmin {
		l[userLabelKey] = LabelUser(user.Name)
	}

	nsList, err := p.kuberntesClient.CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(l)).String(), FieldSelector: labels.Everything().String()})
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

func (p *kubernetesProvider) DeleteCluster(user auth.User, cluster string) error {
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

	return p.kuberntesClient.CoreV1().Namespaces().Delete(NamespaceName(cluster), &metav1.DeleteOptions{})
}

func (p *kubernetesProvider) UpgradeCluster(user auth.User, cluster, version string) error {
	c, ns, err := p.clusterAndNS(user, cluster)
	if err != nil {
		return err
	}

	c.Spec.MasterVersion = version
	c.Status.Phase = api.UpdatingMasterClusterStatusPhase
	c.Status.LastTransitionTime = time.Now()
	c.Status.MasterUpdatePhase = api.StartMasterUpdatePhase

	ns, err = MarshalCluster(p.cps, c, ns)
	if err != nil {
		return err
	}
	ns, err = p.kuberntesClient.CoreV1().Namespaces().Update(ns)

	return err
}
