package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCaptureAndIdentifyEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fixture, err := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=green:s=320x240:r=25", "-t", "1", "-c:v", "mpeg2video", "-threads", "1", "-f", "mpegts", "pipe:1").Output()
	if err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			jsonResponse(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"name":"识别建议","group":"地方","confidence":0.8,"reason":"测试画面"}`}}}})
			return
		}
		w.Header().Set("Content-Type", "video/mp2t")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.Forward.Address = strings.TrimPrefix(server.URL, "http://")
	cfg.AI = AIConfig{BaseURL: server.URL + "/v1", Model: "test-vision"}
	if err = s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	c := Channel{ID: "unknown", Name: "未知频道", URL: "rtp://239.1.1.1:5140", Enabled: true}
	if err = s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	result, err := s.Identify(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := s.Channel(c.ID)
	if err != nil || result.Name != "识别建议" || saved.Name != c.Name || saved.Suggestion == nil {
		t.Fatal("identification must preserve current name and save a suggestion", saved, err)
	}
}

func testService(t *testing.T) *Service {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "panel.json"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	return NewService(store)
}

func TestSettingsAndPlayback(t *testing.T) {
	s := DefaultSettings()
	if s.Forward.Address != "192.168.190.1:4022" {
		t.Fatal("incorrect default forwarding address")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Forward.Address = "127.0.0.1:4022"
	u, err := url.Parse(PlaybackURL(s.Forward, "igmp://239.1.2.3:5140?foo=bar"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != "/rtp/239.1.2.3:5140" || u.Query().Get("fcc") != s.Forward.FCC || u.Query().Get("foo") != "bar" {
		t.Fatal(u)
	}
	u, _ = url.Parse(PlaybackURL(s.Forward, "rtp://239.1.2.3:5140?fcc=10.0.0.1:12"))
	if u.Query().Get("fcc") != "10.0.0.1:12" {
		t.Fatal("channel FCC overwritten")
	}
	if got := PlaybackURL(s.Forward, "https://example.com/live.m3u8?a=b"); got != "https://example.com/live.m3u8?a=b" {
		t.Fatal(got)
	}
	for _, mutate := range []func(*Settings){
		func(s *Settings) { s.Forward.Address = "host:0" },
		func(s *Settings) { s.Scan.StartIP = "127.0.0.1" }, func(s *Settings) { s.Scan.EndIP = "239.255.255.255" }, func(s *Settings) { s.Scan.EndPort = 0 }, func(s *Settings) { s.Scan.Workers = 99 }, func(s *Settings) { s.AI.BaseURL = "file:///tmp/a" },
	} {
		c := s
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("invalid settings accepted", c)
		}
	}
}

func TestLegacyNetworkSettingsMigration(t *testing.T) {
	for _, address := range []string{"", "10.0.0.1:4022"} {
		t.Run(address, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "panel.json")
			cfg := DefaultSettings()
			cfg.Forward.Address = address
			cfg.AI.APIKey = "keep-key"
			b, _ := json.Marshal(cfg)
			var settings map[string]any
			_ = json.Unmarshal(b, &settings)
			settings["network"] = map[string]any{"interface": "missing0", "vlan": 85, "source_ip": "30.1.1.1", "prefix_length": 24}
			c := Channel{ID: "kept", Name: "自定义", Group: "自选", URL: "igmp://239.1.1.1:5140", Enabled: true}
			b, _ = json.Marshal(map[string]any{"version": 1, "settings": settings, "channels": map[string]Channel{c.ID: c}, "imported": []Channel{c}})
			if err := os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			store, err := OpenStore(path, DefaultSettings())
			if err != nil {
				t.Fatal(err)
			}
			got, channels := store.Snapshot()
			wantAddress := address
			if wantAddress == "" {
				wantAddress = "192.168.190.1:4022"
			}
			if got.Forward.Address != wantAddress || got.AI.APIKey != "keep-key" || len(channels) != 1 || channels[0].Name != c.Name || len(store.Imported()) != 1 {
				t.Fatal("migration lost settings or channels")
			}
			b, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(b, []byte(`"network"`)) || !bytes.Contains(b, []byte(`"version": 2`)) {
				t.Fatal("obsolete network settings retained")
			}
			if _, err = OpenStore(path, DefaultSettings()); err != nil {
				t.Fatal("migrated config cannot reopen", err)
			}
		})
	}
}

func TestStoreRestartOverridesAndFailure(t *testing.T) {
	s := testService(t)
	c := Channel{ID: "1", Name: "运营商名称", URL: "igmp://239.1.1.1:5140", Enabled: true, Source: "iptv"}
	if err := s.Store.Import([]Channel{c}); err != nil {
		t.Fatal(err)
	}
	c.Name = "自定义频道"
	c.Group = "自定义组"
	c.Logo = "https://example.com/logo.png"
	c.Enabled = false
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	other := c
	other.Name = "新运营商名称"
	if err := s.Store.Import([]Channel{other}); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(s.Store.path, DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewService(store).Channel("1")
	if err != nil || got.Name != c.Name || got.Enabled || got.Group != c.Group || got.Logo != c.Logo {
		t.Fatalf("override lost: %+v %v", got, err)
	}
	info, _ := os.Stat(s.Store.path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("secrets file permissions", info.Mode())
	}
	before, _ := store.Snapshot()
	store.path = filepath.Join(s.Store.path, "invalid.json")
	after := before
	after.Forward.FCC = "10.0.0.2:80"
	if store.SaveSettings(after) == nil {
		t.Fatal("expected write failure")
	}
	actual, _ := store.Snapshot()
	if !reflect.DeepEqual(actual.Forward, before.Forward) {
		t.Fatal("failed commit changed live settings")
	}
}

func TestConcurrentStoreWrites(t *testing.T) {
	s := testService(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.Store.PutChannel(Channel{ID: fmt.Sprint(i), Name: "测试", URL: "rtp://239.1.1.1:5140", Enabled: true}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	_, all := s.Store.Snapshot()
	if len(all) != 12 {
		t.Fatal(len(all))
	}
}

func request(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestAdminAndMaskedSettings(t *testing.T) {
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.AI.APIKey = "secret-test-key"
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	h := s.Handler("admin")
	if w := request(h, "GET", "/api/panel/settings", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := request(h, "GET", "/api/panel/settings", "admin", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), cfg.AI.APIKey) {
		t.Fatal(w.Code, w.Body.String())
	}
	cfg.AI.APIKey = ""
	b, _ := json.Marshal(map[string]any{"settings": cfg})
	w = request(h, "PUT", "/api/panel/settings", "admin", string(b))
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	got, _ := s.Store.Snapshot()
	if got.AI.APIKey != "secret-test-key" {
		t.Fatal("blank key did not preserve secret")
	}
	b, _ = json.Marshal(map[string]any{"settings": cfg, "clear_api_key": true})
	w = request(h, "PUT", "/api/panel/settings", "admin", string(b))
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	got, _ = s.Store.Snapshot()
	if got.AI.APIKey != "" {
		t.Fatal("key not cleared")
	}
	r := httptest.NewRequest("POST", "/api/panel/scan/start", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Authorization", "Bearer admin")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-site mutation allowed")
	}
	r = httptest.NewRequest("GET", "/api/panel/settings", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	w = httptest.NewRecorder()
	s.Handler("").ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("remote access without token allowed")
	}
}

func TestScanFindsTSNotHTMLAndPreservesEdits(t *testing.T) {
	ts := make([]byte, 188*5)
	for i := 0; i < len(ts); i += 188 {
		ts[i] = 0x47
	}
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fcc") != "10.0.0.1:15970" {
			t.Error("scan omitted shared FCC")
		}
		if r.URL.Path == "/rtp/239.1.1.1:5140" {
			_, _ = w.Write(ts)
		} else {
			_, _ = w.Write([]byte("<html>no stream</html>"))
		}
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.Forward.Address = strings.TrimPrefix(server.URL, "http://")
	cfg.Forward.FCC = "10.0.0.1:15970"
	cfg.Scan.StartIP = "239.1.1.1"
	cfg.Scan.EndIP = "239.1.1.2"
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := s.StartScan(); err != nil {
			t.Fatal(err)
		}
		s.mu.Lock()
		done := s.scanDone
		s.mu.Unlock()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("scan hung")
		}
		job := s.Status()["scan"]
		if job.State != "completed" || job.Done != 2 || job.Found != 1 {
			t.Fatal(job)
		}
		channels, err := s.Channels()
		if err != nil || len(channels) != 1 {
			t.Fatal(channels, err)
		}
		c := channels[0]
		if attempt == 1 && c.Name != "已确认频道" {
			t.Fatal("rescan overwrote manual edit")
		}
		c.Name = "已确认频道"
		if err = s.Store.PutChannel(c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanCancellation(t *testing.T) {
	entered := make(chan struct{}, 1)
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.Forward.Address = strings.TrimPrefix(server.URL, "http://")
	cfg.Scan.Workers = 1
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.StartScan(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("scan never started")
	}
	if s.StartScan() == nil {
		t.Fatal("duplicate scan allowed")
	}
	s.StopScan()
	s.mu.Lock()
	done := s.scanDone
	s.mu.Unlock()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scan failed to cancel")
	}
	if s.Status()["scan"].State != "cancelled" {
		t.Fatal(s.Status())
	}
}

func TestRecognizeCompatibleAPI(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("incorrect API request")
		}
		var b map[string]any
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Error(err)
		}
		raw, _ := json.Marshal(b)
		if !bytes.Contains(raw, []byte("data:image/jpeg;base64,")) || b["model"] != "vision-model" {
			t.Error("missing vision input")
		}
		jsonResponse(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "```json\n{\"name\":\"测试台\",\"group\":\"地方\",\"confidence\":0.9,\"reason\":\"可见台标\"}\n```"}}}})
	}))
	defer server.Close()
	result, err := recognize(context.Background(), AIConfig{BaseURL: server.URL + "/v1", APIKey: "secret", Model: "vision-model"}, []byte("jpeg"), server.Client())
	if err != nil || result.Name != "测试台" {
		t.Fatal(result, err)
	}
}

