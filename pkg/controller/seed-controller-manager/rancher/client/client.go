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

package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func New(opts Options) (*Client, error) {
	if len(opts.Endpoint) == 0 {
		return nil, fmt.Errorf("rancher server endpoint can't be empty")
	}
	if opts.AccessKey == "" || opts.SecretKey == "" {
		return nil, fmt.Errorf("access/secret must be provided")
	}
	c := getHTTPClient(opts.Insecure)
	client := &Client{
		options: opts,
		client:  &c,
	}

	if err := client.login(); err != nil {
		return nil, fmt.Errorf("can't login to rancher server: %v", err)
	}

	return client, nil
}

func (c *Client) login() error {
	urlStr := fmt.Sprintf("%s/v3-public/localProviders/local?action=login", c.options.Endpoint)
	msg := map[string]interface{}{
		"description":  "",
		"username":     c.options.AccessKey,
		"password":     c.options.SecretKey,
		"responseType": "json",
		"ttl":          0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %v", err)
	}
	resp, err := c.client.Post(urlStr, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		return fmt.Errorf("http post request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("login failed with status: %v", resp.StatusCode)
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode server response: %v", err)
	}
	token, ok := data["token"].(string)
	if !ok {
		return fmt.Errorf("can't find rancher token")
	}
	c.options.Token = token
	return nil
}

func (c *Client) do(urlStr, data string, into interface{}) error {
	if c.client == nil {
		return fmt.Errorf("invalid http client")
	}
	if urlStr == "" {
		return fmt.Errorf("empty URL")
	}
	var err error
	var req *http.Request
	if data != "" {
		req, err = http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data))
	} else {
		req, err = http.NewRequest(http.MethodGet, urlStr, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Add("Authorization", c.getAuthHeader())

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()
	if !isHTTPOK(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: [%d %s]: %v", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}
	if into == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(&into)

}

func (c *Client) getAuthHeader() string {
	if c.options.Token != "" {
		return fmt.Sprintf("Bearer %s", c.options.Token)
	}
	return ""
}

func isHTTPOK(s int) bool {
	return s >= 200 && s < 300
}

func appendFilters(urlStr string, filters Filters) (string, error) {
	if filters == nil {
		return urlStr, nil
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range filters {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func getHTTPClient(insecure bool) http.Client {
	tr := http.DefaultTransport
	if insecure {
		tr.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return http.Client{
		Transport: tr,
	}
}
