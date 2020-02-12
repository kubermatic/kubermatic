package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/watcher/common"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/admin"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//RegisterV1Admin declares all router paths for the admin users
func (r Routing) RegisterV1Admin(mux *mux.Router) {
	//
	// Defines a set of HTTP endpoints for the admin users
	mux.Methods(http.MethodGet).
		Path("/admin").
		Handler(r.getAdmins())

	mux.Methods(http.MethodPut).
		Path("/admin").
		Handler(r.setAdmin())

	mux.Methods(http.MethodGet).
		Path("/admin/settings").
		Handler(r.getKubermaticSettings())

	mux.Methods(http.MethodPatch).
		Path("/admin/settings").
		Handler(r.patchKubermaticSettings())

	mux.HandleFunc("/admin/settings/ws", r.getKubermaticSettingsWebsocket)
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

	go writer(ws, r.resourceWatcher, r.settingsProvider)
	reader(ws)
}

func writer(ws *websocket.Conn, watcher common.ResourceWatcher, provider provider.SettingsProvider) {
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

	watcher.SubscribeSettings(func(data interface{}) {
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

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	_ = ws.SetReadDeadline(time.Now().Add(time.Minute))
	ws.SetPongHandler(func(string) error {
		_ = ws.SetReadDeadline(time.Now().Add(time.Minute))
		return nil
	})
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println(p)
	}
}

// swagger:route GET /api/v1/admin/settings admin getKubermaticSettings
//
//     Gets the global settings.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GlobalSettings
//       401: empty
//       403: empty
func (r Routing) getKubermaticSettings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.KubermaticSettingsEndpoint(r.settingsProvider)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v1/admin/settings admin patchKubermaticSettings
//
//     Patches the global settings.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GlobalSettings
//       401: empty
//       403: empty
func (r Routing) patchKubermaticSettings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateKubermaticSettingsEndpoint(r.userInfoGetter, r.settingsProvider)),
		admin.DecodePatchKubermaticSettingsReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin admin getAdmins
//
//     Returns list of admin users.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Admin
//       401: empty
//       403: empty
func (r Routing) getAdmins() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.GetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin admin setAdmin
//
//     Allows setting and clearing admin role for users.
//
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Admin
//       401: empty
//       403: empty
func (r Routing) setAdmin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.SetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		admin.DecodeSetAdminReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
