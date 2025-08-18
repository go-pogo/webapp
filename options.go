// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"net/http"

	"github.com/go-pogo/buildinfo"
	"github.com/go-pogo/easytls"
	"github.com/go-pogo/healthcheck"
	"github.com/go-pogo/serv"
	"github.com/go-pogo/serv/accesslog"
	"github.com/go-pogo/telemetry"
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
	name     string
	server   ServerConfig
	servOpts []serv.Option
	logger   Logger
}

func WithLogger(log Logger) Option {
	return func(base *Base, config *config) error {
		base.router.log = log
		config.logger = log
		return nil
	}
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
			if config.logger != nil {
				config.logger.LogBuildInfo(bld, modules...)
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
		if config.logger != nil {
			config.logger.SetOTELLogger()
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

func WithServerOption(opts ...serv.Option) Option {
	return func(_ *Base, config *config) error {
		if config.servOpts == nil {
			config.servOpts = make([]serv.Option, 0, len(opts))
		}
		config.servOpts = append(config.servOpts, opts...)
		return nil
	}
}

func WithHealthChecker(opts ...healthcheck.Option) Option {
	return func(base *Base, config *config) error {
		var err error
		if config.logger != nil {
			pre := []healthcheck.Option{healthcheck.WithLogger(config.logger)}
			opts = append(pre, opts...)
		}

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

func (c config) servLogger() serv.Logger {
	if c.logger == nil {
		return serv.NopLogger()
	}
	return c.logger
}

func (c config) accessLogger() accesslog.Logger {
	if c.logger == nil {
		return accesslog.NopLogger()
	}
	return c.logger
}
