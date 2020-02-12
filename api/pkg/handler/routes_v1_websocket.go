package handler

import (
	"context"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	wsh "github.com/kubermatic/kubermatic/api/pkg/handler/websocket"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/watcher"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketWriter func(providers watcher.Providers, ws *websocket.Conn)

func (r Routing) RegisterV1Websocket(mux *mux.Router) {
	providers := getProviders(r)

	mux.HandleFunc("/ws/admin/settings/", getHandler(wsh.WriteSettings, providers, r))
}

func getProviders(r Routing) watcher.Providers {
	return watcher.Providers{
		SettingsProvider: r.settingsProvider,
		SettingsWatcher:  r.settingsWatcher,
	}
}

func getHandler(writer WebsocketWriter, providers watcher.Providers, routing Routing) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		err := verifyAuthorizationToken(req, routing.tokenVerifiers)
		if err != nil {
			log.Logger.Debug(err)
			return
		}

		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Logger.Debug(err)
			}
			return
		}

		go writer(providers, ws)
		requestLoggingReader(ws)
	}
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

func requestLoggingReader(websocket *websocket.Conn) {
	defer func() {
		err := websocket.Close()
		if err != nil {
			log.Logger.Debug(err)
		}
	}()

	websocket.SetReadLimit(512)

	for {
		_, message, err := websocket.ReadMessage()
		if err != nil {
			break
		}

		log.Logger.Debug(message)
	}
}
