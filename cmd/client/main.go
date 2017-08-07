package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"strings"

	"regexp"

	"net/url"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/uuid"
)

const (
	timeSleep  = time.Second * 5
	hostname   = "dev.kubermatic.io"
	outputPath = "/_artifacts/"
)
const (
	password    = "password"
	username    = "demo1@cluster"
	clientID    = "kubermatic"
	redirectURI = "http://localhost:8000/login"
)

var jwtToken = ""
var localRE = regexp.MustCompile(`/local\?req=([^\"]+)`)

type client struct {
	token          string
	baseURL        string
	kubeconfigFile string
	cluster        *api.Cluster
	client         *http.Client
	seeds          []api.Datacenter
}

func newClient(domain string, token string, outPath string) *client {
	return &client{
		token:          token,
		baseURL:        fmt.Sprintf("https://%s/api/v1", domain),
		kubeconfigFile: path.Join(outPath, "kubeconfig"),
		client:         &http.Client{},
	}
}

// setAuth sets the jwt token int a requests Authorization header field
func (c *client) setAuth(r *http.Request) {
	r.Header.Add("Authorization", "Bearer "+c.token)
}

type clusterRequest struct {
	Cloud   *api.CloudSpec   `json:"cloud"`
	Spec    *api.ClusterSpec `json:"spec"`
	SSHKeys []string         `json:"ssh_keys"`
}

func newClusterRequest() clusterRequest {
	return clusterRequest{
		Cloud: &api.CloudSpec{},
		Spec: &api.ClusterSpec{
			HumanReadableName: "e2e-" + uuid.ShortUID(4),
		},
	}
}

func (c *clusterRequest) applyDO() {
	c.Cloud.Name = "digitalocean"
	c.Cloud.Region = "do-ams2"
	c.Cloud.Secret = "a9c807e5951fb3a7d5541fe5ecbfafaaa2d1ea8a9f3804a837e21586ab9c198d"
	c.Cloud.Digitalocean = &api.DigitaloceanCloudSpec{}
}

type nodeRequest struct {
	Instances int          `json:"instances"`
	Spec      api.NodeSpec `json:"spec"`
}

func newNodeRequest(cl api.Cluster) *nodeRequest {
	return &nodeRequest{
		Instances: 1,
		Spec: api.NodeSpec{
			DatacenterName: cl.Spec.Cloud.DatacenterName,
		},
	}
}

func (n *nodeRequest) applyDO(cl api.Cluster) {
	n.Spec.Digitalocean = &api.DigitaloceanNodeSpec{
		Size: "2gb",
	}
}

// createNodes creates nodes
func (c *client) createNodes(nodeCount int, cluster api.Cluster) error {
	n := newNodeRequest(cluster)
	n.applyDO(cluster)
	return c.smartDo(fmt.Sprintf("/dc/%s/cluster/%s/node", cluster.Seed, cluster.Metadata.Name), n, nil)
}

func newSSHKey() *extensions.UserSSHKey {
	return &extensions.UserSSHKey{
		Name:      "e2e-test-key-" + uuid.ShortUID(4),
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDhtMw2eqE8vNitm9XZ6TAE5dL+pk3rLRA/39Pko0RRB1h6isevlAbG560t9vwAu7w3F59O0zbmbnN/0C0qcaz1sxZfdAPGCUESppYsxL2t7lhoaCCoK5pHXB8Iv3e8wPyuuugfXP0tS4oXI72tnmj9SJYCF3lxh02HLl2v0RsRto0Ojsx7anP98IcVsZWoRk3Xfh0UIoup2bwZ8F1DCtNrshu5pYr1zRklM7ANIrqzjHYjVwu/GGTkuUccEoiU8833hIHSd74Itdvk7p5iHeLRhu02rFLxCtG5BUiagpxg3ErvYMFrjQHO2wLggSRbKtqdWCSeAPV9Rf4GFSLtsBaXfUqb2PimAIPqXfMucEmUDWumWSbyZDPjZ+p7fLEI+BLsnT9NyFHjLqToUmYDz+a/8j8wt6iFC08/5z2SPu/71kEJlOYBgOW8KxhCotw1S07qnlvfdc4BXViXxeu9iYwVlv/257LQvmKzyfVqwMTouHw+jbNDOrFz+ozBs8frKYwXDuWDwzPyBDzkrloU8WUso1Mgiw/4vGCNx5x5yk7oAfzGjYlh3Dyvw/2SulpMuxoYnRkIlVVW6QYueFS4v+be/Ch6HkxBuqNZ2M8Z8X2GODaHIfAIlfWc8+xJNceAcSKou8Vda/LCSwHITl15TL0iKoWvlIutuXKOQ4gST81YQw== luk.burchard@gmail.com",
	}
}

