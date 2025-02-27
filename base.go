// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"net/http"

	"github.com/go-pogo/buildinfo"
	"github.com/go-pogo/easytls"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/healthcheck"
	"github.com/go-pogo/serv"
	"github.com/go-pogo/serv/accesslog"
	"github.com/go-pogo/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	ErrApplyOptions errors.Msg = "error while applying option(s)"
	ErrSetupServer  errors.Msg = "failed to setup server"
)

var _ healthcheck.HealthChecker = (*Base)(nil)

type Base struct {
	log    *Logger
	build  *buildinfo.BuildInfo
	telem  *telemetry.Telemetry
	health *healthcheck.Checker
	router *router
	server serv.Server
}

func New(name string, log *Logger, opts ...Option) (*Base, error) {
	conf := config{name: name}
	base := &Base{
		router: &router{
			ServeMux: serv.NewServeMux(),
			log:      log,
		},
	}

	// apply options
	var err error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		err = errors.Append(err, opt(base, &conf))
	}
	if err != nil {
		return nil, errors.Wrap(err, ErrApplyOptions)
	}

	// setup server
	if err = base.server.With(
		conf.server.Port,
		serv.WithName(conf.name),
		serv.WithLogger(log),
		serv.WithTLSConfig(easytls.DefaultTLSConfig(), conf.server.TLS),
	); err != nil {
		return nil, errors.Wrap(err, ErrSetupServer)
	}

	// wrap router
	var handler http.Handler = base.router
	if conf.server.AccessLog {
		handler = accesslog.Middleware(log, handler)
	}
	if base.telem != nil {
		handler = otelhttp.NewHandler(handler, name,
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
			otelhttp.WithMeterProvider(base.telem.MeterProvider()),
			otelhttp.WithTracerProvider(base.telem.TracerProvider()),
		)
	}

	base.server.Handler = handler
	return base, nil
}

func (base *Base) Logger() *Logger { return base.log }

func (base *Base) Telemetry() *telemetry.Telemetry { return base.telem }

func (base *Base) HealthChecker() *healthcheck.Checker { return base.health }

func (base *Base) RouteHandler() serv.RouteHandler { return base.router }

func (base *Base) Server() *serv.Server { return &base.server }

func (base *Base) CheckHealth(_ context.Context) healthcheck.Status {
	switch base.server.State() {
	case serv.StateUnstarted:
		return healthcheck.StatusUnknown
	case serv.StateStarted:
		return healthcheck.StatusHealthy
	default:
		return healthcheck.StatusUnhealthy
	}
}

func (base *Base) Run(ctx context.Context) error {
	if ctx != nil {
		base.server.BaseContext = serv.BaseContext(ctx)
	}
	return base.server.Run()
}

func (base *Base) Shutdown(ctx context.Context) error {
	// shutdown server before shutting down other services
	serverErr := base.server.Shutdown(ctx)
	telemErr := errors.Append(base.telem.ForceFlush(ctx), base.telem.Shutdown(ctx))

	return errors.Append(serverErr, telemErr)
}
