package client

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/uuid"
)

const timeSleep = time.Second * 5

var jwtFlag string
var auth0Domain = ""

// setAuth sets the jwt token int a requests Authorization header field
func setAuth(r *http.Request) {
	r.Header.Add("Authorization", "Bearer "+jwtFlag)
}

var (
	dcFlag     = "us-central1"
	domainFlag = "dev.kubermatic.io"
)

type clusterRequest struct {
	Cloud   *api.CloudSpec   `json:"cloud"`
	Spec    *api.ClusterSpec `json:"spec"`
	SSHKeys []string         `json:"ssh_keys"`
}

func NewClusterRequest() clusterRequest {
	return clusterRequest{
		Cloud: &api.CloudSpec{},
		Spec: &api.ClusterSpec{
			HumanReadableName: "e2e-" + uuid.ShortUID(4),
		},
	}
}

func (c *clusterRequest) applyAWS() {
	c.Cloud.Name = "aws"
	c.Cloud.Region = "aws-us-west-2a"
	c.Cloud.User = "AKIAIF5EOAWOD4BLMJGA"
	c.Cloud.Secret = "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"
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

func NewNodeRequest(cl api.Cluster) *nodeRequest {
	return &nodeRequest{
		Instances: 1,
		Spec: api.NodeSpec{
			DatacenterName: cl.Spec.Cloud.DatacenterName,
		},
	}
}

func (n *nodeRequest) applyAWS(cl api.Cluster) {
	panic("Not implemented")
}

func (n *nodeRequest) applyDO(cl api.Cluster) {
	n.Spec.Digitalocean = &api.DigitaloceanNodeSpec{
		Size:               "2gb",
		SSHKeyFingerprints: []string{"80:ba:7a:3b:3f:89:b1:b4:cd:b8:b4:fb:6c:a4:62:d0", "dd:c1:43:1a:fe:cb:9c:3f:48:20:78:c8:fe:cf:d5:a8", "79:cc:81:d6:7a:d5:2b:db:1b:c6:68:15:6e:4f:44:05", "b0:1c:92:9b:d7:25:33:a3:82:5f:60:b4:15:52:fb:d5", "be:b4:1b:c0:ad:01:9f:ef:d7:52:00:6b:69:e9:95:f2", "61:e9:45:14:67:94:8a:c9:d6:5e:6f:8c:4a:0b:51:f9", "65:f2:d1:22:d0:af:a6:4c:1c:b1:9d:cb:aa:39:2f:99", "ef:dc:8b:66:b0:f0:60:63:ea:57:75:fe:6e:1e:01:c1"},
	}
}

// createNodes creates nodes
func createNodes(nodeCount int, cluster api.Cluster) error {
	n := NewNodeRequest(cluster)
	n.applyDO(cluster)
	buf, err := json.Marshal(n)
	if err != nil {
		return err
	}

	// Create node request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/api/v1/dc/%s/cluster/%s/node", domainFlag, cluster.Seed, cluster.Metadata.Name), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	return err
}

func NewSSHKey() *extensions.UserSSHKey {
	return &extensions.UserSSHKey{
		Name:      "e2e-test-key-" + uuid.ShortUID(4),
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDhtMw2eqE8vNitm9XZ6TAE5dL+pk3rLRA/39Pko0RRB1h6isevlAbG560t9vwAu7w3F59O0zbmbnN/0C0qcaz1sxZfdAPGCUESppYsxL2t7lhoaCCoK5pHXB8Iv3e8wPyuuugfXP0tS4oXI72tnmj9SJYCF3lxh02HLl2v0RsRto0Ojsx7anP98IcVsZWoRk3Xfh0UIoup2bwZ8F1DCtNrshu5pYr1zRklM7ANIrqzjHYjVwu/GGTkuUccEoiU8833hIHSd74Itdvk7p5iHeLRhu02rFLxCtG5BUiagpxg3ErvYMFrjQHO2wLggSRbKtqdWCSeAPV9Rf4GFSLtsBaXfUqb2PimAIPqXfMucEmUDWumWSbyZDPjZ+p7fLEI+BLsnT9NyFHjLqToUmYDz+a/8j8wt6iFC08/5z2SPu/71kEJlOYBgOW8KxhCotw1S07qnlvfdc4BXViXxeu9iYwVlv/257LQvmKzyfVqwMTouHw+jbNDOrFz+ozBs8frKYwXDuWDwzPyBDzkrloU8WUso1Mgiw/4vGCNx5x5yk7oAfzGjYlh3Dyvw/2SulpMuxoYnRkIlVVW6QYueFS4v+be/Ch6HkxBuqNZ2M8Z8X2GODaHIfAIlfWc8+xJNceAcSKou8Vda/LCSwHITl15TL0iKoWvlIutuXKOQ4gST81YQw== luk.burchard@gmail.com",
	}
}

func createSSHKey(key *extensions.UserSSHKey) error {
	buf, err := json.Marshal(key)
	if err != nil {
		return err
	}
	fmt.Printf("Submitting key %s\n", string(buf))
	req, err := http.NewRequest("POST", "https://"+domainFlag+"/api/v1/ssh-keys", bytes.NewReader(buf))
	if err != nil {
		println("1")
		return err
	}
	setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		println("2")
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("3")
		return err
	}

	var newKey extensions.UserSSHKey
	fmt.Printf("Got key %s\n", string(data))
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&newKey)
	if err != nil {
		fmt.Printf("%v\n", err)
		println("4")
		return err
	}
	println("DONE")

	*key = newKey
	return nil
}

