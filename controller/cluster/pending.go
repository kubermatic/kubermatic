package cluster

import (
	"fmt"
	"path"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/resources"
	"github.com/kubermatic/api/controller/template"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (cc *clusterController) syncPendingCluster(c *api.Cluster) (changedC *api.Cluster, err error) {
	_, err = cc.checkTimeout(c)
	if err != nil {
		return nil, err
	}

	changedC, err = cc.pendingCreateRootCA(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// create token-users first and also persist immediately because this
	// changes the cluster. The later secrets and other resources don't.
	changedC, err = cc.launchingCheckTokenUsers(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// check that all services are available
	changedC, err = cc.launchingCheckServices(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	changedC, err = cc.pendingCheckSecrets(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// check that the ingress is available
	err = cc.launchingCheckIngress(c)
	if err != nil {
		return nil, err
	}

	// check that all pcv's are available
	err = cc.launchingCheckPvcs(c)
	if err != nil {
		return nil, err
	}

	// check that all deployments are available
	err = cc.launchingCheckDeployments(c)
	if err != nil {
		return nil, err
	}

	err = cc.launchingCheckDefaultPlugins(c)
	if err != nil {
		return nil, err
	}

	c.Status.LastTransitionTime = time.Now()
	c.Status.Phase = api.LaunchingClusterStatusPhase
	return c, nil
}

func (cc *clusterController) pendingCreateRootCA(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.RootCA.Key != nil {
		return nil, nil
	}

	rootCAReq := csr.CertificateRequest{
		CN: fmt.Sprintf("root-ca.%s.%s.%s", c.Metadata.Name, cc.dc, cc.externalURL),
		KeyRequest: &csr.BasicKeyRequest{
			A: "rsa",
			S: 2048,
		},
		CA: &csr.CAConfig{
			Expiry: fmt.Sprintf("%dh", 24*365*10),
		},
	}
	var err error
	c.Status.RootCA.Cert, _, c.Status.RootCA.Key, err = initca.New(&rootCAReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	return c, nil
}

func (cc *clusterController) pendingCheckSecrets(c *api.Cluster) (*api.Cluster, error) {
	secrets := map[string]func(cc *clusterController, c *api.Cluster, t *template.Template) (*api.Cluster, *v1.Secret, error){
		"apiserver-auth":   createApiserverAuth,
		"apiserver-ssh":    createApiserverSSH,
		"etcd-public-auth": createEtcdAuth,
	}

	recreateSecrets := map[string]struct{}{}
	if len(c.Status.ApiserverSSH) == 0 {
		recreateSecrets["apiserver-ssh"] = struct{}{}
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	for s, gen := range secrets {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.secretStore.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if exists {
			if _, recreate := recreateSecrets[s]; !recreate {
				glog.V(6).Infof("Skipping already existing secret %q", key)
				continue
			}

			err = cc.client.CoreV1().Secrets(ns).Delete(s, &v1.DeleteOptions{})
			if err != nil {
				return nil, err
			}
		}

		t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-secret.yaml"))
		if err != nil {
			return nil, err
		}

		changedC, secret, err := gen(cc, c, t)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().Secrets(ns).Create(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create secret for %s: %v", s, err)
		}

		cc.recordClusterEvent(c, "pending", "Created secret %q", key)

		if changedC != nil {
			return changedC, nil
		}
	}

	return nil, nil
}

func (cc *clusterController) launchingCheckTokenUsers(c *api.Cluster) (*api.Cluster, error) {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	key := fmt.Sprintf("%s/token-users", ns)
	_, exists, err := cc.secretStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if exists {
		glog.V(6).Infof("Skipping already existing secret %q", key)
		return nil, nil
	}

	secret, err := generateTokenUsers(cc, c)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token users: %v", err)
	}
	_, err = cc.client.CoreV1().Secrets(ns).Create(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret for token user: %v", err)
	}
	cc.recordClusterEvent(c, "launching", "Created secret %q", key)
	return c, nil
}

func (cc *clusterController) launchingCheckServices(c *api.Cluster) (*api.Cluster, error) {
	services := map[string]func(cc *clusterController, c *api.Cluster, s string) (*v1.Service, error){
		"etcd":             loadServiceFile,
		"etcd-public":      loadServiceFile,
		"apiserver":        loadServiceFile,
		"apiserver-public": loadServiceFile,
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	for s, gen := range services {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.serviceStore.GetByKey(key)
		if err != nil {
			return nil, err
		}

		if exists {
			glog.V(6).Infof("Skipping already existing service %q", key)
			continue
		}

		services, err := gen(cc, c, s)
		if err != nil {
			return nil, fmt.Errorf("failed to generate service %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().Services(ns).Create(services)
		if err != nil {
			return nil, fmt.Errorf("failed to create service %s: %v", s, err)
		}

		cc.recordClusterEvent(c, "launching", "Created service %q", s)
	}

	if c.Address.EtcdURL != "" {
		return nil, nil
	}

	c.Address.EtcdURL = fmt.Sprintf("https://etcd.%s.%s.%s", c.Metadata.Name, cc.dc, cc.externalURL)

	return c, nil
}

func (cc *clusterController) launchingCheckIngress(c *api.Cluster) error {
	ingress := map[string]func(cc *clusterController, c *api.Cluster, s string) (*extensionsv1beta1.Ingress, error){
		"k8sniff": loadIngressFile,
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	for s, gen := range ingress {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.ingressStore.GetByKey(key)
		if err != nil {
			return err
		}
		if exists {
			glog.V(6).Infof("Skipping already existing ingress %q", key)
			return nil
		}
		ingress, err := gen(cc, c, s)
		if err != nil {
			return fmt.Errorf("failed to generate %s: %v", s, err)
		}

		_, err = cc.client.ExtensionsV1beta1().Ingresses(ns).Create(ingress)
		if err != nil {
			return fmt.Errorf("failed to create ingress %s: %v", s, err)
		}

		cc.recordClusterEvent(c, "launching", "Created ingress")
	}
	return nil
}

func (cc *clusterController) launchingCheckDeployments(c *api.Cluster) error {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	deps := map[string]func(c *api.Cluster, v *api.MasterVersion, masterResourcesPath, overwriteHost, dc string) (*extensionsv1beta1.Deployment, error){
		"etcd":               resources.LoadDeploymentFile,
		"etcd-public":        resources.LoadDeploymentFile,
		"apiserver":          resources.LoadApiserver,
		"controller-manager": resources.LoadDeploymentFile,
		"scheduler":          resources.LoadDeploymentFile,
	}

	existingDeps, err := cc.depStore.ByIndex("namespace", ns)
	if err != nil {
		return err
	}

	for s, gen := range deps {
		exists := false
		for _, obj := range existingDeps {
			dep := obj.(*extensionsv1beta1.Deployment)
			if role, found := dep.Spec.Selector.MatchLabels["role"]; found && role == s {
				exists = true
				break
			}
		}
		if exists {
			glog.V(7).Infof("Skipping already existing dep %q for cluster %q", s, c.Metadata.Name)
			continue
		}

		dep, err := gen(c, cc.latestVersion, cc.masterResourcesPath, cc.overwriteHost, cc.dc)
		if err != nil {
			return fmt.Errorf("failed to generate deployment %s: %v", s, err)
		}

		_, err = cc.client.ExtensionsV1beta1().Deployments(ns).Create(dep)
		if err != nil {
			return fmt.Errorf("failed to create deployment %s: %v", s, err)
		}

		cc.recordClusterEvent(c, "launching", "Created dep %q", s)
	}

	return nil
}

func (cc *clusterController) launchingCheckPvcs(c *api.Cluster) error {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	pvcs := map[string]func(cc *clusterController, c *api.Cluster, s string) (*v1.PersistentVolumeClaim, error){
		"etcd":        loadPVCFile,
		"etcd-public": loadPVCFile,
	}

	for s, gen := range pvcs {
		key := fmt.Sprintf("%s/%s-pvc", ns, s)
		_, exists, err := cc.pvcStore.GetByKey(key)
		if err != nil {
			return err
		}

		if exists {
			glog.V(6).Infof("Skipping already existing pvc %q", key)
			continue
		}

		pvc, err := gen(cc, c, s)
		if err != nil {
			return fmt.Errorf("failed to generate pvc %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().PersistentVolumeClaims(ns).Create(pvc)
		if err != nil {
			return fmt.Errorf("failed to create pvc %s; %v", s, err)
		}

		cc.recordClusterEvent(c, "launching", "Created pvc %q", s)
	}

	return nil
}

func (cc *clusterController) launchingCheckDefaultPlugins(c *api.Cluster) error {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	defaultPlugins := map[string]string{
		"heapster":            "heapster",
		"kubedns":             "kubedns",
		"kubeproxy":           "kube-proxy",
		"kubernetesdashboard": "kubernetes-dashboard",
	}

	for safeName, name := range defaultPlugins {
		metaName := fmt.Sprintf("addon-default-%s", safeName)
		_, exists, err := cc.addonStore.GetByKey(fmt.Sprintf("%s/%s", ns, metaName))
		if err != nil {
			return err
		}

		if exists {
			glog.V(6).Infof("Skipping already existing default addon %q", metaName)
			continue
		}

		addon := &extensions.ClusterAddon{
			Metadata: v1.ObjectMeta{
				Name: metaName,
			},
			Name:  name,
			Phase: extensions.PendingAddonStatusPhase,
		}

		_, err = cc.tprClient.ClusterAddons(fmt.Sprintf("cluster-%s", c.Metadata.Name)).Create(addon)
		if err != nil {
			return fmt.Errorf("failed to create default addon third-party-resource %s; %v", name, err)
		}
	}

	return nil
}
