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

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	v1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/watcher"

	"code.cloudfoundry.org/go-pubsub"
	"github.com/gorilla/websocket"
)

func WriteUser(providers watcher.Providers, ws *websocket.Conn, userEmail string) {
	initialUser, err := providers.UserProvider.UserByEmail(userEmail)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	bindings, err := providers.MemberMapper.MappingsFor(initialUser.Spec.Email)
	if err != nil {
		log.Logger.Error("cannot get project mappings for user %s: %v", initialUser.Name, err)
		return
	}
	initialExtUser := apiv1.ConvertInternalUserToExternal(initialUser, true, bindings...)

	initialResponse, err := json.Marshal(initialExtUser)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, initialResponse); err != nil {
		log.Logger.Error(err)
		return
	}

	hashID, err := providers.UserWatcher.CalculateHash(userEmail)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	providers.UserWatcher.Subscribe(func(rawUser interface{}) {
		var response []byte
		if rawUser != nil {
			user, ok := rawUser.(*v1.User)
			if !ok {
				log.Logger.Error("cannot convert user for user watch: %v", rawUser)
				return
			}
			log.Logger.Errorf("Subscription event: Got a new subscription event for user: %s", user.Name)

			bindings, err := providers.MemberMapper.MappingsFor(user.Spec.Email)
			if err != nil {
				log.Logger.Error("cannot get project mappings for user %s: %v", user.Name, err)
				return
			}
			externalUser := apiv1.ConvertInternalUserToExternal(user, true, bindings...)

			response, err = json.Marshal(externalUser)
			if err != nil {
				log.Logger.Error(err)
				return
			}
		} else {
			log.Logger.Error("Subscription event: got a null event")
			// Explicitly set null response instead returning defaulted user structure.
			// It allows clients to distinct null response and default or empty user.
			response, err = json.Marshal(nil)
			if err != nil {
				log.Logger.Error(err)
				return
			}
		}

		if err := ws.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Logger.Error(err)
			return
		}
	}, pubsub.WithPath([]uint64{hashID}))
}
