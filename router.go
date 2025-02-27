// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"net/http"

	"github.com/go-pogo/serv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type registerRouteLogger interface {
	LogRegisterRoute(route serv.Route)
}

var _ serv.Router = (*router)(nil)

type router struct {
	*serv.ServeMux
	log registerRouteLogger
}

func (mux *router) Handle(pattern string, handler http.Handler) {
	mux.HandleRoute(serv.Route{
		Pattern: pattern,
		Handler: handler,
	})
}

func (mux *router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleRoute(serv.Route{
		Pattern: pattern,
		Handler: http.HandlerFunc(handler),
	})
}

func (mux *router) HandleRoute(route serv.Route) {
	if mux.log != nil {
		mux.log.LogRegisterRoute(route)
	}

	route.Handler = otelhttp.WithRouteTag(route.Pattern, route.Handler)
	mux.ServeMux.HandleRoute(route)
}
