// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !dev

package webapp

func NewLogger(conf LoggerConfig) *Logger { return NewProductionLogger(conf) }
