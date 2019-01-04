package handler

import (
	"context"
	"net/http"
)

func decodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}
