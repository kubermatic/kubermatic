package nanny

// KubeletInterface is an interface to manage a kubelet process
type KubeletInterface interface {
	WriteKubeConfig(n *Node, c *Cluster, path string) (err error)
	WriteStartConfig(n *Node, c *Cluster, path string) (err error)
	Start(c *Cluster) (err error)
	Stop(c *Cluster) (err error)
	Running(c *Cluster) (ok bool, err error)
}
