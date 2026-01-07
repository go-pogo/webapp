// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

import (
	"context"
	"io"
	"net/http"
	"os"
	"runtime"
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
	LogBuildInfo(bld *buildinfo.BuildInfo)
	LogBuildInfoModules(bld *buildinfo.BuildInfo, modules ...string)
}

type RegisterRouteLogger interface {
	LogRegisterRoute(route serv.Route)
}

type OTELLoggerSetter interface {
	SetOTELLogger()
}

type Config struct {
	Level         zerolog.Level `env:"LOG_LEVEL" default:"warn" description:"Valid levels are: debug, info, warn, error, fatal, panic"`
	WithTimestamp bool          `env:"LOG_TIMESTAMP" default:"true" description:"Starts the log's line with a timestamp when true"`
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

// Logger wraps a [zerolog.Logger] and implements several log interfaces.
type Logger struct{ zerolog.Logger }

// NewProductionLogger returns a production ready [Logger].
func NewProductionLogger(conf Config) *Logger {
	return newLogger(os.Stdout, conf)
}

// NewDevelopmentLogger returns a [Logger] configured for development
// environments.
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

// LogBuildInfo is part of the [BuildInfoLogger] interface.
func (l *Logger) LogBuildInfo(bld *buildinfo.BuildInfo) {
	if bld == nil {
		return
	}

	log := l.Info()
	if bld.Version != "" {
		log.Str("version", bld.Version)
	}
	if bld.Revision != "" {
		log.Str("vcs_revision", bld.Revision)
	}
	if !bld.Time.IsZero() {
		log.Time("vcs_time", bld.Time)
	}

	if bld.GoVersion != "" {
		log.Str("go_version", bld.GoVersion)
	} else {
		log.Str("go_version", runtime.Version())
	}
	log.Msg("buildinfo")
}

// LogBuildInfoModules is part of the [BuildInfoLogger] interface.
func (l *Logger) LogBuildInfoModules(bld *buildinfo.BuildInfo, modules ...string) {
	if bld == nil {
		return
	}

	for _, name := range modules {
		if mod := bld.Module(name); mod != nil {
			l.Info().
				Str("path", mod.Path).
				Str("version", mod.Version).
				Str("checksum", mod.Sum).
				Msg("module")
		}
	}
}

// LogRegisterRoute is part of the [RegisterRouteLogger] interface.
func (l *Logger) LogRegisterRoute(route serv.Route) {
	l.Debug().
		Str("name", route.Name).
		Str("method", route.Method).
		Str("pattern", route.Pattern).
		Msg("register route")
}

// LogServerStart is part of the [serv.Logger] interface.
func (l *Logger) LogServerStart(name, addr string) {
	l.Info().
		Str("name", name).
		Str("addr", addr).
		Msg("server starting")
}

// LogServerStartTLS is part of the [serv.Logger] interface.
func (l *Logger) LogServerStartTLS(name, addr, certFile, keyFile string) {
	l.Info().
		Str("name", name).
		Str("addr", addr).
		Str("cert_file", certFile).
		Str("key_file", keyFile).
		Msg("server starting")
}

// LogServerShutdown is part of the [serv.Logger] interface.
func (l *Logger) LogServerShutdown(name string) {
	l.Info().
		Str("name", name).
		Msg("server shutting down")
}

// LogServerClose is part of the [serv.Logger] interface.
func (l *Logger) LogServerClose(name string) {
	l.Info().
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

	event := l.WithLevel(lvl).
		Str("server", det.ServerName).
		Str("handler", det.HandlerName)

	if det.RequestID != "" {
		event.Str("request_id", det.RequestID)
	}

	event.Str("user_agent", det.UserAgent).
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
	l.Info().
		Stringer("status", status).
		Stringer("old_status", oldStatus).
		Msg("health changed")

	for name, stat := range details {
		l.Debug().
			Str("name", name).
			Stringer("status", stat).
			Msg("health")
	}
}

// LogHealthChecked is part of the [healthclient.Logger] interface.
func (l *Logger) LogHealthChecked(stat healthcheck.Status) {
	l.Info().
		Stringer("status", stat).
		Msg("health checked")
}

// LogHealthCheckFailed is part of the [healthclient.Logger] interface.
func (l *Logger) LogHealthCheckFailed(stat healthcheck.Status, err error) {
	l.Err(err).
		Stringer("status", stat).
		Msg("health check failed")
}

// SetOTELLogger is part of the [OTELLoggerSetter] interface.
func (l *Logger) SetOTELLogger() {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		l.Err(err).Msg("otel error")
	}))

	zl := l.Level(zerolog.DebugLevel)
	otel.SetLogger(zerologr.New(&zl))
}
