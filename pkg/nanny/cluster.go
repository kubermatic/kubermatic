package nanny

// Cluster contains a join configuration to pass to kubelet
type Cluster struct {
	Name               string `json:"name"`
	APIServerURL       string `json:"apiserver_url"`
	KubeConfig         string `json:"kubeconfig"`
	ApiserverSSHPubKey string `json:"apiserver_ssh_pub_key"`
}
