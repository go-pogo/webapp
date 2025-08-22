// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dev

package autoenv

import (
	"os"
	"runtime"
)

func NewLoader() *Loader {
	n := SkipCaller
	if SkipCaller < 0 {
		n += runtime.Callers(0, make([]uintptr, CaptureCallers))
	}
	return NewDevelopmentLoader(n, os.Args[1:]...)
}
