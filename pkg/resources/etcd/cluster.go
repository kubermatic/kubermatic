/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/v3/etcdserver/etcdserverpb"
	"go.etcd.io/etcd/v3/pkg/transport"

	"k8c.io/kubermatic/v2/pkg/resources"
)

const (
	MetricsPort = 2378
	ClientPort  = 2379
	PeersPort   = 2380
)

func hostname(n int, namespace string, port int) string {
	return fmt.Sprintf("etcd-%d.etcd.%s.svc.cluster.local:%d", n, namespace, port)
}

func LocalEndpoint() string {
	return fmt.Sprintf("https://127.0.0.1:%d", ClientPort)
}

func ClientEndpoints(members int, namespace string) []string {
	endpoints := []string{}
	for i := 0; i < members; i++ {
		endpoints = append(endpoints, fmt.Sprintf("https://%s", hostname(i, namespace, ClientPort)))
	}
	return endpoints
}

func PeerURLs(members int, namespace string) []string {
	urls := []string{}
	for i := 0; i < members; i++ {
		urls = append(urls, hostname(i, namespace, PeersPort))
	}
	return urls
}

func MembersList(members int, namespace string) []string {
	list := PeerURLs(members, namespace)
	for i, member := range list {
		list[i] = fmt.Sprintf("etcd-%d=%s", i, member)
	}
	return list
}

type Client struct {
	client  *clientv3.Client
	tlsInfo *transport.TLSInfo
}

func NewLocalClient(tlsInfo *transport.TLSInfo) (*Client, error) {
	return NewClient([]string{LocalEndpoint()}, tlsInfo)
}

func NewClient(endpoints []string, tlsInfo *transport.TLSInfo) (*Client, error) {
	if tlsInfo == nil {
		tlsInfo = &transport.TLSInfo{
			CertFile:       resources.EtcdClientCertFile,
			KeyFile:        resources.EtcdClientKeyFile,
			TrustedCAFile:  resources.EtcdTrustedCAFile,
			ClientCertAuth: true,
		}
	}

	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS client config: %v", err)
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build etcd client: %v", err)
	}

	return &Client{
		client:  client,
		tlsInfo: tlsInfo,
	}, nil
}

func (c *Client) MemberList(ctx context.Context) ([]*etcdserverpb.Member, error) {
	resp, err := c.client.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Members, err
}

func (c *Client) Close() error {
	return c.client.Close()
}

// Healthy checks if the cluster is healthy. If a member is given, only this member's
// health is checked.
func (c *Client) Healthy(ctx context.Context, member *etcdserverpb.Member) (bool, error) {
	client := c.client

	if member != nil {
		clientURLs := member.ClientURLs
		if len(clientURLs) == 0 {
			return false, errors.New("member has no client URLs; was it started already?")
		}

		cluster, err := NewClient(clientURLs[len(clientURLs)-1:], c.tlsInfo)
		if err != nil {
			// swallow any connection error at this point
			return false, nil
		}

		client = cluster.client
		defer client.Close()
	}

	// just get a key from etcd, this is how `etcdctl endpoint health` works!
	_, err := client.Get(ctx, "healthy")

	// silently swallow/drop transient errors
	return err == nil || err == rpctypes.ErrPermissionDenied, nil
}

func (c *Client) IsClusterMember(ctx context.Context, name string) (bool, error) {
	members, err := c.MemberList(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list cluster members: %v", err)
	}

	for i, member := range members {
		if member.Name == name {
			return true, nil
		}

		// if the member is not started yet, its name would be empty,
		// in that case, we match for peerURL host.
		url, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return false, fmt.Errorf("failed to parse member %d peer URL: %v", i, err)
		}

		if url.Host == name {
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) MemberAdd(ctx context.Context, peerURLs []string) error {
	_, err := c.client.MemberAdd(ctx, peerURLs)
	return err
}

func (c *Client) MemberUpdate(ctx context.Context, memberID uint64, peerURLs []string) error {
	_, err := c.client.MemberUpdate(ctx, memberID, peerURLs)
	return err
}

func (c *Client) MemberRemove(ctx context.Context, memberID uint64) error {
	_, err := c.client.MemberRemove(ctx, memberID)
	return err
}

func (c *Client) Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error) {
	return c.client.Status(ctx, endpoint)
}
