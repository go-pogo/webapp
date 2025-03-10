// Copyright (c) 2025, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webapp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pogo/serv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithServerConfig(t *testing.T) {
	want := ServerConfig{Port: 12345}
	base, err := New("", WithServerConfig(want))
	assert.NoError(t, err)
	assert.Equal(t, want.Port.Addr(), base.server.Addr)
}

func TestWithHealthChecker(t *testing.T) {
	t.Run("before", func(t *testing.T) {
		base, err := New("")
		require.NoError(t, err)
		assert.Nil(t, base.HealthChecker())
	})
	t.Run("after", func(t *testing.T) {
		base, err := New("", WithHealthChecker())
		assert.NoError(t, err)
		assert.NotNil(t, base.HealthChecker())
	})
}

func TestWithIgnoreFaviconRoute(t *testing.T) {
	base, err := New("", WithIgnoreFaviconRoute())
	assert.NoError(t, err)

	srv := httptest.NewServer(base.RouteHandler().(serv.Router))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/favicon.ico")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