// waitNS waits for the Namespace to get created.
func waitNS(cl api.Cluster) error {
	for {
		req, err := http.NewRequest("GET", "https://"+domainFlag+"/api/v1/dc/"+cl.Seed+"/cluster/"+cl.Metadata.Name, nil)
		if err != nil {
			return err
		}
		setAuth(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		// Read all Data from the body,
		// We cloud also use a tee reader but this would have been a bit overkill.
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// We use the cluster state to get notified when a new NS gets created.
		// When a NS gets created the cluster will revice an cluster URL.
		var clusterState api.Cluster
		if err = json.NewDecoder(bytes.NewReader(data)).Decode(&clusterState); err != nil {
			log.Println(string(data))
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
func deleteCluster(cluster api.Cluster, client *http.Client) error {
	log.Printf("Deleting %q\n", cluster.Metadata.Name)
	req, err := http.NewRequest("DELETE", "https://"+domainFlag+"/api/v1/dc/"+dcFlag+"/cluster/"+cluster.Metadata.Name, nil)
	if err != nil {
		return err
	}
	setAuth(req)
	_, err = client.Do(req)
	return err
}

// up is the main entry point for the up method
func up(nodes int, typ string) ([]byte, error) {

	key := NewSSHKey()
	err := createSSHKey(key)
	if err != nil {
		log.Fatalf("Couldn't post key: %v\n", err)
	}

	// Create cluster:
	// This creates the cluster, NS, and cloud provider
	log.Printf("Creating %d nodes", nodes)

	// Test AWS or DO
	request := NewClusterRequest()
	request.SSHKeys = []string{key.Metadata.Name, "80:ba:7a:3b:3f:89:b1:b4:cd:b8:b4:fb:6c:a4:62:d0", "dd:c1:43:1a:fe:cb:9c:3f:48:20:78:c8:fe:cf:d5:a8", "79:cc:81:d6:7a:d5:2b:db:1b:c6:68:15:6e:4f:44:05", "b0:1c:92:9b:d7:25:33:a3:82:5f:60:b4:15:52:fb:d5", "be:b4:1b:c0:ad:01:9f:ef:d7:52:00:6b:69:e9:95:f2", "61:e9:45:14:67:94:8a:c9:d6:5e:6f:8c:4a:0b:51:f9", "65:f2:d1:22:d0:af:a6:4c:1c:b1:9d:cb:aa:39:2f:99", "ef:dc:8b:66:b0:f0:60:63:ea:57:75:fe:6e:1e:01:c1"}
	if strings.EqualFold(typ, "aws") {
		request.applyAWS()
	} else {
		request.applyDO()
	}

	buf, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("Couldn't serialize to json: %v\n", err)
	}
	log.Printf("Creating cluster: %s\n", string(buf))

	req, err := http.NewRequest("POST", "https://"+domainFlag+"/api/v1/cluster", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var cluster api.Cluster
	if err = json.NewDecoder(bytes.NewReader(data)).Decode(&cluster); err != nil {
		log.Println(err)
		return nil, err
	}

	// Wait for a Namespace:
	// wait for NS to not get errors when setting the cloud provider
	if err = waitNS(cluster); err != nil {
		log.Println(err)
		return nil, err
	}

	// Create Nodes:
	// This is creating N nodes
	if err = createNodes(nodes, cluster); err != nil {
		log.Println(err)
		return nil, err
	}

	return getKubeConfig(cluster)
}

func getKubeConfig(cl api.Cluster) ([]byte, error) {
	u := "https://" + domainFlag + "/api/v1/dc/" + cl.Seed + "/cluster/" + cl.Metadata.Name + "/kubeconfig"
	u += "?token=" + jwtFlag

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	setAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// purge is the main entry point for the merge command
func purge() error {
	client := &http.Client{}

	// Get clusters list
	req, err := http.NewRequest("GET", "https://"+domainFlag+"/api/v1/dc/"+dcFlag+"/cluster", nil)
	if err != nil {
		return err
	}

	setAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	var clusters []api.Cluster
	if err = json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return err
	}

	// Same pattern as in up
	done := make(chan struct{}, 5)
	waitAll := sync.WaitGroup{}
	waitAll.Add(len(clusters))
	for _, cluster := range clusters {
		go func(cl api.Cluster) {
			// Place ticket
			done <- struct{}{}
			log.Println(deleteCluster(cl, client))
			// Remove ticket
			<-done
			waitAll.Done()
		}(cluster)
	}
	// Wait for all workers
	waitAll.Wait()
	return nil
}

func getBearer(audience_domain, client_id, client_secret string) string {
	request := struct {
		GrantType    string `json:"grant_type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Audience     string `json:"audience"`
	}{
		"client_credentials",
		client_id,
		client_secret,
		audience_domain,
	}

	url := "https://" + auth0Domain + "/oauth/token"

	reqBody, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("Cant get Bearer: %v\n", err)
	}

	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	var response struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalln("Error unmarshalling Auth0 response")
	}
	return response.TokenType + " " + response.AccessToken
}

func main() {
	flag.Parse()
	printError := func() {
		log.Printf("Wrong usage. Use:\n\n\t %s [up [aws]|purge]\n\n", os.Args[0])
		os.Exit(1)
	}

	if len(flag.Args()) < 1 {
		printError()
	}

	//jwtFlag = getBearer("", "", "")
	jwtFlag = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhcHBfbWV0YWRhdGEiOnsicm9sZXMiOlsidXNlciJdfSwiaXNzIjoiaHR0cHM6Ly9rdWJlcm1hdGljLmV1LmF1dGgwLmNvbS8iLCJzdWIiOiJnaXRodWJ8NzM4NzcwMyIsImF1ZCI6InpxYUdBcUJHaVdENnRjZTdmY0hMMDNRWllpMUFDOXdGIiwiZXhwIjoxNDk0OTUzNTUxLCJpYXQiOjE0OTQ5MTc1NTF9.FVa-z88AMMEA5jg-Ud5SOG8U7kQ1foSUamDLbZbxNn4"

	var err error
	switch flag.Arg(0) {
	case "up":
		out, _ := up(1, flag.Arg(1))
		fmt.Printf("%s", string(out))
	case "purge":
		err = purge()
	default:
		printError()
	}

	if err != nil {
		log.Fatal(err)
	}
}
