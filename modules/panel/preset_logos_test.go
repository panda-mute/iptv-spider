package panel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePresetLogosOnce(t *testing.T) {
	dir := t.TempDir()
	s := &Service{LogosDir: dir}

	// 1. Initial run should copy logos and create .initialized marker
	if err := s.EnsurePresetLogos(); err != nil {
		t.Fatalf("EnsurePresetLogos failed: %v", err)
	}

	marker := filepath.Join(dir, ".initialized")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected .initialized marker to exist: %v", err)
	}

	testFile := filepath.Join(dir, "CCTV-1.png")
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("expected CCTV-1.png to exist in logos dir: %v", err)
	}

	// 2. Modify one file to verify it is NOT overwritten on subsequent runs
	customData := []byte("custom-cctv1-data")
	if err := os.WriteFile(testFile, customData, 0644); err != nil {
		t.Fatalf("failed to write custom data: %v", err)
	}

	if err := s.EnsurePresetLogos(); err != nil {
		t.Fatalf("second EnsurePresetLogos failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	if string(data) != string(customData) {
		t.Fatalf("EnsurePresetLogos overwrote existing file: got %s, want %s", string(data), string(customData))
	}
}

func TestFindLocalLogo(t *testing.T) {
	dir := t.TempDir()
	s := &Service{LogosDir: dir}
	if err := s.EnsurePresetLogos(); err != nil {
		t.Fatalf("EnsurePresetLogos failed: %v", err)
	}

	tests := []struct {
		name     string
		wantLogo string
	}{
		// 1. CCTV rules: CCTV-1 / CCTV-1HD share logo
		{"CCTV-1", "/logos/CCTV-1.png"},
		{"CCTV-1HD", "/logos/CCTV-1.png"},
		{"CCTV-1高清", "/logos/CCTV-1.png"},
		{"CCTV1HD", "/logos/CCTV-1.png"},
		{"CCTV-5+HD", "/logos/CCTV-5+.png"},
		{"CCTV-16 4K", "/logos/CCTV-16 4K.png"},
		{"CCTV-16HD", "/logos/CCTV-16.png"},
		{"CCTV-4K", "/logos/CCTV-4K.png"},

		// 2. 卫视 / 卫视HD share logo, 卫视4K has distinct logo
		{"浙江卫视", "/logos/浙江卫视.png"},
		{"浙江卫视HD", "/logos/浙江卫视.png"},
		{"浙江卫视高清", "/logos/浙江卫视.png"},
		{"浙江卫视 4K", "/logos/浙江卫视4K.png"},
		{"浙江卫视4K", "/logos/浙江卫视4K.png"},
		{"东方卫视", "/logos/东方卫视.png"},
		{"东方卫视HD", "/logos/东方卫视.png"},
		{"东方卫视4K", "/logos/东方卫视4K.png"},
		{"北京卫视HD", "/logos/北京卫视.png"},
		{"北京卫视4K", "/logos/北京卫视4K.png"},

		// 3. 东方购物-1 / 东方购物-2 share logo
		{"东方购物-1", "/logos/东方购物.png"},
		{"东方购物-2", "/logos/东方购物.png"},
		{"东方购物", "/logos/东方购物.png"},

		// 4. Aliases
		{"体育频道", "/logos/五星体育.png"},
		{"卡酷卡通", "/logos/卡酷少儿.png"},

		// 5. Special HD channels that have their own file
		{"CHC动作电影HD", "/logos/CHC动作电影HD.png"},
		{"多彩文体HD", "/logos/多彩文体HD.png"},
		{"多彩文体4K", "/logos/多彩文体4K.png"},

		// 6. Unknown / empty channels
		{"", ""},
		{"未知频道", ""},
		{"未知频道 (233.18.204.206:5140)", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.FindLocalLogo(tc.name)
			if got != tc.wantLogo {
				t.Errorf("FindLocalLogo(%q) = %q, want %q", tc.name, got, tc.wantLogo)
			}
		})
	}
}

func TestChannelsPrioritizesLocalLogos(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store)
	s.LogosDir = filepath.Join(dir, "logos")
	if err := s.EnsurePresetLogos(); err != nil {
		t.Fatal(err)
	}

	// Channel without logo should get local logo
	c1 := Channel{ID: "1", Name: "CCTV-1HD", URL: "igmp://233.18.204.210:5140", Enabled: true}
	// Channel with 4K variation should get 4K local logo
	c2 := Channel{ID: "2", Name: "浙江卫视 4K", URL: "igmp://233.18.204.229:5140", Enabled: true}
	// Channel with manual custom logo should NOT be overwritten
	c3 := Channel{ID: "3", Name: "湖南卫视", Logo: "https://example.com/custom.png", URL: "igmp://233.18.204.227:5140", Enabled: true}

	if err := s.Store.PutChannel(c1); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.PutChannel(c2); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.PutChannel(c3); err != nil {
		t.Fatal(err)
	}

	channels, err := s.Channels()
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]Channel{}
	for _, c := range channels {
		byName[c.Name] = c
	}

	if ch, ok := byName["CCTV-1HD"]; !ok || ch.Logo != "/logos/CCTV-1.png" || !ch.LogoAutomatic {
		t.Errorf("expected CCTV-1HD to get /logos/CCTV-1.png, got: %+v", ch)
	}
	if ch, ok := byName["浙江卫视 4K"]; !ok || ch.Logo != "/logos/浙江卫视4K.png" || !ch.LogoAutomatic {
		t.Errorf("expected 浙江卫视 4K to get /logos/浙江卫视4K.png, got: %+v", ch)
	}
	if ch, ok := byName["湖南卫视"]; !ok || ch.Logo != "https://example.com/custom.png" || ch.LogoAutomatic {
		t.Errorf("expected 湖南卫视 manual logo to be preserved, got: %+v", ch)
	}
}

func TestServeLocalLogoFromEmbeddedFallback(t *testing.T) {
	// Empty directory without extracted files should still serve embedded preset logos
	dir := t.TempDir()
	s := &Service{LogosDir: dir}

	r := httptest.NewRequest("GET", "/logos/CCTV-1.png", nil)
	w := httptest.NewRecorder()
	s.ServeLocalLogo(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("expected Content-Type image/png, got %s", ct)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Fatalf("expected non-empty body")
	}
}

func TestMatchLogoRegionWithLocalLogos(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store)
	s.LogosDir = filepath.Join(dir, "logos")
	if err := s.EnsurePresetLogos(); err != nil {
		t.Fatal(err)
	}

	match, err := s.MatchLogoRegion(context.Background(), "浙江卫视HD", "卫视", "")
	if err != nil {
		t.Fatalf("MatchLogoRegion failed: %v", err)
	}
	if match.Logo != "/logos/浙江卫视.png" || match.Confidence != 1.0 {
		t.Errorf("expected /logos/浙江卫视.png with confidence 1.0, got %+v", match)
	}
}
