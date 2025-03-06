// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rungroup

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-pogo/errors/errgroup"
)

// Group is similar to an [errgroup.Group] created with [errgroup.WithContext].
// Its internal context is derived from [signal.NotifyContext] and is canceled
// when one of the listed signals arrives. Another difference is [Group.Go]
// accepts a function with a [context.Context] as its first argument.
//
// It's zero value is valid and by default listens to the arrival of signals
// [syscall.SIGINT] and/or [syscall.SIGTERM]. It uses [context.Background] as
// first argument to the function passed to [Group.Go].
type Group struct {
	grp        *errgroup.Group
	ctx        context.Context
	stopNotify context.CancelFunc
	initOnce   sync.Once
}

// New returns a [Group] with its context derived from [signal.NotifyContext]
// and passes the provided signals to it. It defaults to [syscall.SIGINT] and
// [syscall.SIGTERM] when no signals are provided.
//
// The derived [context.Context] is canceled the first time a function passed
// to [Group.Go] returns a non-nil error, or when one of the listed signals
// arrives, or the first time [Group.Wait] returns, whichever occurs first.
func New(parent context.Context, signals ...os.Signal) *Group {
	var g Group
	g.init(parent, signals)
	return &g
}

func (g *Group) init(ctx context.Context, signals []os.Signal) {
	g.initOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		if signals == nil {
			signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
		}

		ctx, g.stopNotify = signal.NotifyContext(ctx, signals...)
		g.grp, g.ctx = errgroup.WithContext(ctx)
	})
}

// Wait blocks until all function calls from [Group.Go] have returned, or when
// the internal context is canceled. It then returns all collected errors as a
// (multi) error (if any).
func (g *Group) Wait() error {
	g.init(nil, nil)
	go func() {
		// stop receiving signals when g.grp is done
		defer g.stopNotify()
		// Wait calls its internal context.CancelCauseFunc and adds any
		// collected errors to it, these can be retrieved via context.Cause
		_ = g.grp.Wait()
	}()

	<-g.ctx.Done()
	return context.Cause(g.ctx)
}

// Go calls the given function in a new goroutine. The [Group]'s internal
// context is passed as argument to the function.
func (g *Group) Go(fn func(ctx context.Context) error) {
	g.init(nil, nil)
	g.grp.Go(func() error {
		return fn(g.ctx)
	})
}
