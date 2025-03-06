// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"time"

	"github.com/go-pogo/errors"
	"github.com/go-pogo/errors/errgroup"
	"github.com/go-pogo/webapp/rungroup"
)

const (
	ErrDuringRun      errors.Msg = "an error occurred during run"
	ErrDuringShutdown errors.Msg = "an error occurred during shutdown"
)

func Run(ctx context.Context, targets ...func(ctx context.Context) error) error {
	grp := rungroup.New(ctx)
	for i := range targets {
		grp.Go(targets[i])
	}

	return errors.Wrap(grp.Wait(), ErrDuringRun)
}

// Shutdown calls all targets and blocks until all are called and have returned.
// Returned errors from these functions are collected and returned at the end.
func Shutdown(ctx context.Context, targets ...func(ctx context.Context) error) error {
	var grp errgroup.Group
	for i := range targets {
		grp.Go(func() error {
			return targets[i](ctx)
		})
	}
	return errors.Wrap(grp.Wait(), ErrDuringShutdown)
}

// ShutdownTimeout calls all targets and blocks until all are called and have
// returned, or when the timeout elapses. Returned errors from these functions
// are collected and returned at the end.
func ShutdownTimeout(ctx context.Context, timeout time.Duration, targets ...func(ctx context.Context) error) error {
	ctx, cancelFn := context.WithTimeout(ctx, timeout)
	defer cancelFn()
	return Shutdown(ctx, targets...)
}
