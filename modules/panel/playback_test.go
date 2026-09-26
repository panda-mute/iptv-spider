package panel

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUnicastAndCatchupExport(t *testing.T) {
	s := testService(t)
	c := Channel{ID: "51", Name: "test", URL: "igmp://239.1.1.1:5140", UnicastURL: "rtsp://10.1.1.1:554/live/a?AuthInfo=a%2Bb", OperatorID: "operator-51", CatchupDays: 7, Enabled: true}
	if err := s.Store.Import([]Channel{c}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"multicast", "unicast", "http"} {
		w := httptest.NewRecorder()
		s.ServePlaylist(w, httptest.NewRequest("GET", "http://panel.test/api/playlist?mode="+mode, nil))
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, `catchup-days="7"`) || !strings.Contains(body, "utc=${start}&lutc=${end}") {
			t.Fatal(mode, w.Code, body)
		}
		target := map[string]string{"multicast": "/rtp/239.1.1.1:5140", "unicast": "/rtsp/10.1.1.1:554/live/a?AuthInfo=a%2Bb", "http": "http://panel.test/api/play?id=51&mode=http"}[mode]
		if !strings.Contains(body, target) {
			t.Fatal(mode, body)
		}
	}
	// Metadata overrides and renamed IDs still follow refreshed signed URLs.
	c.ID = "custom"
	c.OriginalID = "51"
	c.Group = "new"
	if err := s.Store.PutChannel(c); err != nil {
		t.Fatal(err)
	}
	refreshed := c
	refreshed.ID = "51"
	refreshed.OriginalID = ""
	refreshed.UnicastURL = "rtsp://10.1.1.1:554/live/new?AuthInfo=new"
	if err := s.Store.Import([]Channel{refreshed}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Channel("custom")
	if got.UnicastURL != refreshed.UnicastURL || got.OperatorID != "operator-51" {
		t.Fatal(got)
	}
	cfg, _ := s.Store.Snapshot()
	cfg.Forward.PlayMode = "unicast"
	s.Store.SaveSettings(cfg)
	w := httptest.NewRecorder()
	s.ServePlaylist(w, httptest.NewRequest("GET", "http://panel.test/api/playlist?mode=multicast", nil))
	if !strings.Contains(w.Body.String(), "/rtp/") {
		t.Fatal("explicit multicast ignored")
	}
}
func TestCatchupRedirectAndValidation(t *testing.T) {
	s := testService(t)
	c := Channel{ID: "1", Name: "test", URL: "rtp://239.1.1.1:5140", UnicastURL: "rtsp://10.1.1.1:554/live/a?AuthInfo=a%2Bb", CatchupDays: 7, Enabled: true}
	s.Store.PutChannel(c)
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)
	query := "&start=" + strconv.FormatInt(start.Unix(), 10) + "&end=" + strconv.FormatInt(end.Unix(), 10)
	w := httptest.NewRecorder()
	s.ServePlay(w, httptest.NewRequest("GET", "/api/play?id=1&mode=unicast"+query, nil))
	u, _ := url.Parse(w.Header().Get("Location"))
	zone := time.FixedZone("CST", 8*3600)
	if w.Code != 302 || u.Query().Get("AuthInfo") != "a+b" || u.Query().Get("playseek") != start.In(zone).Format("20060102150405")+"-"+end.In(zone).Format("20060102150405") {
		t.Fatal(w.Code, u)
	}
	for _, bad := range []string{"&start=abc&end=4", "&start=1&end=2", "&start=" + strconv.FormatInt(end.Unix(), 10) + "&end=" + strconv.FormatInt(start.Unix(), 10)} {
		w = httptest.NewRecorder()
		s.ServePlay(w, httptest.NewRequest("GET", "/api/play?id=1&mode=unicast"+bad, nil))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	c.OperatorID = "operator-1"
	s.Store.PutChannel(c)
	cfg, _ := s.Store.Snapshot()
	cfg.IPTV = IPTV{Enabled: true, UID: "u", SN: "s", MAC: "m", IP: "10.0.0.1", AuthHost: "host:7001"}
	s.Store.SaveSettings(cfg)
	called := false
	s.ResolveHTTP = func(_ context.Context, _ Settings, id string, a, b time.Time) (string, error) {
		called = true
		if id != "operator-1" || !a.Equal(start) || !b.Equal(end) {
			t.Error(id, a, b)
		}
		return "http://stream.test/live.m3u8?token=signed", nil
	}
	w = httptest.NewRecorder()
	s.ServePlay(w, httptest.NewRequest("GET", "/api/play?id=1&mode=http"+query, nil))
	if !called || w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "token=signed") {
		t.Fatal(w.Code)
	}
}

