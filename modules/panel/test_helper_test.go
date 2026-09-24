package panel

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t testing.TB, h http.Handler) *httptest.Server {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping test server in restricted sandbox: %v", err)
		return nil
	}
	ts := httptest.NewUnstartedServer(h)
	ts.Listener = l
	ts.Start()
	return ts
}
