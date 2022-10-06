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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	wsh "k8c.io/kubermatic/v2/pkg/handler/websocket"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticcontext "k8c.io/kubermatic/v2/pkg/util/context"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/watcher"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
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
type WebsocketTerminalWriter func(ctx context.Context, ws *websocket.Conn, client ctrlruntimeclient.Client, k8sClient kubernetes.Interface, cfg *rest.Config, userEmailID string)

const (
	maxNumberOfTerminalActiveConnectionsPerUser = 5
	terminalActiveConnectionsMemoryDuration     = 24 * time.Hour
)

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	providers := getProviders(r)

	mux.HandleFunc("/ws/admin/settings", getSettingsWatchHandler(wsh.WriteSettings, providers, r))
	mux.HandleFunc("/ws/me", getUserWatchHandler(wsh.WriteUser, providers, r))
	mux.HandleFunc("/ws/projects/{project_id}/clusters/{cluster_id}/terminal", getTerminalWatchHandler(wsh.Terminal, providers, r, maxNumberOfTerminalActiveConnectionsPerUser, terminalActiveConnectionsMemoryDuration))
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

type connections struct {
	active map[string]int
	mutex  sync.Mutex
}

func newConnections() *connections {
	return &connections{
		active: make(map[string]int),
	}
}

func (l *connections) getActiveConnections(key string) int {
	l.mutex.Lock()
	activeConnections := l.active[key]
	l.mutex.Unlock()

	return activeConnections
}

func (l *connections) increaseActiveConnections(key string) {
	l.mutex.Lock()
	_, alreadyCreated := l.active[key]
	if !alreadyCreated {
		l.active[key] = 0
	}
	l.active[key]++
	l.mutex.Unlock()
}

func (l *connections) decreaseActiveConnections(key string) {
	l.mutex.Lock()
	_, alreadyCreated := l.active[key]
	if !alreadyCreated {
		l.active[key] = 0
	}
	if l.active[key] > 0 {
		l.active[key]--
	}
	l.mutex.Unlock()
}

func (l *connections) releaseMemory() {
	l.mutex.Lock()
	for key := range l.active {
		delete(l.active, key)
	}
	l.mutex.Unlock()
}

func getTerminalWatchHandler(writer WebsocketTerminalWriter, providers watcher.Providers, routing Routing, maxNumberOfConnections int, memoryDuration time.Duration) func(w http.ResponseWriter, req *http.Request) {
	connectionsPerUser := newConnections()

	// Cleaning the map from time to time to release the memory
	go func() {
		for {
			time.Sleep(memoryDuration)
			connectionsPerUser.releaseMemory()
		}
	}()

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

		cluster, err := handlercommon.GetCluster(ctx, providers.ProjectProvider, providers.PrivilegedProjectProvider, providers.UserInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return
		}

		userEmailID := wsh.EncodeUserEmailtoID(authenticatedUser.Email)
		k8sClient, err := clusterProvider.GetAdminK8sClientForUserCluster(ctx, cluster)
		if err != nil {
			return
		}
		cfg, err := clusterProvider.GetAdminClientConfigForUserCluster(ctx, cluster)
		if err != nil {
			return
		}
		client, err := clusterProvider.GetAdminClientForUserCluster(ctx, cluster)
		if err != nil {
			return
		}

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Logger.Debug(err)
			return
		}
		defer func(ws *websocket.Conn) {
			_ = ws.WriteJSON(wsh.TerminalMessage{Op: "msg", Data: "connection closed"})
			_ = ws.Close()
		}(ws)

		// Checking user active connections for project cluster
		userProjectClusterUniqueKey := fmt.Sprintf("%s-%s-%s", projectID, clusterID, authenticatedUser.Email)
		if connectionsPerUser.getActiveConnections(userProjectClusterUniqueKey) >= maxNumberOfConnections {
			err = errors.New("reached the maximum number of terminal active connections for the user")
			log.Logger.Debug(err)
			_ = ws.WriteJSON(wsh.TerminalMessage{
				Op:   "msg",
				Data: string(wsh.ConnectionPoolExceeded),
			})
			return
		}
		connectionsPerUser.increaseActiveConnections(userProjectClusterUniqueKey)
		defer connectionsPerUser.decreaseActiveConnections(userProjectClusterUniqueKey)

		if !wsh.WaitFor(5*time.Second, 2*time.Minute, func() bool {
			kubeconfigSecret := &corev1.Secret{}
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{
				Namespace: metav1.NamespaceSystem,
				Name:      handlercommon.KubeconfigSecretName(userEmailID),
			}, kubeconfigSecret); err != nil {
				log.Logger.Debug(err)
				_ = ws.WriteJSON(wsh.TerminalMessage{
					Op:   "msg",
					Data: string(wsh.KubeconfigSecretMissing),
				})
				return false
			}
			return true
		}) {
			return
		}

		writer(ctx, ws, client, k8sClient, cfg, userEmailID)
	}
}

type terminalReq struct {
	ClusterID string
}

func (req terminalReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func verifyAuthorizationToken(req *http.Request, tokenVerifier auth.TokenVerifier, tokenExtractor auth.TokenExtractor) (*apiv1.User, error) {
	token, err := tokenExtractor.Extract(req)
	if err != nil {
		return nil, err
	}

	claims, err := tokenVerifier.Verify(req.Context(), token)
	if err != nil {
		return nil, err
	}

	if claims.Subject == "" {
		return nil, utilerrors.NewNotAuthorized()
	}

	user := &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			Name: claims.Name,
		},
		Email: claims.Email,
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
