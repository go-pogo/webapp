// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contextgroup

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-pogo/errors/errgroup"
	"github.com/go-pogo/errors/errlist"
)

var _ errlist.ErrorLister = (*Group)(nil)

// Group is similar to [errgroup.Group]. The main difference is [Group.Go]
// accepts a function with a [context.Context] as its first argument.
//
// It's zero value is valid, it contains [context.Background] as its internal
// context and does not cancel on error.
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

// New returns a new [Group] with ctx as its internal context, which is passed
// to the function(s) passed to [Group.Go]. The context is not canceled when a
// function passed to [Group.Go] returns an error.
func New(ctx context.Context) *Group {
	return &Group{
		wg:  new(errgroup.Group),
		ctx: ctx,
	}
}

// WithNotifyContext returns a new [Group] with an internal context derived
// from [signal.NotifyContext] and its cancel function. This means the
// context is canceled the first time a function passed to [Group.Go]
// returns a non-nil error, or when one of the listed signals arrives, or the
// first time [Group.Wait] returns, whichever occurs first.
func WithNotifyContext(parent context.Context, signals ...os.Signal) *Group {
	if signals == nil {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	g := &Group{wg: new(errgroup.Group)}
	g.ctx, g.cancel = signal.NotifyContext(parent, signals...)
	return g
}

// ErrorList returns an [errlist.List] of collected errors from the called
// functions passed to [Group.Go].
func (g *Group) ErrorList() *errlist.List {
	g.init()
	return g.wg.ErrorList()
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

// Go calls the given function in a new goroutine. The [Group]'s internal
// context is passed as argument to the function.
//
// The first call to return a non-nil error cancels the [Group]'s context, if
// it was created by calling [WithNotifyContext]. The error will be returned
// by [Group.Wait].
func (g *Group) Go(fn func(ctx context.Context) error) {
	g.init()
	g.wg.Go(func() error {
		if g.cancel == nil {
			return fn(g.ctx)
		}

		if err := fn(g.ctx); err != nil {
			g.cancel()
			return err
		}
		return nil
	})
}
