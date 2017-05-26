package nanny

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/coreos/go-systemd/login1"
	"github.com/golang/glog"
)

// ProviderInterface is the client to talk to the provider api
type ProviderInterface interface {
	CreateNode(n *Node) (*Node, error)
	GetNode(UID string) (*Node, error)
	GetClusterConfig(n *Node) (*Cluster, error)
}

// NewProvider creates a new provider instance
func NewProvider(client *http.Client, endpoint string, authUser string, authPass string) ProviderInterface {
	return &provider{
		client:   client,
		endpoint: endpoint,
		authUser: authUser,
		authPass: authPass,
	}
}

var (
	// ErrNotAssigned tells that the node is not assigned to a cluster
	ErrNotAssigned = errors.New("invalid argument")
)

const (
	notFoundText = "Node does not exist"
)

// provider is the default provider implementation to use in main code
type provider struct {
	client   *http.Client
	endpoint string
	authUser string
	authPass string
}

type simpleResponse struct {
	Body       []byte
	StatusCode int
}

func (p *provider) Do(method, urlStr string, rbody io.Reader) (*simpleResponse, error) {
	request, err := http.NewRequest(method, urlStr, rbody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	if p.authUser != "" || p.authPass != "" {
		request.SetBasicAuth(p.authUser, p.authPass)
	}

	glog.V(4).Infof("Executing %s request to %s%s", request.Method, request.Host, request.RequestURI)
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to provider api: %v", err)
	}
	defer func(response *http.Response) {
		err := response.Body.Close()
		if err != nil {
			glog.Errorf("Error closing body buffer: %v", err)
		}
	}(response)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	glog.V(6).Infof("Response: Status: %d Body: %s", response.StatusCode, string(body))

	return &simpleResponse{
		Body:       body,
		StatusCode: response.StatusCode,
	}, nil
}

func (p *provider) getURL(raw string) string {
	u, err := url.Parse(p.endpoint)
	if err != nil {
		glog.Errorf("Failed to parse given URL: %v", err)
	}

	u.Path = path.Join(u.Path, raw)
	return u.String()
}

// CreateNode creates a node at the provider api
func (p *provider) CreateNode(n *Node) (n2 *Node, err error) {
	b := bytes.NewBuffer([]byte{})
	err = json.NewEncoder(b).Encode(&n)
	if err != nil {
		return nil, fmt.Errorf("failed encode node to json: %v", err)
	}

	r, err := p.Do("POST", p.getURL("/nodes"), b)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	if r.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("got unexpected statuscode %d. Expected %d", r.StatusCode, http.StatusCreated)
	}

	if err = json.Unmarshal(r.Body, &n2); err != nil {
		return nil, fmt.Errorf("error decoding json body: %v", err)
	}

	return
}

// GetNode retrieves a node from the provider api
func (p *provider) GetNode(UID string) (n *Node, err error) {
	r, err := p.Do("GET", p.getURL(fmt.Sprintf("/nodes/%s", UID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	if r.StatusCode != http.StatusOK {
		if r.StatusCode == http.StatusNotFound {
			return
		}

		err = fmt.Errorf("got unexpected statuscode %d. Expected %d", r.StatusCode, http.StatusOK)
		return
	}

	if err = json.Unmarshal(r.Body, &n); err != nil {
		return nil, fmt.Errorf("error decoding json body: %v", err)
	}

	return
}

// GetClusterConfig loads the current cluster config from the api
func (p *provider) GetClusterConfig(n *Node) (c *Cluster, err error) {
	// Being explicit about c
	c = nil
	r, err := p.Do("GET", p.getURL(fmt.Sprintf("/nodes/%s/cluster", n.UID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	if r.StatusCode == http.StatusNotFound {
		if !strings.HasPrefix(string(r.Body), notFoundText) {
			return nil, ErrNotAssigned
		}

		conn, err := login1.New()
		if err != nil {
			return nil, err
		}
		conn.Reboot(false)
		os.Exit(1)
	}

	if r.StatusCode != http.StatusOK {
		err = fmt.Errorf("got unexpected statuscode %d. Expected %d", r.StatusCode, http.StatusOK)
		return nil, err
	}

	if err = json.Unmarshal(r.Body, &c); err != nil {
		return nil, fmt.Errorf("error decoding json body: %v", err)
	}

	b, err := base64.StdEncoding.DecodeString(c.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 encoded kubeconfig %q failed: %v", c.KubeConfig, err)
	}
	c.KubeConfig = string(b)

	return c, nil
}

// IsNotAssigned tells if the error means that the node is not assigned to a cluster
func IsNotAssigned(err error) bool {
	return err == ErrNotAssigned
}
