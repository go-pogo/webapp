// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build dev

package logger

import (
	"fmt"
	"os"

	"github.com/go-pogo/errors"
	"github.com/rs/zerolog"
)

func New(conf Config) *Logger { return NewDevelopmentLogger(conf) }

func init() {
	zerolog.ErrorMarshalFunc = func(err error) interface{} {
		if errors.GetStackTrace(err) != nil {
			// print complete stack trace for easier debugging during development
			_, _ = fmt.Fprintf(os.Stdout, "\n%+v\n", err)
		}

		return fmt.Sprintf("%v", err)
	}
}
