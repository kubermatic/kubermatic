package seed

import (
	"time"

	coreinformers "k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
	extv1beta1 "k8s.io/client-go/informers/extensions/v1beta1"
	rbacv1beta1 "k8s.io/client-go/informers/rbac/v1beta1"
	"k8s.io/client-go/kubernetes"

	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	crdinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/informers/externalversions"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/informers/externalversions/etcdoperator/v1beta2"
	appsv1beta1 "k8s.io/client-go/informers/apps/v1beta1"
)

// Group is a group of informers needed for kubermatic
type Group struct {
	NamespaceInformer          corev1.NamespaceInformer
	DeploymentInformer         appsv1beta1.DeploymentInformer
	SecretInformer             corev1.SecretInformer
	ServiceInformer            corev1.ServiceInformer
	IngressInformer            extv1beta1.IngressInformer
	PvcInformer                corev1.PersistentVolumeClaimInformer
	ConfigMapInformer          corev1.ConfigMapInformer
	ServiceAccountInformer     corev1.ServiceAccountInformer
	ClusterRoleBindingInformer rbacv1beta1.ClusterRoleBindingInformer

	EtcdClusterInformer etcdoperatorv1beta2.EtcdClusterInformer
}

// New returns a instance of Group
func New(kubeClient kubernetes.Interface, crdClient crdclient.Interface) *Group {
	coreInformers := coreinformers.NewSharedInformerFactory(kubeClient, 5*time.Minute)
	g := Group{}
	g.NamespaceInformer = coreInformers.Core().V1().Namespaces()
	g.DeploymentInformer = coreInformers.Apps().V1beta1().Deployments()
	g.SecretInformer = coreInformers.Core().V1().Secrets()
	g.ServiceInformer = coreInformers.Core().V1().Services()
	g.IngressInformer = coreInformers.Extensions().V1beta1().Ingresses()
	g.PvcInformer = coreInformers.Core().V1().PersistentVolumeClaims()
	g.ConfigMapInformer = coreInformers.Core().V1().ConfigMaps()
	g.ServiceAccountInformer = coreInformers.Core().V1().ServiceAccounts()
	g.ClusterRoleBindingInformer = coreInformers.Rbac().V1beta1().ClusterRoleBindings()

	crdInformers := crdinformers.NewSharedInformerFactory(crdClient, 5*time.Minute)
	g.EtcdClusterInformer = crdInformers.Etcd().V1beta2().EtcdClusters()

	return &g
}

// HasSynced tells if the all informers of the group have synced
func (g *Group) HasSynced() bool {
	return g.NamespaceInformer.Informer().HasSynced() &&
		g.DeploymentInformer.Informer().HasSynced() &&
		g.SecretInformer.Informer().HasSynced() &&
		g.ServiceInformer.Informer().HasSynced() &&
		g.IngressInformer.Informer().HasSynced() &&
		g.PvcInformer.Informer().HasSynced() &&
		g.ConfigMapInformer.Informer().HasSynced() &&
		g.ServiceAccountInformer.Informer().HasSynced() &&
		g.ClusterRoleBindingInformer.Informer().HasSynced() &&
		g.EtcdClusterInformer.Informer().HasSynced()

}

// Run starts all informers of the group
func (g *Group) Run(stopCh <-chan struct{}) {
	//k8s
	go g.NamespaceInformer.Informer().Run(stopCh)
	go g.DeploymentInformer.Informer().Run(stopCh)
	go g.SecretInformer.Informer().Run(stopCh)
	go g.ServiceInformer.Informer().Run(stopCh)
	go g.IngressInformer.Informer().Run(stopCh)
	go g.PvcInformer.Informer().Run(stopCh)
	go g.ConfigMapInformer.Informer().Run(stopCh)
	go g.ServiceAccountInformer.Informer().Run(stopCh)
	go g.ClusterRoleBindingInformer.Informer().Run(stopCh)

	//crd
	go g.EtcdClusterInformer.Informer().Run(stopCh)
}
