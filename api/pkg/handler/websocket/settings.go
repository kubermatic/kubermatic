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

	api "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/watcher"

	"github.com/gorilla/websocket"
)

func WriteSettings(providers watcher.Providers, ws *websocket.Conn) {
	initialSettings, err := providers.SettingsProvider.GetGlobalSettings()
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	initialResponse, err := json.Marshal(api.GlobalSettings(initialSettings.Spec))
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, initialResponse); err != nil {
		log.Logger.Debug(err)
		return
	}

	providers.SettingsWatcher.Subscribe(func(settings interface{}) {
		var response []byte
		if settings != nil {
			var externalSettings api.GlobalSettings
			internalSettings, ok := settings.(*v1.KubermaticSetting)
			if ok {
				externalSettings = api.GlobalSettings(internalSettings.Spec)
			} else {
				log.Logger.Debug("cannot convert settings: %v", settings)
			}

			response, err = json.Marshal(externalSettings)
			if err != nil {
				log.Logger.Debug(err)
				return
			}
		} else {
			// Explicitly set null response instead returning defaulted global settings structure.
			// It allows clients to distinct null response and default or empty global settings structure.
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
	})
}
