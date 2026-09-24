package panel

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"CCTV-1", "CCTV-1"},
		{"新闻综合 (高清)", "新闻综合_高清"},
		{"Channel / Test : * ? \" < > |", "Channel_Test"},
		{"  Hello   World  ", "Hello_World"},
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsLocalLogo(t *testing.T) {
	if !isLocalLogo("/logos/cctv1.png") {
		t.Error("expected /logos/cctv1.png to be local")
	}
	if !isLocalLogo("/api/panel/logos/local/cctv1.png") {
		t.Error("expected /api/panel/logos/local/ to be local")
	}
	if isLocalLogo("https://example.com/logo.png") {
		t.Error("remote url should not be local")
	}
	if isLocalLogo("/logos/bad\r\npath.png") {
		t.Error("newline in path should be rejected")
	}
}

func TestFCCListStoreAndSettings(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := store.Snapshot()
	if len(snap.Forward.FCCList) == 0 {
		t.Fatal("expected default FCCList to be populated")
	}
	defaultLen := len(snap.Forward.FCCList)

	// Add custom FCC
	customFCC := "10.20.30.40:15970"
	snap.Forward.FCC = customFCC
	if err := store.SaveSettings(snap); err != nil {
		t.Fatal(err)
	}

	snap2, _ := store.Snapshot()
	if snap2.Forward.FCC != customFCC {
		t.Errorf("got FCC %q, want %q", snap2.Forward.FCC, customFCC)
	}
	found := false
	for _, f := range snap2.Forward.FCCList {
		if f == customFCC {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("custom FCC %q was not saved to FCCList", customFCC)
	}
	if len(snap2.Forward.FCCList) <= defaultLen {
		t.Errorf("expected FCCList to grow, got %d <= %d", len(snap2.Forward.FCCList), defaultLen)
	}

	// AddFCCs helper
	if err := store.AddFCCs("10.50.60.70:15970", "124.75.26.151:15970"); err != nil {
		t.Fatal(err)
	}
	snap3, _ := store.Snapshot()
	foundDiscovered := false
	for _, f := range snap3.Forward.FCCList {
		if f == "10.50.60.70:15970" {
			foundDiscovered = true
			break
		}
	}
	if !foundDiscovered {
		t.Error("AddFCCs did not add 10.50.60.70:15970")
	}

	// Channel with local logo validation
	c := Channel{
		ID:      "test-chan",
		Name:    "测试频道",
		Group:   "央视",
		URL:     "rtp://239.1.1.1:5140",
		Logo:    "/logos/test_chan_123.png",
		Enabled: true,
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Channel.Validate failed for local logo: %v", err)
	}
}


func TestSaveUploadedLogo(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store)
	s.LogosDir = filepath.Join(dir, "logos")

	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}

	url, err := s.SaveUploadedLogo("东方卫视 4K", "custom_logo.png", bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("SaveUploadedLogo failed: %v", err)
	}
	if !strings.HasPrefix(url, "/logos/东方卫视_4K_") || !strings.HasSuffix(url, ".png") {
		t.Errorf("unexpected saved logo URL: %s", url)
	}

	savedFile := filepath.Join(s.LogosDir, strings.TrimPrefix(url, "/logos/"))
	data, err := os.ReadFile(savedFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if !bytes.Equal(data, pngBytes) {
		t.Errorf("saved file content mismatch")
	}

	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="40"/></svg>`
	svgUrl, err := s.SaveUploadedLogo("CCTV-1", "test.svg", strings.NewReader(svgContent))
	if err != nil {
		t.Fatalf("SaveUploadedLogo SVG failed: %v", err)
	}
	if !strings.HasSuffix(svgUrl, ".svg") {
		t.Errorf("expected .svg extension, got %s", svgUrl)
	}

	if _, err := s.SaveUploadedLogo("test", "empty.png", strings.NewReader("")); err == nil {
		t.Error("expected error for empty upload, got nil")
	}

	if _, err := s.SaveUploadedLogo("test", "malicious.txt", strings.NewReader("not an image at all")); err == nil {
		t.Error("expected error for text upload, got nil")
	}
}

func TestUploadLogoHTTPRoute(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store)
	s.LogosDir = filepath.Join(dir, "logos")

	token := "test-secret-token"
	handler := s.Handler(token)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "江苏卫视4K")
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	_, _ = part.Write(pngBytes)
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/panel/logos/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp["url"], "/logos/江苏卫视4K_") {
		t.Errorf("unexpected url in response: %v", resp)
	}

	getReq := httptest.NewRequest("GET", resp["url"], nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != 200 {
		t.Fatalf("expected HTTP 200 serving uploaded logo, got %d", getRec.Code)
	}
	if !bytes.Equal(getRec.Body.Bytes(), pngBytes) {
		t.Errorf("served logo body does not match uploaded bytes")
	}
}

func TestWebAssetsServed(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store)
	handler := s.Handler("test-token")

	for _, path := range []string{"/", "/app.js", "/style.css"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("expected 200 for %s, got %d", path, rec.Code)
		}
		if path == "/" && !strings.Contains(rec.Body.String(), "upload-logo-btn") {
			t.Errorf("index.html (via /) does not contain upload-logo-btn")
		}
		if path == "/app.js" && !strings.Contains(rec.Body.String(), "uploadLogoFile") {
			t.Errorf("app.js does not contain uploadLogoFile")
		}
		if path == "/style.css" && !strings.Contains(rec.Body.String(), "width: 135px") {
			t.Errorf("style.css does not contain max-width: 140px for channel-name-text")
		}
	}
}