func (c *client) createSSHKey(key *extensions.UserSSHKey) error {
	return c.smartDo("/ssh-keys", key, key)
}

// waitNS waits for the Namespace to get created.
func (c *client) waitNS(cl api.Cluster) error {
	for {
		var clusterState api.Cluster
		err := c.smartDo("/dc/"+cl.Seed+"/cluster/"+cl.Metadata.Name, nil, &clusterState)
		if err != nil {
			return err
		}

		// Check if cluster NS is created and running
		if clusterState.Address.URL != "" && clusterState.Status.Phase == api.RunningClusterStatusPhase {
			break
		}
		log.Println("Waiting for NS to get created ....")
		// Sleep until next check
		time.Sleep(timeSleep)
	}
	return nil
}

// deleteCluster deletes all clusters for a given user
func (c *client) deleteCluster(cluster api.Cluster) error {
	log.Printf("Deleting %q\n", cluster.Metadata.Name)
	req, err := http.NewRequest("DELETE", c.baseURL+"/dc/"+cluster.Seed+"/cluster/"+cluster.Metadata.Name, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)
	_, err = c.client.Do(req)
	return err
}

// up is the main entry point for the up method
func (c *client) up(nodes int) error {
	key := newSSHKey()
	err := c.createSSHKey(key)
	if err != nil {
		log.Fatalf("Couldn't post key: %v\n", err)
	}

	// Create cluster:
	// This creates the cluster, NS, and cloud provider
	log.Printf("Creating %d nodes", nodes)

	request := newClusterRequest()
	request.SSHKeys = []string{key.Metadata.Name}
	request.applyDO()

	var cluster api.Cluster
	err = c.smartDo("/cluster", request, &cluster)
	if err != nil {
		return err
	}

	// Wait for a Namespace:
	// wait for NS to not get errors when setting the cloud provider
	if err = c.waitNS(cluster); err != nil {
		log.Println(err)
		return err
	}

	// Create Nodes:
	// This is creating N nodes
	if err = c.createNodes(nodes, cluster); err != nil {
		log.Println(err)
		return err
	}

	c.cluster = &cluster

	return nil
}

func (c *client) getKubeConfig() ([]byte, error) {
	u := c.baseURL + "/dc/" + c.cluster.Seed + "/cluster/" + c.cluster.Metadata.Name + "/kubeconfig"
	u += "?token=" + c.token

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// God I hate this linter soo much!
	defer func() { err = resp.Body.Close(); _ = err }()
	return ioutil.ReadAll(resp.Body)
}

func (c *client) expandpath(apiPath string) string {
	return c.baseURL + apiPath
}

func (c *client) smartDo(apiPath string, data interface{}, into interface{}) error {
	var buf io.Reader
	var method string
	if data != nil {
		log.Printf("Preparing POST request under %q\n", c.expandpath(apiPath))
		d, err := json.Marshal(data)
		log.Printf("With data %s\n", string(d))
		buf = bytes.NewReader(d)
		if err != nil {
			return err
		}
		method = "POST"
	} else {
		log.Printf("Preparing GET request under %q\n", c.expandpath(apiPath))
		method = "GET"
	}

	// Create node request
	req, err := http.NewRequest(method, c.expandpath(apiPath), buf)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.client.Do(req)
	if into == nil {
		return err
	}
	if err != nil {
		return err
	}
	defer func() { err := resp.Body.Close(); _ = err }()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf("Got response %s\n", string(body))

	return json.Unmarshal(body, into)
}

