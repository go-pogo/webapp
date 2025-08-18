// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autoenv

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-pogo/env"
	"github.com/go-pogo/env/dotenv"
	"github.com/go-pogo/env/envfile"
	"github.com/go-pogo/errors"
)

const ErrLoadEnvFile = "failed to load .env file"

// Load environment variables
func Load() error { return newLoader().Load() }

var loadOnce = sync.OnceValue(Load)

func Unmarshal(v any) error {
	if err := loadOnce(); err != nil {
		return err
	}
	return env.NewDecoder(env.System()).Decode(v)
}

type Loader interface {
	Load() error
}

var _ Loader = (*ProductionLoader)(nil)

type ProductionLoader struct {
	Dir string
}

func NewProductionLoader() *ProductionLoader {
	return &ProductionLoader{Dir: "/"}
}

func (pl *ProductionLoader) Load() error {
	if err := envfile.Load(filepath.Join(pl.Dir, ".env")); err != nil {
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) {
			return errors.Wrap(err, ErrLoadEnvFile)
		}
	}
	return nil
}

var _ Loader = (*DevelopmentLoader)(nil)

type DevelopmentLoader struct {
	Dir string
	Env dotenv.ActiveEnvironment
}

func NewDevelopmentLoader(skipCaller int, args ...string) *DevelopmentLoader {
	ae, _ := dotenv.GetActiveEnvironmentOr(args, dotenv.Development)
	_, file, _, _ := runtime.Caller(skipCaller + 1)

	return &DevelopmentLoader{
		Env: ae,
		Dir: filepath.Dir(file),
	}
}

func (dl *DevelopmentLoader) PrefixDir(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dl.Dir, path)
}

func (dl *DevelopmentLoader) Load() error {
	environ, err := dotenv.Read(dl.Dir, dl.Env).Environ()
	if err != nil {
		return err
	}

	if err = env.Load(environ); err != nil {
		return err
	}
	return nil
}
