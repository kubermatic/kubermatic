package nanny

import (
	"reflect"
	"time"

	"github.com/golang/glog"
)

const (
	userApiserver = "apiserver"
)

//StartCheckLoop starts the nanny routine, the loop to sync the kubelet state with the provider endpoint
func StartCheckLoop(k KubeletInterface, p ProviderInterface, n *Node, reload time.Duration) {
	glog.Infof("Starting check loop for node %q", n.UID)

	checkNode(n, p)

	var currentConfig *Cluster

	for {
		time.Sleep(reload)

		c, err := p.GetClusterConfig(n)

		// Test if GetClusterConfig succseded
		if err != nil || c == nil {
			if IsNotAssigned(err) {
				glog.Info("Not assigned to a cluster")
			} else {
				glog.Errorf("Error fetching cluster config for node %q: %v", n.UID, err)
			}

			if currentConfig == nil {
				continue
			}
		}

		// Test if current context has updated
		if currentConfig != nil && reflect.DeepEqual(c, currentConfig) {
			glog.Infof("Cluster config didn't change, skipping actions")
			continue
		}

		// Renew current context
		currentConfig = c
		glog.Info("Syncing new config to kubelet service ...")
		syncClusterState(k, n, currentConfig)
	}
}

func checkNode(n *Node, p ProviderInterface) {
	knownNode, err := p.GetNode(n.UID)

	if err != nil {
		glog.Fatalf("Error reading node %q from provider endpoint: %v", n.UID, err)
	}
	if knownNode == nil {
		glog.Infof("Node %q not found at endpoint. Will create it", n.UID)

		_, err := p.CreateNode(n)
		if err != nil {
			glog.Fatalf("Error creating node %q at provider endpoint: %v", n.UID, err)
		}

		glog.Infof("Node %q created", n.UID)
	}

	glog.Infof("Node %q checked.", n.UID)
}

func syncClusterState(k KubeletInterface, n *Node, c *Cluster) {
	if c == nil {
		glog.Infof("Syncing no cluster to kubelet ...")
		if err := k.Stop(c); err != nil {
			glog.Errorf("Error stopping systemd unit: %v", err)
		}

		return
	}

	apiSSHPubKeyManager, err := NewUserSSHKeyManager(userApiserver)
	if err != nil {
		glog.Errorf("could not initialize SSHPubKeyManager for user %q: %v", userApiserver, err)
	} else {
		keys, err := apiSSHPubKeyManager.AddPubKey(c.ApiserverSSHPubKey, true)
		if err != nil {
			glog.Errorf("could not add ssh pub key for user %q: %v", userApiserver, err)
		} else {
			glog.Infof("installed ssh pub key for user %q:", userApiserver)
			glog.Info("installed keys:", keys)
		}
	}

	glog.Infof("Synching cluster %q to kubelet ...", c.Name)
	if err := k.WriteKubeConfig(n, c, DefaultKubeConfigPath); err != nil {
		glog.Errorf("Error writing KubeConfig file: %v", err)
	}

	if err := k.WriteStartConfig(n, c, DefaultSystemdUnitFilePath); err != nil {
		glog.Errorf("Error writing systemd unit config file: %v", err)
	}

	if err := k.Start(c); err != nil {
		glog.Errorf("Error starting systemd unit: %v", err)
	}

	// try 5 times and check if the kubelet is running
	for try := 0; try < 5; try++ {
		// sleep 3 seconds between checks
		time.Sleep(5 * time.Second)
		glog.Infof("Doing check #%d on systemd unit status", try)

		if ok, err := k.Running(c); err != nil {
			glog.Errorf("Error gettting systemd unit status: %v", err)
		} else if !ok {
			glog.Errorf("Error: kubelet failed to start!")
		}
	}
}
