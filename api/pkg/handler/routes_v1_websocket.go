package handler

import (
	"context"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	wsh "github.com/kubermatic/kubermatic/api/pkg/handler/websocket"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/util/hash"
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

	mux.HandleFunc("/ws/admin/settings", getHandler(wsh.WriteSettings, providers, r))
}

func getProviders(r Routing) watcher.Providers {
	return watcher.Providers{
		SettingsProvider: r.settingsProvider,
		SettingsWatcher:  r.settingsWatcher,
	}
}

func getHandler(writer WebsocketWriter, providers watcher.Providers, routing Routing) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := verifyAuthorizationToken(req, routing.tokenVerifiers)
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

func verifyAuthorizationToken(req *http.Request, tokenVerifier auth.TokenVerifier) (*v1.User, error) {
	tokenExtractor := auth.NewHeaderBearerTokenExtractor("Authorization")
	token, err := tokenExtractor.Extract(req)
	if err != nil {
		return nil, err
	}

	claims, err := tokenVerifier.Verify(context.TODO(), token)
	if err != nil {
		return nil, err
	}

	if claims.Subject == "" {
		return nil, errors.NewNotAuthorized()
	}

	id, err := hash.GetUserID(claims.Subject)
	if err != nil {
		return nil, errors.NewNotAuthorized()
	}

	user := &v1.User{
		ObjectMeta: v1.ObjectMeta{
			ID:   id,
			Name: claims.Name,
		},
		Email: claims.Email,
	}

	if user.ID == "" {
		return nil, errors.NewNotAuthorized()
	}

	return user, nil
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
