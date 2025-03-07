// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ctxgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var grp Group
		grp.Go(func(ctx context.Context) error {
			assert.Equal(t, context.Background(), ctx)
			return nil
		})
		assert.NoError(t, grp.Wait())
	})

	t.Run("parent ctx", func(t *testing.T) {
		ctx, cancelFn := context.WithCancel(context.Background())
		defer cancelFn()

		grp := New(ctx)
		grp.Go(func(have context.Context) error {
			assert.Equal(t, ctx, have)
			return nil
		})
		assert.NoError(t, grp.Wait())
	})
}
