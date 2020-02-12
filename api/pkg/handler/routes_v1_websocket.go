package handler

import (
	"context"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	wsh "github.com/kubermatic/kubermatic/api/pkg/handler/websocket"
	"github.com/kubermatic/kubermatic/api/pkg/log"

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
	err := verifyAuthorizationToken(req, r.tokenVerifiers)
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

	go wsh.WriteSettings(ws, r.settingsWatcher, r.settingsProvider)
	requestLoggingReader(ws)
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
