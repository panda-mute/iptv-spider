package panel

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestLogoSourceRegionAndPriority(t *testing.T) {
	cfg := defaultLogoSources()
	if v := configuredLogo(cfg, "CCTV-1HD", ""); !strings.Contains(v, "raw.githubusercontent.com") {
		t.Fatal(v)
	}
	if v := configuredLogo(cfg, "新闻综合", ""); !strings.Contains(v, "raw.githubusercontent.com") || !strings.Contains(v, ".png") {
		t.Fatal(v)
	}
	// No longer prioritizes foreign source for CCTV-1
	if v := configuredLogo(cfg, "CCTV-1", "foreign"); !strings.Contains(v, "raw.githubusercontent.com") {
		t.Fatal("expected github logo without domestic/foreign distinction", v)
	}
	names := githubLogos([]string{"logo/CCTV1.png", "logo/sub/CCTV1.png", "logo/sub/中文.png", "other/a.png", "logo/readme.txt"}, "owner/repo", "main", "logo")
	if len(names) != 2 || !strings.HasSuffix(names["CCTV1"], "/logo/CCTV1.png") || strings.Contains(names["中文"], "中文") {
		t.Fatal(names)
	}
	s := testService(t)
	settings, _ := s.Store.Snapshot()
	sources := settings.Logos.GetSources()
	if len(sources) == 0 || sources[0] != PrimaryLogoSource {
		t.Fatal("default source missing", sources)
	}
}
func TestConfiguredLogoSourceRefreshAndPersist(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/list" {
			t.Error(r.URL.Path)
		}
		w.Write([]byte(`{"logos":[{"name":"TEST","file":"test image.webp"}]}`))
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.Logos = LogoSources{Primary: server.URL}
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	result := s.RefreshLogoSources(context.Background())
	if result["primary_count"] != 1 {
		t.Fatal(result)
	}
	value := configuredLogo(cfg.Logos, "TEST", "domestic")
	if !strings.HasPrefix(value, server.URL) || !strings.Contains(value, "%20") {
		t.Fatal(value)
	}
	sourceCatalogs.Lock()
	delete(sourceCatalogs.data, server.URL)
	sourceCatalogs.Unlock()
	if err := s.LoadLogoSources(); err != nil {
		t.Fatal(err)
	}
	if got := configuredLogo(cfg.Logos, "TEST", "domestic"); got != value {
		t.Fatal(got)
	}
	server.Close()
	if err := refreshSource(context.Background(), server.URL); err == nil {
		t.Fatal("network failure expected")
	}
	if got := configuredLogo(cfg.Logos, "TEST", "domestic"); got != value {
		t.Fatal("failed refresh discarded catalog")
	}
}

func TestDarkLogoDeprioritizationAndFirstFinancial(t *testing.T) {
	cfg := defaultLogoSources()
	// 第一财经 must match Shanghai First Financial (normal, non-dark)
	v := configuredLogo(cfg, "第一财经", "")
	if uv, _ := url.PathUnescape(v); !strings.Contains(uv, "上海第一财经.png") || strings.Contains(uv, "深色") {
		t.Fatalf("expected normal 上海第一财经 logo, got: %s", v)
	}

	// 五星体育 must match Shanghai Five Star Sports (normal, non-dark)
	v2 := configuredLogo(cfg, "五星体育", "")
	if uv, _ := url.PathUnescape(v2); !strings.Contains(uv, "五星体育.png") || strings.Contains(uv, "深色") {
		t.Fatalf("expected normal 五星体育 logo, got: %s", v2)
	}

	// 风云足球 must match CCTV 风云足球 (normal, non-dark)
	v3 := configuredLogo(cfg, "风云足球", "")
	if uv, _ := url.PathUnescape(v3); !strings.Contains(uv, "风云足球.png") || strings.Contains(uv, "深色") {
		t.Fatalf("expected normal 风云足球 logo, got: %s", v3)
	}
}
