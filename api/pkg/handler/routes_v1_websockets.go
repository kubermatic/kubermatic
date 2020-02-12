package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/watcher/common"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	mux.HandleFunc("/ws/admin/settings/", r.getKubermaticSettingsWebsocket)
}

func (r Routing) getKubermaticSettingsWebsocket(w http.ResponseWriter, req *http.Request) {
	x := auth.NewHeaderBearerTokenExtractor("Authorization")
	token, err := x.Extract(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = r.tokenVerifiers.Verify(context.TODO(), token)
	if err != nil {
		fmt.Println(err)
		return
	}

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			kubermaticlog.Logger.Error(err)
		}
		return
	}

	go writer(ws, r.settingsWatcher, r.settingsProvider)
	requestLoggingReader(ws, r.log)
}

func writer(ws *websocket.Conn, watcher common.SettingsWatcher, provider provider.SettingsProvider) {
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

	watcher.Subscribe(func(data interface{}) {
		fmt.Println(data)

		//var res []byte
		//if obj != nil {
		//	js, err := json.Marshal(obj)
		//	if err != nil {
		//		res = js
		//	}
		//}

		//if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		//	return
		//}

	})
}

func requestLoggingReader(websocket *websocket.Conn, logger *zap.SugaredLogger) {
	defer func() {
		err := websocket.Close()
		if err != nil {
			logger.Error(err)
		}
	}()

	websocket.SetReadLimit(512)

	for {
		_, message, err := websocket.ReadMessage()
		if err != nil {
			break
		}

		logger.Debug(message)
	}
}
