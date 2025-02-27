// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package waitgroup

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Group is similar to [errgroup.Group]. It's zero value is valid, it contains
// [context.Background] as its internal context and does not cancel on error.
type Group struct {
	wg       *errgroup.Group
	ctx      context.Context
	cancel   context.CancelFunc
	initOnce sync.Once
}

func (g *Group) init() {
	g.initOnce.Do(func() {
		if g.wg == nil {
			g.wg = new(errgroup.Group)
			g.ctx = context.Background()
		}
	})
}

// WithContext returns a new [Group] similar to an [errgroup.Group], but with
// ctx as its internal context which is passed to the function(s) passed to
// [Group.Go].
func WithContext(ctx context.Context) *Group {
	return &Group{
		wg:  new(errgroup.Group),
		ctx: ctx,
	}
}

// WithTimeout returns a new [Group] similar to an [errgroup.Group], but with
// an internal context derived from [context.WithTimeout], which is passed to
// the function(s) passed to [Group.Go].
func WithTimeout(parent context.Context, timeout time.Duration) *Group {
	g := &Group{wg: new(errgroup.Group)}
	g.ctx, g.cancel = context.WithTimeout(parent, timeout)
	return g
}

// WithNotifyContext returns a new [Group] similar to [errgroup.WithContext],
// with an internal context derived from [signal.NotifyContext]. This means the
// context is canceled the first time a function passed to [Group.Go]
// returns a non-nil error, or when one of the listed signals arrives, or the
// first time [Group.Wait] returns, whichever occurs first.
func WithNotifyContext(ctx context.Context, signals ...os.Signal) *Group {
	if signals == nil {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	var g Group
	ctx, g.cancel = signal.NotifyContext(ctx, signals...)
	g.wg, g.ctx = errgroup.WithContext(ctx)
	return &g
}

// Go calls the given function in a new goroutine. The [Group]'s internal
// context is passed as argument to the function.
//
// The first call to return a non-nil error cancels the [Group]'s context, if
// the group was created by calling [WithNotifyContext]. The error will be
// returned by [Group.Wait].
func (g *Group) Go(fn func(ctx context.Context) error) {
	g.init()
	g.wg.Go(func() error {
		return fn(g.ctx)
	})
}

// Wait blocks until all function calls from [Group.Go] have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	g.init()
	if g.cancel != nil {
		defer g.cancel()
	}
	return g.wg.Wait()
}
