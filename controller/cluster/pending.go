package cluster

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider/kubernetes"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/yaml"
)

func (cc *clusterController) pendingCheckTimeout(c *api.Cluster) (*api.Cluster, error) {
	now := time.Now()
	timeSinceCreation := now.Sub(c.Status.LastTransitionTime)
	if timeSinceCreation > launchTimeout {
		glog.Infof("Launch timeout for cluster %q after %v", c.Metadata.Name, timeSinceCreation)
		c.Status.Phase = api.FailedClusterStatusPhase
		c.Status.LastTransitionTime = now
		return c, nil
	}

	return nil, nil
}

func (cc *clusterController) pendingCheckTokenUsers(c *api.Cluster) (*api.Cluster, error) {
	generateTokenUsers := func() (*kapi.Secret, error) {
		rawToken := make([]byte, 64)
		_, err := crand.Read(rawToken)
		if err != nil {
			return nil, err
		}
		token := sha256.Sum256(rawToken)
		hexToken := hex.EncodeToString(token[:])

		secret := kapi.Secret{
			ObjectMeta: kapi.ObjectMeta{
				Name: "token-users",
			},
			Type: kapi.SecretTypeOpaque,
			Data: map[string][]byte{
				"file": []byte(fmt.Sprintf("%s,admin,admin", hexToken)),
			},
		}
		c.Address = &api.ClusterAddress{
			URL:   fmt.Sprintf(cc.urlPattern, c.Metadata.Name),
			Token: hexToken,
		}
		return &secret, nil
	}
	ns := kubernetes.NamePrefix + c.Metadata.Name

	key := fmt.Sprintf("%s/token-users", ns)
	_, exists, err := cc.secretStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if exists {
		glog.V(6).Infof("Skipping already existing secret %q", key)
		return nil, nil
	}

	secret, err := generateTokenUsers()
	if err != nil {
		return nil, err
	}
	_, err = cc.client.Secrets(ns).Create(secret)
	if err != nil {
		return nil, err
	}
	cc.recordClusterEvent(c, "pending", "Created secret %q", key)
	return c, nil
}

func (cc *clusterController) pendingCheckSecrets(c *api.Cluster) error {
	loadFile := func(s string) (*kapi.Secret, error) {
		file, err := ioutil.ReadFile(path.Join(cc.masterResourcesPath, s+"-secret.yaml"))
		if err != nil {
			return nil, err
		}
		var secret kapi.Secret
		jsonBytes, err := yaml.ToJSON(file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonBytes, &secret)
		if err != nil {
			return nil, err
		}
		return &secret, nil
	}
	secrets := map[string]func(s string) (*kapi.Secret, error){
		"apiserver-auth": loadFile,
		"apiserver-ssh":  loadFile,
	}

	ns := kubernetes.NamePrefix + c.Metadata.Name
	for s, gen := range secrets {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.secretStore.GetByKey(key)
		if err != nil {
			return err
		}
		if exists {
			glog.V(6).Infof("Skipping already existing secret %q", key)
			continue
		}

		secret, err := gen(s)
		if err != nil {
			return err
		}
		_, err = cc.client.Secrets(ns).Create(secret)
		if err != nil {
			return err
		}
		cc.recordClusterEvent(c, "pending", "Created secret %q", key)
	}

	return nil
}

func (cc *clusterController) pendingCheckServices(c *api.Cluster) error {
	loadFile := func(s string) (*kapi.Service, error) {
		file, err := ioutil.ReadFile(path.Join(cc.masterResourcesPath, s+"-service.yaml"))
		if err != nil {
			return nil, err
		}
		var service kapi.Service
		jsonBytes, err := yaml.ToJSON(file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonBytes, &service)
		if err != nil {
			return nil, err
		}
		return &service, nil
	}
	services := map[string]func(s string) (*kapi.Service, error){
		"etcd":             loadFile,
		"etcd-public":      loadFile,
		"apiserver":        loadFile,
		"apiserver-public": loadFile,
	}

	ns := kubernetes.NamePrefix + c.Metadata.Name
	for s, gen := range services {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.serviceStore.GetByKey(key)
		if err != nil {
			return err
		}
		if exists {
			glog.V(6).Infof("Skipping already existing service %q", key)
			continue
		}

		services, err := gen(s)
		if err != nil {
			return err
		}
		_, err = cc.client.Services(ns).Create(services)
		if err != nil {
			return err
		}
		cc.recordClusterEvent(c, "pending", "Created service %q", s)
	}

	return nil
}