// purge is the main entry point for the merge command
func (c *client) purge() error {
	waitAll := sync.WaitGroup{}
	for _, s := range c.seeds {
		var clusters []api.Cluster
		err := c.smartDo("/dc/"+s.Metadata.Name+"/cluster", nil, &clusters)
		if err != nil {
			return err
		}

		// Same pattern as in up
		done := make(chan struct{}, 5)
		waitAll.Add(len(clusters))
		for _, cluster := range clusters {
			go func(cl api.Cluster) {
				// Place ticket
				done <- struct{}{}
				err := c.deleteCluster(cl)
				if err != nil {
					log.Printf("Error deleting cluster %q, got error: %v", cl.Spec.HumanReadableName, err)
				}
				// Remove ticket
				<-done
				waitAll.Done()
			}(cluster)
		}
	}
	// Wait for all workers
	waitAll.Wait()
	return nil
}

func (c *client) updateSeeds() error {
	var dcs []api.Datacenter
	err := c.smartDo("/dc", nil, &dcs)
	if err != nil {
		return err
	}

	seeds := make([]api.Datacenter, 0)
	for _, dc := range dcs {
		if dc.Seed {
			seeds = append(seeds, dc)
		}
	}
	c.seeds = seeds
	return nil
}

func (c *client) writeKubeconfig() error {
	data, err := c.getKubeConfig()
	if err != nil {
		return err
	}
	log.Printf("Writing kubeconfig to %q\n", c.kubeconfigFile)
	log.Printf("With data %s\n", string(data))
	return ioutil.WriteFile(c.kubeconfigFile, data, 0666)
}

func errFatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func getToken() {
	cl := &http.Client{}
	authBaseUrl := "https://auth.int.kubermatic.io/"
	nonce := uuid.ShortUID(20)

	/*
	  First request to retrieve the Mail login option
	*/
	baseValues := url.Values{
		"response_type": {"id_token"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"scope":         {"openid"},
		"nonce":         {nonce},
	}

	u := authBaseUrl + "auth?" + baseValues.Encode()
	log.Println(u)
	resp, err := cl.Get(u)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	// Find the mail option
	match := localRE.FindSubmatch(data)
	if len(match) != 2 {
		panic(string(data))
	}
	reqUID := string(match[1])

	/*
	  Visit the mail option site
	*/
	localURL := authBaseUrl + "auth/local?req=" + reqUID
	formData := url.Values{"req": {reqUID}}
	cl.PostForm(localURL, formData)

	/*
	  Perform the login
	*/
	formData = url.Values{"login": {username}, "password": {password}}
	resp, err = cl.PostForm(localURL, formData)
	data, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	// Check for errors
	errorString := "Invalid username and password"
	if strings.Contains(string(data), errorString) {
		panic("Fucking wrong passowrd")
	}

	/*
	  Approve the login
	*/
	formData = url.Values{"req": {reqUID}, "approval": {"approve"}}
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err = cl.PostForm(authBaseUrl+"approval?req="+reqUID, formData)
	if err != nil {
		panic(err)
	}
	l, err := resp.Location()
	if err != nil {
		panic(err)
	}

	// Get the token
	q, err := url.ParseQuery(l.Fragment)
	if err != nil {
		panic(err)
	}
	log.Println(q.Get("id_token"))
	jwtToken = q.Get("id_token")
}

func main() {
	getToken()
	c := newClient(hostname, jwtToken, outputPath)

	if len(os.Args) != 2 {
		log.Printf("Wrong usage. Use:\n\n\t %s [up|purge]\n\n", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		errFatal(c.up(2))
		errFatal(c.writeKubeconfig())
	case "purge":
		errFatal(c.updateSeeds())
		errFatal(c.purge())
	}
}
