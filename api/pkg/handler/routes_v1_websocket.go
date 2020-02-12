package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/watcher"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	mux.HandleFunc("/ws/admin/settings/", r.getKubermaticSettingsWebsocket)
}

func (r Routing) getKubermaticSettingsWebsocket(w http.ResponseWriter, req *http.Request) {
	err := verifyAuthorizationToken(req, r.tokenVerifiers)
	if err != nil {
		r.log.Error(err)
		return
	}

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			r.log.Error(err)
		}
		return
	}

	go writer(ws, r.settingsWatcher, r.settingsProvider)
	requestLoggingReader(ws, r.log)
}

func writer(ws *websocket.Conn, watcher watcher.SettingsWatcher, provider provider.SettingsProvider) {
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

func verifyAuthorizationToken(req *http.Request, tokenVerifier auth.TokenVerifier) error {
	tokenExtractor := auth.NewHeaderBearerTokenExtractor("Authorization")
	token, err := tokenExtractor.Extract(req)
	if err != nil {
		return err
	}

	_, err = tokenVerifier.Verify(context.TODO(), token)
	return err
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
