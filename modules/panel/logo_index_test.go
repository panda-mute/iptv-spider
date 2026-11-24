package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexCatalogMatching(t *testing.T) {
	names, dark, err := parseIndexCatalog([]byte(`[
		{"name":"CCTV-1","path":"logo/国外/CCTV-1.png"},
		{"name":"CCTV1","path":"logo/国内/CCTV/CCTV1__variant.png","variant":true},
		{"name":"CCTV1HD","path":"logo/国内/CCTV/CCTV1.png"},
		{"name":"上海第一财经","path":"logo/国内/地方/上海/上海第一财经.png"},
		{"name":"Dark","path":"logo/dark/Dark.png"},
		{"name":"Bad","path":"logo/../Bad.png"},
		{"name":"Bad2","path":"https://other.test/Bad.png"},
		{"name":"Bad3","path":"logo/Bad.txt"}
	]`), "https://logos.test/")
	if err != nil {
		t.Fatal(err)
	}
	if got := names["CCTV1"]; got != "https://logos.test/logo/%E5%9B%BD%E5%86%85/CCTV/CCTV1.png" {
		t.Fatalf("expected domestic primary artwork, got %s", got)
	}
	if names["第一财经"] == "" || names["第一财经"] != names["上海第一财经"] {
		t.Fatal("province alias missing", names)
	}
	if len(names) != 3 || len(dark) != 1 || names["DARK"] != "" {
		t.Fatal("invalid path or dark artwork in normal catalog", names, dark)
	}
}

func TestIndexSourceRefreshAndPersist(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"测试频道","path":"logo/国内/测试 频道.png","variant":false}]`))
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.Logos = LogoSources{Sources: []string{server.URL + "/"}}
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	result := s.RefreshLogoSources(context.Background())
	if errs := result["errors"].([]string); len(errs) != 0 {
		t.Fatal(errs)
	}
	value := configuredLogo(cfg.Logos, "测试频道", "")
	if !strings.HasPrefix(value, server.URL+"/logo/") || !strings.Contains(value, "%20") {
		t.Fatal(value)
	}
	sourceCatalogs.Lock()
	delete(sourceCatalogs.data, server.URL)
	sourceCatalogs.Unlock()
	if err := s.LoadLogoSources(); err != nil {
		t.Fatal(err)
	}
	if got := configuredLogo(cfg.Logos, "测试频道", ""); got != value {
		t.Fatalf("catalog not restored: %q", got)
	}
}

func TestLegacyDefaultLogoMigration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		logos   LogoSources
		migrate bool
	}{
		{"default", LogoSources{Sources: []string{legacyPrimaryLogoSource + "/"}}, true},
		{"legacy field", LogoSources{Primary: legacyPrimaryLogoSource}, true},
		{"custom", LogoSources{Sources: []string{"https://custom.test"}}, false},
		{"multiple", LogoSources{Sources: []string{legacyPrimaryLogoSource, "https://custom.test"}}, false},
		{"legacy secondary", LogoSources{Primary: legacyPrimaryLogoSource, Secondary: "https://custom.test"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := DefaultSettings()
			settings.Logos = tc.logos
			b, err := json.Marshal(state{Version: 2, Settings: settings})
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(t.TempDir(), "panel.json")
			if err := os.WriteFile(file, b, 0600); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ { // Reopen to verify the migration was persisted.
				store, err := OpenStore(file, DefaultSettings())
				if err != nil {
					t.Fatal(err)
				}
				got, _ := store.Snapshot()
				want := tc.logos.GetSources()
				if tc.migrate {
					want = []string{PrimaryLogoSource}
				}
				if strings.Join(got.Logos.GetSources(), "\n") != strings.Join(want, "\n") {
					t.Fatal("unexpected sources", got.Logos.GetSources())
				}
			}
		})
	}
}