func (cc *clusterController) pendingCheckReplicationController(c *api.Cluster) error {
	loadFile := func(s string) (*kapi.ReplicationController, error) {
		file, err := ioutil.ReadFile(path.Join(cc.masterResourcesPath, s+"-rc.yaml"))
		if err != nil {
			return nil, err
		}
		var rc kapi.ReplicationController
		jsonBytes, err := yaml.ToJSON(file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonBytes, &rc)
		if err != nil {
			return nil, err
		}
		return &rc, nil
	}
	rcs := map[string]func(s string) (*kapi.ReplicationController, error){
		"etcd":               loadFile,
		"etcd-public":        loadFile,
		"apiserver":          loadFile,
		"controller-manager": loadFile,
		"scheduler":          loadFile,
	}

	ns := kubernetes.NamePrefix + c.Metadata.Name
	existingRCs, err := cc.rcStore.ByIndex("namespace", ns)
	if err != nil {
		return err
	}
	for s, gen := range rcs {
		exists := false
		for _, obj := range existingRCs {
			rc := obj.(*kapi.ReplicationController)
			if role, found := rc.Spec.Selector["role"]; found && role == s {
				exists = true
				break
			}
		}
		if exists {
			glog.V(7).Infof("Skipping already existing rc %q for cluster %q", s, c.Metadata.Name)
			continue
		}

		rc, err := gen(s)
		if err != nil {
			return err
		}
		_, err = cc.client.ReplicationControllers(ns).Create(rc)
		if err != nil {
			return err
		}
		cc.recordClusterEvent(c, "pending", "Created rc %q", s)
	}

	return nil
}

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamePrefix + c.Metadata.Name
	rcs, err := cc.rcStore.ByIndex("namespace", ns)
	if err != nil {
		return false, nil, err
	}
	health := api.ClusterHealth{
		ClusterHealthStatus: api.ClusterHealthStatus{
			Etcd: []bool{false},
		},
	}
	healthMapping := map[string]*bool{
		"etcd": &health.Etcd[0],
		// "etcd-public" TODO(sttts): add etcd-public?
		"apiserver":          &health.Apiserver,
		"controller-manager": &health.Controller,
		"scheduler":          &health.Scheduler,
	}
	allHealthy := true
	for _, obj := range rcs {
		rc := obj.(*kapi.ReplicationController)
		role := rc.Spec.Selector["role"]
		rcHealth, err := cc.healthyRC(rc)
		if err != nil {
			return false, nil, err
		}
		allHealthy = allHealthy && rcHealth
		if !rcHealth {
			glog.V(6).Infof("Cluster %q rc %q is not healthy", c.Metadata.Name, rc.Name)
		}
		if m, found := healthMapping[role]; found {
			*m = rcHealth
		}
	}

	return allHealthy, &health, nil
}

func (cc *clusterController) syncPendingCluster(c *api.Cluster) (*api.Cluster, error) {
	changedC, err := cc.pendingCheckTimeout(c)
	if err != nil {
		return nil, err
	}
	if changedC != nil {
		return changedC, nil
	}

	// create token-users first and also persist immediately because this
	// changes the cluster. The later secrets and other resources don't.
	changedC, err = cc.pendingCheckTokenUsers(c)
	if err != nil {
		return changedC, err
	}
	if changedC != nil {
		return changedC, nil
	}

	// check that all secrets are available
	err = cc.pendingCheckSecrets(c)
	if err != nil {
		return nil, err
	}

	// check that all services are available
	err = cc.pendingCheckServices(c)
	if err != nil {
		return nil, err
	}

	// check that all replication controllers are available
	err = cc.pendingCheckReplicationController(c)
	if err != nil {
		return nil, err
	}

	// check that all replication controllers are healthy
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, err
	}
	if health != nil && (c.Status.Health == nil ||
		!reflect.DeepEqual(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Metadata.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = time.Now()
		changedC = c
	}
	if !allHealthy {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", c.Metadata.Name, c.Status.Health)
		return changedC, nil
	}

	// no error until now? We are running.
	c.Status.Phase = api.RunningClusterStatusPhase
	c.Status.LastTransitionTime = time.Now()

	return c, nil
}
