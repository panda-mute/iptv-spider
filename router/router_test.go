package router

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"iptv-spider/modules/panel"
)

func TestPanelRoutesAndLegacyExports(t *testing.T) {
	store, err := panel.OpenStore(filepath.Join(t.TempDir(), "panel.json"), panel.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	panel.Current = panel.NewService(store)
	defer func() { panel.Current = nil }()
	t.Setenv("IPTV_PANEL_TOKEN", "test-token")
	app := iris.New()
	InitRouters(app)
	if err = app.Build(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, token string
		status              int
		contains            string
	}{
		{"GET", "/", "", 200, "IPTV WORKSPACE"}, {"GET", "/app.js", "", 200, "loadChannels"}, {"GET", "/style.css", "", 200, "font-family"},
		{"GET", "/api/panel/settings", "", 401, "令牌"}, {"GET", "/api/panel/settings", "test-token", 200, `"address":"192.168.190.1:4022"`},
		{"GET", "/api/m3u8", "", 200, "#EXTM3U"}, {"GET", "/api/playlist?fmt=json", "", 200, "[]"}, {"GET", "/api/run?task=update-chi", "", 410, "POST"},
		{"GET", "/api/epg", "", 503, "尚未同步"}, {"GET", "/api/tsM3u8", "", 200, "#EXTM3U"},
		{"GET", "/api/panel/interfaces", "test-token", 404, ""},
		{"POST", "/api/panel/network/apply", "test-token", 404, ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			app.ServeHTTP(w, r)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("status %d, body %s", w.Code, w.Body)
			}
			if tc.path == "/" || (tc.path == "/api/panel/settings" && tc.status == 200) {
				if strings.Contains(w.Body.String(), "VLAN") || strings.Contains(w.Body.String(), `"network"`) {
					t.Fatal("removed network settings are still exposed")
				}
			}
		})
	}
}
