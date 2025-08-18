// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dev

package autoenv

import (
	"os"
	"runtime"
)

func newLoader() Loader {
	n := runtime.Callers(0, make([]uintptr, 16))
	return NewDevelopmentLoader(n-4, os.Args[1:]...)
}
