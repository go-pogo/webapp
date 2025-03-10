// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package rungroup

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/go-pogo/errors"
	"github.com/stretchr/testify/assert"
)

func TestGroup_Wait(t *testing.T) {
	t.Run("cancel after received signal", func(t *testing.T) {
		grp := New(context.Background(), syscall.SIGUNUSED)
		grp.Go(func(ctx context.Context) error {
			timeoutCtx, cancelFn := context.WithTimeout(ctx, 1000*time.Millisecond)
			defer cancelFn()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timeoutCtx.Done():
				assert.Fail(t, "context canceled due to timeout")
			}
			return nil
		})
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGUNUSED); err != nil {
			t.Logf("error during syscall %s", err)
		}

		// wait so the above signal can be intercepted
		time.Sleep(50 * time.Millisecond)
		// break out of Wait as soon as a listed Signal is received
		assert.Error(t, grp.Wait(), context.Canceled)
	})
	t.Run("cancel after function error", func(t *testing.T) {
		var want errors.Msg = "some err"
		var grp Group
		grp.Go(func(ctx context.Context) error {
			return errors.New(want)
		})
		grp.Go(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
				t.Logf("error during syscall %s", err)
			}
			return nil
		})

		// break out of Wait as soon as an error is returned via on of the
		// functions passed to Go
		assert.Error(t, grp.Wait(), want)
	})
}
