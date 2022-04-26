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

package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	wsh "k8c.io/kubermatic/v2/pkg/handler/websocket"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	kubermaticcontext "k8c.io/kubermatic/v2/pkg/util/context"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/util/hash"
	"k8c.io/kubermatic/v2/pkg/watcher"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header["Origin"]
		if len(origin) == 0 {
			return true
		}

		u, err := url.Parse(origin[0])
		if err != nil {
			return false
		}

		if u.Host == r.Host {
			return true
		}

		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			return false
		}

		if u.Hostname() == host {
			return true
		}

		return false
	},
}

type WebsocketSettingsWriter func(ctx context.Context, providers watcher.Providers, ws *websocket.Conn)
type WebsocketUserWriter func(ctx context.Context, providers watcher.Providers, ws *websocket.Conn, userEmail string)
type WebsocketTerminalWriter func(ctx context.Context, providers watcher.Providers, ws *websocket.Conn, seedClient kubernetes.Interface, seedCfg *rest.Config, namespace, podName string)

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	providers := getProviders(r)

	mux.HandleFunc("/ws/admin/settings", getSettingsWatchHandler(wsh.WriteSettings, providers, r))
	mux.HandleFunc("/ws/me", getUserWatchHandler(wsh.WriteUser, providers, r))
	mux.HandleFunc("/ws/projects/{project_id}/clusters/{cluster_id}/terminal", getTerminalWatchHandler(wsh.Terminal, providers, r))
}

func getProviders(r Routing) watcher.Providers {
	return watcher.Providers{
		SettingsProvider:          r.settingsProvider,
		SettingsWatcher:           r.settingsWatcher,
		UserProvider:              r.userProvider,
		UserWatcher:               r.userWatcher,
		MemberMapper:              r.userProjectMapper,
		ProjectProvider:           r.projectProvider,
		PrivilegedProjectProvider: r.privilegedProjectProvider,
		UserInfoGetter:            r.userInfoGetter,
		SeedsGetter:               r.seedsGetter,
		ClusterProviderGetter:     r.clusterProviderGetter,
	}
}

func getSettingsWatchHandler(writer WebsocketSettingsWriter, providers watcher.Providers, routing Routing) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := verifyAuthorizationToken(req, routing.tokenVerifiers, routing.tokenExtractors)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		go writer(req.Context(), providers, ws)
		requestLoggingReader(ws)
	}
}

func getUserWatchHandler(writer WebsocketUserWriter, providers watcher.Providers, routing Routing) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		user, err := verifyAuthorizationToken(req, routing.tokenVerifiers, routing.tokenExtractors)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		go writer(req.Context(), providers, ws, user.Email)
		requestLoggingReader(ws)
	}
}

func getTerminalWatchHandler(writer WebsocketTerminalWriter, providers watcher.Providers, routing Routing) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		authenticatedUser, err := verifyAuthorizationToken(req, routing.tokenVerifiers, routing.tokenExtractors)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		clusterID, err := common.DecodeClusterID(ctx, req)
		if err != nil {
			return
		}

		projectReq, err := common.DecodeProjectRequest(ctx, req)
		if err != nil {
			return
		}
		projectID := projectReq.(common.ProjectReq).ProjectID

		request := terminalReq{
			ClusterID: clusterID,
		}

		clusterProvider, ctx, err := middleware.GetClusterProvider(ctx, request, providers.SeedsGetter, providers.ClusterProviderGetter)
		if err != nil {
			return
		}
		privilegedClusterProvider := clusterProvider.(provider.PrivilegedClusterProvider)

		user, err := providers.UserProvider.UserByEmail(ctx, authenticatedUser.Email)
		if err != nil {
			return
		}
		ctx = context.WithValue(ctx, middleware.ClusterProviderContextKey, clusterProvider)
		ctx = context.WithValue(ctx, middleware.PrivilegedClusterProviderContextKey, privilegedClusterProvider)
		ctx = context.WithValue(ctx, kubermaticcontext.UserCRContextKey, user)

		_, err = handlercommon.GetCluster(ctx, providers.ProjectProvider, providers.PrivilegedProjectProvider, providers.UserInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return
		}

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		k8sClusterProvider := privilegedClusterProvider.(*kubernetesprovider.ClusterProvider)

		namespace, podName, err := getWebTerminalNamespaceAndPodName(ctx, privilegedClusterProvider, clusterID)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		writer(ctx, providers, ws, k8sClusterProvider.GetSeedClusterAdminClient(), k8sClusterProvider.SeedAdminConfig(), namespace, podName)
	}
}

func getWebTerminalNamespaceAndPodName(ctx context.Context, privilegedClusterProvider provider.PrivilegedClusterProvider, clusterName string) (string, string, error) {
	namespace := fmt.Sprintf("cluster-%s", clusterName)

	seedRuntimeClient := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()

	pods := &corev1.PodList{}
	err := seedRuntimeClient.List(ctx, pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabels{"app": "web-terminal"})
	if err != nil {
		return "", "", err
	}

	if len(pods.Items) == 0 {
		return "", "", fmt.Errorf("could not find the web terminal for the cluster %s", clusterName)
	}

	podName := pods.Items[0].Name

	return namespace, podName, nil
}

type terminalReq struct {
	ClusterID string
}

func (req terminalReq) GetSeedCluster() v1.SeedCluster {
	return v1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func verifyAuthorizationToken(req *http.Request, tokenVerifier auth.TokenVerifier, tokenExtractor auth.TokenExtractor) (*v1.User, error) {
	token, err := tokenExtractor.Extract(req)
	if err != nil {
		return nil, err
	}

	claims, err := tokenVerifier.Verify(req.Context(), token)
	if err != nil {
		return nil, err
	}

	if claims.Subject == "" {
		return nil, errors.NewNotAuthorized()
	}

	id, err := hash.GetUserID(claims.Subject)
	if err != nil {
		return nil, errors.NewNotAuthorized()
	}

	user := &v1.User{
		ObjectMeta: v1.ObjectMeta{
			ID:   id,
			Name: claims.Name,
		},
		Email: claims.Email,
	}

	if user.ID == "" {
		return nil, errors.NewNotAuthorized()
	}

	return user, nil
}

func requestLoggingReader(websocket *websocket.Conn) {
	defer func() {
		err := websocket.Close()
		if err != nil {
			log.Logger.Debug(err)
		}
	}()

	for {
		_, message, err := websocket.ReadMessage()
		if err != nil {
			log.Logger.Debug(err)
			break
		}

		log.Logger.Debug(message)
	}
}
