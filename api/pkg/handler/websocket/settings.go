package websocket

import (
	"encoding/json"

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

	initialResponse, err := json.Marshal(initialSettings)
	if err != nil {
		log.Logger.Debug(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, initialResponse); err != nil {
		log.Logger.Debug(err)
		return
	}

	providers.SettingsWatcher.Subscribe(func(settings interface{}) {
		response, err := json.Marshal(settings)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		if err := ws.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Logger.Debug(err)
			return
		}
	})
}
