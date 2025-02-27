// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"net/http"

	"github.com/go-logr/zerologr"
	"github.com/go-pogo/buildinfo"
	"github.com/go-pogo/easytls"
	"github.com/go-pogo/healthcheck"
	"github.com/go-pogo/serv"
	"github.com/go-pogo/serv/accesslog"
	"github.com/go-pogo/telemetry"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

const (
	BuildInfoRoute   = buildinfo.MetricName
	HealthCheckRoute = "healthcheck"
	FaviconRoute     = "favicon"
)

type ServerConfig struct {
	Port      serv.Port `default:"8080"`
	AccessLog bool      `default:"true"`
	TLS       easytls.Config
}

type Option func(base *Base, config *config) error

type config struct {
	name   string
	server ServerConfig
}

func WithBuildInfo(bld *buildinfo.BuildInfo) Option {
	if bld == nil {
		return nil
	}

	return func(base *Base, config *config) error {
		base.build = bld
		base.router.HandleRoute(serv.Route{
			Name:    BuildInfoRoute,
			Method:  http.MethodGet,
			Pattern: buildinfo.PathPattern,
			Handler: buildinfo.HTTPHandler(bld),
		})
		return nil
	}
}

func WithBuildInfoVersion(altVersion string, modules ...string) Option {
	return func(base *Base, config *config) error {
		bld, err := buildinfo.New(altVersion)
		if err != nil {
			return err
		}

		if optFn := WithBuildInfo(bld); optFn != nil {
			if base.log != nil {
				base.log.LogBuildInfo(bld, modules...)
			}
			return optFn(base, config)
		}
		return nil
	}
}

func WithTelemetryConfig(conf telemetry.Config) Option {
	return func(base *Base, config *config) error {
		builder := telemetry.NewBuilder(conf).Global().WithDefaultExporter()
		if base.build != nil {
			builder.TracerProvider.WithAttributes(semconv.ServiceVersion(base.build.Version()))
			builder.TracerProvider.WithBuildInfo(base.build.Internal())
		}

		var err error
		base.telem, err = builder.Build()
		if err != nil {
			return err
		}

		if base.log != nil {
			otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
				base.log.Logger.Err(err).Msg("otel error")
			}))

			zl := base.log.Logger.Level(zerolog.DebugLevel)
			otel.SetLogger(zerologr.New(&zl))
		}
		return nil
	}
}

func WithServerConfig(conf ServerConfig) Option {
	return func(_ *Base, config *config) error {
		config.server = conf
		return nil
	}
}

func WithHealthChecker(opts ...healthcheck.Option) Option {
	return func(base *Base, config *config) error {
		var err error
		base.health, err = healthcheck.New(opts...)
		if err != nil {
			return err
		}

		base.health.Register(config.name, base)
		base.router.HandleRoute(serv.Route{
			Name:    HealthCheckRoute,
			Method:  http.MethodGet,
			Pattern: healthcheck.PathPattern,
			Handler: healthcheck.HTTPHandler(base.health),
		})
		return nil
	}
}

func WithRoutesRegisterer(rr serv.RoutesRegisterer) Option {
	return func(base *Base, _ *config) error {
		rr.RegisterRoutes(base.router)
		return nil
	}
}

func WithNotFoundHandler(h http.Handler) Option {
	return func(base *Base, _ *config) error {
		base.router.WithNotFoundHandler(h)
		return nil
	}
}

func WithIgnoreFaviconRoute() Option {
	return func(base *Base, _ *config) error {
		base.router.HandleRoute(serv.Route{
			Name:    FaviconRoute,
			Method:  http.MethodGet,
			Pattern: "/favicon.ico",
			Handler: accesslog.IgnoreHandler(serv.NoContentHandler()),
		})
		return nil
	}
}
