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

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	wsh "k8c.io/kubermatic/v2/pkg/handler/websocket"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/util/hash"
	"k8c.io/kubermatic/v2/pkg/watcher"
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
type WebsocketTerminalWriter func(ctx context.Context, providers watcher.Providers, ws *websocket.Conn)

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	providers := getProviders(r)

	mux.HandleFunc("/ws/admin/settings", getSettingsWatchHandler(wsh.WriteSettings, providers, r))
	mux.HandleFunc("/ws/me", getUserWatchHandler(wsh.WriteUser, providers, r))
	mux.HandleFunc("/ws/projects/{project_id}/clusters/{cluster_id}/terminal", getTerminalWatchHandler(wsh.Terminal, providers, r))
}

func getProviders(r Routing) watcher.Providers {
	return watcher.Providers{
		SettingsProvider: r.settingsProvider,
		SettingsWatcher:  r.settingsWatcher,
		UserProvider:     r.userProvider,
		UserWatcher:      r.userWatcher,
		MemberMapper:     r.userProjectMapper,
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
		ctx := context.Background()
		user, err := verifyAuthorizationToken(req, routing.tokenVerifiers, routing.tokenExtractors)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		clusterID, err := common.DecodeClusterID(ctx, req)
		if err != nil {
			return
		}

		projectID, err := common.DecodeProjectRequest(ctx, req)
		if err != nil {
			return
		}

		fmt.Printf("user: %s, clusterID: %s, projectID %s", user.Email, clusterID, projectID)

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		writer(req.Context(), providers, ws)
		// commenting this because the websocket reader should be used to read the user messages (shell commands), not for logs
		//requestLoggingReader(ws)
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
