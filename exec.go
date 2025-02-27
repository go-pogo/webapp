// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"context"

	"github.com/go-pogo/errors"
	"github.com/go-pogo/webapp/waitgroup"
)

const (
	ErrDuringRun      errors.Msg = "an error occurred during run"
	ErrDuringShutdown errors.Msg = "an error occurred during shutdown"
)

func Run(ctx context.Context, targets ...func(ctx context.Context) error) error {
	wg := waitgroup.WithNotifyContext(ctx)
	for i := range targets {
		wg.Go(targets[i])
	}

	return errors.Wrap(wg.Wait(), ErrDuringRun)
}

// Shutdown calls all targets and blocks until all are called and have returned.
// Returned errors from these functions are collected and returned at the end.
func Shutdown(ctx context.Context, targets ...func(ctx context.Context) error) error {
	wg := waitgroup.WithContext(ctx)
	for i := range targets {
		wg.Go(targets[i])
	}
	return errors.Wrap(wg.Wait(), ErrDuringShutdown)
}