func TestPlaylistMetadataAndLiveChanges(t *testing.T) {
	s := testService(t)
	c := Channel{ID: "one", Name: "测试频道", Group: "自选", Logo: "https://example.com/a.png", URL: "igmp://239.1.1.1:5140", Enabled: true}
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"10.0.0.1:4022", "10.0.0.2:4022"} {
		cfg, _ := s.Store.Snapshot()
		cfg.Forward.Address = address
		if err := s.Store.SaveSettings(cfg); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		s.ServePlaylist(w, httptest.NewRequest("GET", "/api/m3u8", nil))
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, address+"/rtp/") || !strings.Contains(body, `group-title="自选"`) || !strings.Contains(body, `tvg-logo="https://example.com/a.png"`) {
			t.Fatal(body)
		}
	}
	c.Enabled = false
	_ = s.Store.PutChannel(c)
	w := httptest.NewRecorder()
	s.ServePlaylist(w, httptest.NewRequest("GET", "/api/m3u8", nil))
	if strings.Contains(w.Body.String(), "#EXTINF") {
		t.Fatal("disabled channel exported")
	}
}

func TestEditChannelIDPersistsWithoutDuplicates(t *testing.T) {
	s := testService(t)
	original := Channel{ID: "51", Name: "测试频道", Group: "央视", URL: "rtp://239.1.1.1:5140", Enabled: true, Source: "iptv"}
	other := original
	other.ID = "52"
	if err := s.Store.Import([]Channel{original, other}); err != nil {
		t.Fatal(err)
	}
	edited := original
	edited.ID = "custom-51"
	edited.Group = "自定义分组"
	edited.Logo = "https://example.com/logo.png"
	put := func(oldID string, c Channel) int {
		t.Helper()
		b, _ := json.Marshal(c)
		r := httptest.NewRequest("PUT", "/api/panel/channels/"+oldID, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer test")
		w := httptest.NewRecorder()
		s.Handler("test").ServeHTTP(w, r)
		return w.Code
	}
	if code := put("51", edited); code != 200 {
		t.Fatal("rename", code)
	}
	if err := s.Store.Import([]Channel{original, other}); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(s.Store.path, DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	s = NewService(store)
	list, err := s.Channels()
	if err != nil || len(list) != 2 {
		t.Fatal("duplicate after sync/restart", list, err)
	}
	got, err := s.Channel(edited.ID)
	if err != nil || got.OriginalID != "51" || got.Group != edited.Group || got.Logo != edited.Logo || got.Source != "iptv" {
		t.Fatal(got, err)
	}
	if _, err = s.Channel("51"); err == nil {
		t.Fatal("old ID visible")
	}
	if !strings.Contains(s.M3U(list), `tvg-id="custom-51"`) || !strings.Contains(s.M3U(list), `tvg-logo="https://example.com/logo.png"`) {
		t.Fatal("export metadata missing")
	}
	collision := got
	collision.ID = "52"
	if code := put(got.ID, collision); code != 200 {
		t.Fatal("duplicate digital ID allowed", code)
	}
	stale := original
	if code := put("51", stale); code == 200 {
		t.Fatal("stale editor recreated old ID")
	}
	renamed := collision
	renamed.ID = "second-id"
	if code := put(collision.ID, renamed); code != 200 {
		t.Fatal("second rename", code)
	}
	renamed.ID = "51"
	if code := put("second-id", renamed); code != 200 {
		t.Fatal("restore original ID", code)
	}
	list, _ = s.Channels()
	if len(list) != 2 {
		t.Fatal("duplicate after restore", list)
	}
	manual := original
	manual.ID = "manual"
	manual.OriginalID = "52"
	if code := put("manual", manual); code != 200 {
		t.Fatal(code)
	}
	got, _ = s.Channel("manual")
	if got.OriginalID != "" {
		t.Fatal("client can hide another channel")
	}
	manual.ID = "manual-new"
	if code := put("manual", manual); code != 200 {
		t.Fatal("manual rename", code)
	}
	list, _ = s.Channels()
	if len(list) != 3 {
		t.Fatal("manual duplicate", list)
	}
}

func TestAutomaticLogoMatchingAndManualPriority(t *testing.T) {
	for _, name := range []string{"CCTV-1", "CCTV1HD", "CCTV-5+HD", "东方卫视HD", "体育频道", "风云足球HD"} {
		if got := channelLogo(name); !strings.HasPrefix(got, "https://raw.githubusercontent.com/sggc/SDU-IPTV-PRO/main/logo/") {
			t.Errorf("no match for %s", name)
		}
	}
	if channelLogo("CCTV-5") == channelLogo("CCTV-5+") || channelLogo("CCTV-4") == channelLogo("CCTV-4K") {
		t.Fatal("distinct channels conflated")
	}
	if channelLogo("未知频道123") != "" {
		t.Fatal("guessed an unknown channel")
	}
	names := parseLogoCatalog([]byte(`{"logos":[{"name":"中文频道","file":"中文 台标.webp"},{"name":"invalid","file":"../bad.png"}]}`))
	if len(names) != 1 || strings.Contains(names["中文频道"], " ") {
		t.Fatal(names)
	}
	s := testService(t)
	c := Channel{ID: "51", Name: "CCTV-1HD", URL: "rtp://239.1.1.1:5140", Enabled: true}
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Channel(c.ID)
	if !got.LogoAutomatic || got.Logo == "" {
		t.Fatal(got)
	}
	_, stored := s.Store.Snapshot()
	if stored[0].Logo != "" {
		t.Fatal("automatic logo persisted as manual")
	}
	c.Logo = "https://example.com/custom.png"
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Channel(c.ID)
	if got.LogoAutomatic || got.Logo != c.Logo {
		t.Fatal("manual logo overwritten", got)
	}
	c.Logo = ""
	c.Name = "东方卫视"
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Channel(c.ID)
	if got.Logo != channelLogo(c.Name) {
		t.Fatal("rename did not rematch", got)
	}
}