func TestTiviMateAndTelevizoCatchupFormats(t *testing.T) {
	s := testService(t)
	c := Channel{
		ID:          "51",
		Name:        "东方卫视",
		URL:         "rtp://239.1.1.1:5140",
		UnicastURL:  "rtsp://10.1.1.1:554/live/a?AuthInfo=token",
		OperatorID:  "op-51",
		CatchupDays: 7,
		Enabled:     true,
		Group:       "卫视频道",
	}
	if err := s.Store.Import([]Channel{c}); err != nil {
		t.Fatal(err)
	}

	// 1. Verify M3U export has TiviMate & Televizo compatibility tags
	w := httptest.NewRecorder()
	s.ServePlaylist(w, httptest.NewRequest("GET", "http://panel.test/api/playlist?mode=unicast", nil))
	body := w.Body.String()
	if !strings.Contains(body, `catchup="default"`) {
		t.Fatalf("missing catchup=default in M3U: %s", body)
	}
	if !strings.Contains(body, `x-tvg-url="http://panel.test/api/epg"`) {
		t.Fatalf("missing x-tvg-url in M3U header: %s", body)
	}
	if !strings.Contains(body, `url-tvg="http://panel.test/api/epg"`) {
		t.Fatalf("missing url-tvg in M3U header: %s", body)
	}
	if !strings.Contains(body, `tvg-chno="51"`) {
		t.Fatalf("missing tvg-chno in M3U: %s", body)
	}
	if !strings.Contains(body, `timeshift="7"`) {
		t.Fatalf("missing timeshift in M3U: %s", body)
	}
	if !strings.Contains(body, `catchup-days="7"`) {
		t.Fatalf("missing catchup-days in M3U: %s", body)
	}
	if !strings.Contains(body, `utc=${start}&lutc=${end}`) {
		t.Fatalf("missing utc=${start}&lutc=${end} template in M3U: %s", body)
	}

	// Custom catchup template support
	cfgCustom, _ := s.Store.Snapshot()
	cfgCustom.Forward.CatchupTemplate = "&start={utc}&end={utcend}"
	s.Store.SaveSettings(cfgCustom)
	wCustom := httptest.NewRecorder()
	s.ServePlaylist(wCustom, httptest.NewRequest("GET", "http://panel.test/api/playlist?mode=unicast", nil))
	if !strings.Contains(wCustom.Body.String(), `start={utc}&end={utcend}`) {
		t.Fatalf("custom catchup template in M3U failed: %s", wCustom.Body.String())
	}

	now := time.Now()
	startPast := now.Add(-3 * time.Hour).Truncate(time.Second)
	endPast := startPast.Add(time.Hour)

	// 2. TiviMate {utc} & {utcend}
	w = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&start="+strconv.FormatInt(startPast.Unix(), 10)+"&end="+strconv.FormatInt(endPast.Unix(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "playseek=") {
		t.Fatalf("TiviMate {utc}/{utcend} failed: code %d, loc %s", w.Code, w.Header().Get("Location"))
	}

	// 3. Televizo &utc=${start}&lutc=${end}
	zone := time.FixedZone("CST", 8*3600)
	cstStart := startPast.In(zone).Format("20060102150405")
	cstEnd := endPast.In(zone).Format("20060102150405")

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&utc="+strconv.FormatInt(startPast.Unix(), 10)+"&lutc="+strconv.FormatInt(endPast.Unix(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "playseek="+cstStart+"-"+cstEnd) {
		t.Fatalf("Televizo utc/lutc failed: code %d, loc %s", w.Code, w.Header().Get("Location"))
	}

	// 4. Televizo utc/utcend aliases
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&utc="+strconv.FormatInt(startPast.Unix(), 10)+"&utcend="+strconv.FormatInt(endPast.Unix(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 {
		t.Fatalf("Televizo utc/utcend failed: %d", w.Code)
	}

	// 4. Televizo start + duration
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&start="+strconv.FormatInt(startPast.Unix(), 10)+"&duration=3600", nil)
	s.ServePlay(w, req)
	if w.Code != 302 {
		t.Fatalf("start+duration failed: %d", w.Code)
	}

	// 5. Millisecond unix timestamps (13 digits)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&start="+strconv.FormatInt(startPast.UnixMilli(), 10)+"&end="+strconv.FormatInt(endPast.UnixMilli(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 {
		t.Fatalf("millisecond timestamps failed: %d", w.Code)
	}

	// 6. CST datetime strings (14 digits: YYYYMMDDHHMMSS)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&playseek="+cstStart+"-"+cstEnd, nil)
	s.ServePlay(w, req)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "playseek="+cstStart+"-"+cstEnd) {
		t.Fatalf("playseek query failed: code %d, loc %s", w.Code, w.Header().Get("Location"))
	}

	// 7. Timeshift ongoing live show (start is in past, end is in future)
	startOngoing := now.Add(-30 * time.Minute).Truncate(time.Second)
	endOngoing := now.Add(30 * time.Minute).Truncate(time.Second)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?id=51&mode=unicast&start="+strconv.FormatInt(startOngoing.Unix(), 10)+"&end="+strconv.FormatInt(endOngoing.Unix(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 {
		t.Fatalf("timeshift ongoing show failed (expected 302, got %d): %s", w.Code, w.Body.String())
	}

	// 8. HEAD request support
	w = httptest.NewRecorder()
	req = httptest.NewRequest("HEAD", "/api/play?id=51&mode=unicast&start="+strconv.FormatInt(startPast.Unix(), 10)+"&end="+strconv.FormatInt(endPast.Unix(), 10), nil)
	s.ServePlay(w, req)
	if w.Code != 302 || w.Header().Get("Location") == "" {
		t.Fatalf("HEAD request failed: %d", w.Code)
	}

	// 9. Query by channel name fallback
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/play?channel=东方卫视&mode=unicast", nil)
	s.ServePlay(w, req)
	if w.Code != 302 {
		t.Fatalf("channel name lookup failed: %d", w.Code)
	}
}
