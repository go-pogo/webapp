// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ctxgroup

import (
	"context"

	"github.com/go-pogo/errors/errgroup"
)

// Group is similar to [errgroup.Group]. The main difference is [Group.Go]
// accepts a function with a [context.Context] as its first argument.
//
// It's zero value is valid, it contains [context.Background] as its internal
// context and does not cancel on error.
type Group struct {
	grp errgroup.Group
	ctx context.Context
}

// New returns a new [Group] with ctx as its internal context, which is passed
// to the function(s) passed to [Group.Go]. The context is not canceled when a
// function passed to [Group.Go] returns an error.
func New(ctx context.Context) *Group {
	return &Group{ctx: ctx}
}

func (g *Group) context() context.Context {
	if g.ctx == nil {
		return context.Background()
	}
	return g.ctx
}

// Wait blocks until all function calls from the [Group.Go] method have
// returned, then returns all collected errors as a (multi) error.
func (g *Group) Wait() error { return g.grp.Wait() }

// Go calls the given function in a new goroutine. The [Group]'s internal
// context is passed as argument to the function. Errors from all calls are
// collected, combined and returned by [Group.Wait].
func (g *Group) Go(fn func(ctx context.Context) error) {
	g.grp.Go(func() error {
		return fn(g.context())
	})
}
