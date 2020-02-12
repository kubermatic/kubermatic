package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/watcher"

	"github.com/gorilla/websocket"
)

func WriteSettings(ws *websocket.Conn, watcher watcher.SettingsWatcher, provider provider.SettingsProvider) {
	initialSettings, err := provider.GetGlobalSettings()
	if err != nil {
		fmt.Println(err)
		return
	}

	initialResponse, err := json.Marshal(initialSettings)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, initialResponse); err != nil {
		fmt.Println(err)
		return
	}

	watcher.Subscribe(func(settings interface{}) {
		fmt.Println(settings) // TODO: Check "nil" case.

		response, err := json.Marshal(settings)
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := ws.WriteMessage(websocket.TextMessage, response); err != nil {
			fmt.Println(err)
			return
		}
	})
}
