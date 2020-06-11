package websocket

import (
	"encoding/json"

	api "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
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
