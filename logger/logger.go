// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/go-logr/zerologr"
	"github.com/go-pogo/buildinfo"
	"github.com/go-pogo/healthcheck"
	"github.com/go-pogo/healthcheck/healthclient"
	"github.com/go-pogo/serv"
	"github.com/go-pogo/serv/accesslog"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

type BuildInfoLogger interface {
	LogBuildInfo(bld *buildinfo.BuildInfo, modules ...string)
}

type RegisterRouteLogger interface {
	LogRegisterRoute(route serv.Route)
}

type OTELLoggerSetter interface {
	SetOTELLogger()
}

type Config struct {
	Level         zerolog.Level `env:"LOG_LEVEL" default:"debug"`
	WithTimestamp bool          `env:"LOG_TIMESTAMP" default:"true"`
}

var (
	_ BuildInfoLogger     = (*Logger)(nil)
	_ RegisterRouteLogger = (*Logger)(nil)
	_ OTELLoggerSetter    = (*Logger)(nil)

	_ serv.Logger         = (*Logger)(nil)
	_ accesslog.Logger    = (*Logger)(nil)
	_ healthcheck.Logger  = (*Logger)(nil)
	_ healthclient.Logger = (*Logger)(nil)
)

type Logger struct{ zerolog.Logger }

func NewProductionLogger(conf Config) *Logger {
	return newLogger(os.Stdout, conf)
}

func NewDevelopmentLogger(conf Config) *Logger {
	out := zerolog.NewConsoleWriter()
	out.TimeFormat = time.StampMilli
	return newLogger(out, conf)
}

func newLogger(out io.Writer, conf Config) *Logger {
	log := zerolog.New(out).Level(conf.Level)
	if conf.WithTimestamp {
		log = log.With().Timestamp().Logger()
	}
	return &Logger{log}
}

func (l *Logger) LogBuildInfo(bld *buildinfo.BuildInfo, modules ...string) {
	event := l.Logger.Info().
		Str("go_version", bld.GoVersion()).
		Str("version", bld.Version()).
		Str("vcs_revision", bld.Revision()).
		Time("vcs_time", bld.Time())

	for _, name := range modules {
		if mod := bld.Module(name); mod.Version != "" {
			event.Str("module_"+path.Base(mod.Path), mod.Version)
		}
	}

	event.Msg("buildinfo")
}

func (l *Logger) LogRegisterRoute(route serv.Route) {
	l.Logger.Debug().
		Str("name", route.Name).
		Str("method", route.Method).
		Str("pattern", route.Pattern).
		Msg("register route")
}

// LogServerStart is part of the [serv.Logger] interface.
func (l *Logger) LogServerStart(name, addr string) {
	l.Logger.Info().
		Str("name", name).
		Str("addr", addr).
		Msg("server starting")
}

// LogServerStartTLS is part of the [serv.Logger] interface.
func (l *Logger) LogServerStartTLS(name, addr, certFile, keyFile string) {
	l.Logger.Info().
		Str("name", name).
		Str("addr", addr).
		Str("cert_file", certFile).
		Str("key_file", keyFile).
		Msg("server starting")
}

// LogServerShutdown is part of the [serv.Logger] interface.
func (l *Logger) LogServerShutdown(name string) {
	l.Logger.Info().
		Str("name", name).
		Msg("server shutting down")
}

// LogServerClose is part of the [serv.Logger] interface.
func (l *Logger) LogServerClose(name string) {
	l.Logger.Info().
		Str("name", name).
		Msg("server closing")
}

// LogAccess is part of the [accesslog.Logger] interface. Default log level is
// [zerolog.InfoLevel]. Every status code indicating an error is logged as
// [zerolog.WarnLevel]. All remaining requests to the [HealthCheckRoute] are
// logged as [zerolog.DebugLevel]
func (l *Logger) LogAccess(_ context.Context, det accesslog.Details, req *http.Request) {
	lvl := zerolog.InfoLevel
	if det.StatusCode >= 400 {
		lvl = zerolog.WarnLevel
	} else if det.HandlerName == "healthcheck" {
		lvl = zerolog.DebugLevel
	}

	l.Logger.WithLevel(lvl).
		Str("server", det.ServerName).
		Str("handler", det.HandlerName).
		Str("user_agent", det.UserAgent).
		Str("remote_addr", accesslog.RemoteAddr(req)).
		Str("method", req.Method).
		Str("request_uri", accesslog.RequestURI(req)).
		Int("status_code", det.StatusCode).
		Int64("request_count", det.RequestCount).
		Int64("bytes_written", det.BytesWritten).
		Dur("duration", det.Duration).
		Msg(accesslog.Message)
}

// LogHealthChanged is part of the [healthcheck.Logger] interface.
func (l *Logger) LogHealthChanged(status, oldStatus healthcheck.Status, details map[string]healthcheck.Status) {
	l.Logger.Info().
		Stringer("status", status).
		Stringer("old_status", oldStatus).
		Msg("health changed")

	for name, stat := range details {
		l.Logger.Debug().
			Str("name", name).
			Stringer("status", stat).
			Msg("health")
	}
}

// LogHealthChecked is part of the [healthclient.Logger] interface.
func (l *Logger) LogHealthChecked(stat healthcheck.Status) {
	l.Logger.Info().
		Stringer("status", stat).
		Msg("health checked")
}

// LogHealthCheckFailed is part of the [healthclient.Logger] interface.
func (l *Logger) LogHealthCheckFailed(stat healthcheck.Status, err error) {
	l.Logger.Err(err).
		Stringer("status", stat).
		Msg("health check failed")
}

func (l *Logger) SetOTELLogger() {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		l.Logger.Err(err).Msg("otel error")
	}))

	zl := l.Logger.Level(zerolog.DebugLevel)
	otel.SetLogger(zerologr.New(&zl))
}
