// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"net/http"

	"github.com/go-pogo/serv"
	"github.com/go-pogo/webapp/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

var _ serv.Router = (*router)(nil)

type router struct {
	*serv.ServeMux

	log   logger.RegisterRouteLogger
	trace bool
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
	if mux.trace {
		attr := semconv.HTTPRoute(route.Name)
		handler := route.Handler

		route.Handler = http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
			span := trace.SpanFromContext(req.Context())
			span.SetAttributes(attr)

			labeler, _ := otelhttp.LabelerFromContext(req.Context())
			labeler.Add(attr)
			handler.ServeHTTP(wri, req)
		})
	}
	mux.ServeMux.HandleRoute(route)
}
