package handler

import (
	"encoding/json"
	"fmt"
	v12 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/watch"
	"net/http"
	"reflect"
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

// TODO: Auth (middleware) is missing!
func (r Routing) getKubermaticSettingsWebsocket(w http.ResponseWriter, req *http.Request) {
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			kubermaticlog.Logger.Error(err) // TODO: How to log errors like this?
		}
		return
	}

	go writer(ws, r.settingsProvider)
	reader(ws)
}

func writer(ws *websocket.Conn, pr provider.SettingsProvider) {
	gs, err := pr.GetGlobalSettings() // Initial query.
	if err != nil {
		ws.Close()
		return
	}

	js, err := json.Marshal(gs)
	if err != nil {
		ws.Close()
		return
	}

	if err = ws.WriteMessage(websocket.TextMessage, js); err != nil {
		return
	}

	watcher, err := pr.WatchGlobalSettings()

	go func() {
		fmt.Println("starting watch")
		defer watcher.Stop()
		//defer close(self.errChan)
		for {
			select {
			case ev, ok := <-watcher.ResultChan():
				if !ok {
					fmt.Println("watch ended with timeout")
					return
				}

				if err = ws.WriteMessage(websocket.TextMessage, handleEvent(ev)); err != nil {
					return
				}
			}
		}
	}()
}

func handleEvent(event watch.Event) []byte {
	var obj *v12.KubermaticSetting

	fmt.Println("handleevent")
	fmt.Println(event.Type)

	switch event.Type {
	case watch.Added:
		secret, ok := event.Object.(*v12.KubermaticSetting)
		if !ok {
			fmt.Printf("expected settings got %s", reflect.TypeOf(event.Object))
		} else {
			obj = secret
		}
	case watch.Modified:
		secret, ok := event.Object.(*v12.KubermaticSetting)
		if !ok {
			fmt.Printf("expected settings got %s", reflect.TypeOf(event.Object))
		} else {
			obj = secret
		}
	case watch.Deleted:
		obj = nil
	case watch.Error:
		fmt.Println("error")
	}

	var res []byte
	if obj != nil {
		js, err := json.Marshal(obj)
		if err != nil {
			res = js
		}
	}

	return res
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
