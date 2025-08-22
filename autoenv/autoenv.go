// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autoenv

import (
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-pogo/env"
	"github.com/go-pogo/env/dotenv"
	"github.com/go-pogo/errors"
)

// SkipCaller skips the amount of callers when determining the absolute path of
// the main.go file. When this value is negative, it is subtracted from the
// amount of captured caller frames. Otherwise, the absolute value is used.
var SkipCaller = -4

// CaptureCallers is the max amount of caller frames that should be captured
// using [runtime.Callers] to determine the absolute path of the main.go file.
var CaptureCallers uint8 = 16

var loadOnce = sync.OnceValue(func() error {
	return NewLoader().Load()
})

// Unmarshal loads environment variables once using [NewLoader] and then decodes
// v using a [env.NewDecoder].
func Unmarshal(v any) error {
	if err := loadOnce(); err != nil {
		return err
	}
	return env.NewDecoder(env.System()).Decode(v)
}

type Loader struct {
	Dir string

	env dotenv.ActiveEnvironment
}

// NewProductionLoader returns a new [Loader] configured for production
// environments.
func NewProductionLoader() *Loader {
	return &Loader{Dir: "/"}
}

// NewDevelopmentLoader returns a new [Loader] configured for development
// environments.
func NewDevelopmentLoader(skipCaller int, args ...string) *Loader {
	ae, _ := dotenv.GetActiveEnvironmentOr(args, dotenv.Development)
	_, file, _, _ := runtime.Caller(skipCaller + 1)

	return &Loader{
		Dir: filepath.Dir(file),
		env: ae,
	}
}

func (l *Loader) PrefixDir(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.Dir, path)
}

func (l *Loader) Load() error {
	environ, err := dotenv.Read(l.Dir, l.env).Environ()
	if err != nil {
		var noFilesLoaded *dotenv.NoFilesLoadedError
		if errors.As(err, &noFilesLoaded) {
			return nil
		}
		return err
	}
	if err = env.Load(environ); err != nil {
		return err
	}
	return nil
}
