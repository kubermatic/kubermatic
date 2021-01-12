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

package dex

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Client is a Dex client that uses Dex' web UI to acquire an ID token.
type Client struct {
	// clientID is the OIDC client ID.
	ClientID string

	// redirectURI is one of the registered (allowed) redirect URLs,
	// used after a successful authentication. Even though there will
	// be no actual redirect to this URL, it needs to be a valid URL.
	RedirectURI string

	// providerURI is the actual Dex root URL ("homepage"), where users
	// can choose which authentication method they'd like to use. In
	// Dex' case, this is "<protocol>://<host>/dex/auth".
	ProviderURI string

	// client is the HTTP client to use.
	client *http.Client

	// log is our logger
	log *zap.SugaredLogger
}

// NewClient creates a new OIDC client. See the Client struct for definitions on the parameters.
func NewClient(clientID string, redirectURI string, providerURI string, log *zap.SugaredLogger) (*Client, error) {
	httpClient := &http.Client{}
	httpClient.Timeout = 5 * time.Second

	return &Client{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		ProviderURI: providerURI,
		client:      httpClient,
		log:         log.With("client-id", clientID, "provider-uri", providerURI),
	}, nil
}

func (c *Client) Login(ctx context.Context, login string, password string) (string, error) {
	var (
		accessToken string
		err         error
	)

	if err := wait.PollImmediate(3*time.Second, 1*time.Minute, func() (bool, error) {
		accessToken, err = c.tryLogin(ctx, login, password)
		if err != nil {
			c.log.Debugw("Failed to login", zap.Error(err))
			return false, nil
		}

		return true, nil
	}); err != nil {
		return "", err
	}

	return accessToken, nil
}

func (c *Client) tryLogin(ctx context.Context, login string, password string) (string, error) {
	c.log.Debug("Attempting login")

	// fetch login page and acquire the nonce
	loginURL, err := c.fetchLoginURL(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to determine login URL: %v", err)
	}

	c.log.Debugw("Login URL detected", "url", loginURL.String())

	// post the credentials to the login URL and observe the response's Location header
	token, err := c.authenticate(ctx, loginURL, login, password)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate as %q: %v", login, err)
	}

	c.log.Debug("Login successful")

	return token, nil
}

func (c *Client) fetchLoginURL(ctx context.Context) (*url.URL, error) {
	// quick&dirty URL clone, so we don't change the u argument
	loginURL, err := url.Parse(c.ProviderURI)
	if err != nil {
		return nil, fmt.Errorf("invalid provider URL %q: %v", c.ProviderURI, err)
	}

	params := loginURL.Query()
	params.Set("client_id", c.ClientID)
	params.Set("redirect_uri", c.RedirectURI)
	params.Set("response_type", "id_token")
	params.Set("scope", "openid profile email")
	params.Set("nonce", "not-actually-a-nonce")
	// make sure we are redirected and not greeted with a "choose your login method page"
	params.Set("connector_id", "local")
	loginURL.RawQuery = params.Encode()

	c.log.Debugw("Fetching OIDC login page", "url", loginURL.String())

	req, err := http.NewRequest("GET", loginURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for login page: %v", err)
	}

	// Dex will redirect us to the login page with a code attached to the URL;
	// the code is all we want to have, so instead of parsing the HTML, we just
	// inspect the Location header. For this reason we do not follow any redirects.
	c.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		c.log.Debugw("Rejecting redirect", "location", req.URL.String())
		return http.ErrUseLastResponse
	}

	rsp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch login page: %v", err)
	}
	defer rsp.Body.Close()

	location, err := c.getLocation(rsp)
	if err != nil {
		return nil, fmt.Errorf("response headers did not contain a valid Location header: %v", err)
	}

	c.log.Debugw("Found Location header", "location", location.String())

	return loginURL.ResolveReference(location), nil
}

func (c *Client) authenticate(ctx context.Context, loginURL *url.URL, login string, password string) (string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	if err := writer.WriteField("login", login); err != nil {
		return "", fmt.Errorf("failed to add login field to request body: %v", err)
	}
	if err := writer.WriteField("password", password); err != nil {
		return "", fmt.Errorf("failed to add password field to request body: %v", err)
	}

	err := writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to encode request body as multipart: %v", err)
	}

	c.log.Debugw("Sending login request", "url", loginURL.String(), "login", login)

	// prepare request
	req, err := http.NewRequest("POST", loginURL.String(), buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Dex uses a couple of internal redirects that we must follow, but the last
	// redirect to the redirectURI is the one we want to intercept and not follow,
	// because there is no server running that listens on the redirectURI.
	c.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// if we stay on the same host, keep following the redirect
		if req.URL.Host == "" || req.URL.Host == loginURL.Host {
			c.log.Debugw("Allowing redirect", "location", req.URL.String())
			return nil
		}

		// otherwise stop
		c.log.Debugw("Stopping redirect chain", "location", req.URL.String())
		return http.ErrUseLastResponse
	}

	// send request
	rsp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to perform authentication request: %v", err)
	}
	defer rsp.Body.Close()

	// evaluate location header
	location, err := c.getLocation(rsp)
	if err != nil {
		return "", fmt.Errorf("response headers did not contain a valid Location header, are the credentials correct?: %v", err)
	}

	c.log.Debugw("Found Location header")

	// extract token from the location URL's fragment
	query, err := url.ParseQuery(location.Fragment)
	if err != nil {
		return "", fmt.Errorf("Location header does not contain a valid query string in its fragment (%q): %v", location.Fragment, err)
	}

	token := query.Get("id_token")
	if token == "" {
		return "", fmt.Errorf("no ID token found, something bad has happened; query was %v", query)
	}

	return token, nil
}

func (c *Client) getLocation(response *http.Response) (*url.URL, error) {
	// evaluate location header
	if response.StatusCode < 300 || response.StatusCode >= 400 {
		return nil, fmt.Errorf("expected a redirect response, but got status code %d", response.StatusCode)
	}

	location := response.Header.Get("Location")
	if len(location) == 0 {
		return nil, fmt.Errorf("found a redirect (status %d), but has no Location header; this seems like a bug in Dex?", response.StatusCode)
	}

	loc, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("the final Location header %q is not a valid URL: %v", location, err)
	}

	return loc, nil
}
