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

package websocket

import (
	"encoding/json"

	v1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/watcher"

	"code.cloudfoundry.org/go-pubsub"
	"github.com/gorilla/websocket"
)

func WriteUser(providers watcher.Providers, ws *websocket.Conn, userEmail string) {
	initialUser, err := providers.UserProvider.UserByEmail(userEmail)
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	initialResponse, err := json.Marshal(initialUser)
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, initialResponse); err != nil {
		log.Logger.Debug(err)
		return
	}

	hashID, err := providers.UserWatcher.CalculateHash(userEmail)
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	providers.UserWatcher.Subscribe(func(rawUser interface{}) {
		var response []byte
		if rawUser != nil {
			user, ok := rawUser.(*v1.User)
			if !ok {
				log.Logger.Warn("cannot convert user for user watch: %v", rawUser)
				return
			}

			response, err = json.Marshal(user)
			if err != nil {
				log.Logger.Debug(err)
				return
			}
		} else {
			// Explicitly set null response instead returning defaulted user structure.
			// It allows clients to distinct null response and default or empty user.
			response, err = json.Marshal(nil)
			if err != nil {
				log.Logger.Debug(err)
				return
			}
		}

		if err := ws.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Logger.Debug(err)
			return
		}
	}, pubsub.WithPath([]uint64{hashID}))
}
