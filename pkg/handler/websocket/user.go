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
	"context"
	"encoding/json"

	"code.cloudfoundry.org/go-pubsub"
	"github.com/gorilla/websocket"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/watcher"
)

func WriteUser(ctx context.Context, providers watcher.Providers, ws *websocket.Conn, userEmail string) {
	// There can be a race here if the user changes between getting the initial data and setting up the subscription
	initialUser, err := providers.UserProvider.UserByEmail(ctx, userEmail)
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	userBindings, err := providers.MemberMapper.MappingsFor(ctx, initialUser.Spec.Email)
	if err != nil {
		log.Logger.Debug("cannot get project mappings for user %s: %v", initialUser.Name, err)
		return
	}
	groupBindings, err := providers.MemberMapper.GroupMappingsFor(ctx, initialUser.Spec.Groups)
	if err != nil {
		log.Logger.Debug("cannot get project mappings for groups %q+: %v", initialUser.Spec.Groups, err)
		return
	}
	initialExtUser := apiv1.ConvertInternalUserToExternal(initialUser, true, userBindings, groupBindings)

	initialResponse, err := json.Marshal(initialExtUser)
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

	unSub := providers.UserWatcher.Subscribe(func(rawUser interface{}) {
		var response []byte
		if rawUser != nil {
			user, ok := rawUser.(*kubermaticv1.User)
			if !ok {
				log.Logger.Warn("cannot convert user for user watch: %v", rawUser)
				return
			}

			userBindings, err := providers.MemberMapper.MappingsFor(ctx, user.Spec.Email)
			if err != nil {
				log.Logger.Debug("cannot get project mappings for user %s: %v", user.Name, err)
				return
			}

			groupBindings, err := providers.MemberMapper.GroupMappingsFor(ctx, initialUser.Spec.Groups)
			if err != nil {
				log.Logger.Debug("cannot get project mappings for groups %q+: %v", initialUser.Spec.Groups, err)
				return
			}

			externalUser := apiv1.ConvertInternalUserToExternal(user, true, userBindings, groupBindings)

			response, err = json.Marshal(externalUser)
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

	ws.SetCloseHandler(func(code int, text string) error {
		unSub()
		return writeCloseMessage(ws, code)
	})
}
