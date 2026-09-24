package panel

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInvalidChannelIDDefinitions(t *testing.T) {
	if InvalidChannelID != "0" {
		t.Fatalf("Expected InvalidChannelID to be \"0\", got %s", InvalidChannelID)
	}
	invalidCases := []string{"", "0", "-1", "none", "unknown", "scan-239.1.1.1:5140"}
	for _, tc := range invalidCases {
		if !IsInvalidChannelID(tc) {
			t.Errorf("Expected IsInvalidChannelID(%q) to be true", tc)
		}
		if IsValidChannelID(tc) {
			t.Errorf("Expected IsValidChannelID(%q) to be false", tc)
		}
	}
	validCases := []string{"1", "2", "51", "102", "custom", "CCTV-1"}
	for _, tc := range validCases {
		if IsInvalidChannelID(tc) {
			t.Errorf("Expected IsInvalidChannelID(%q) to be false", tc)
		}
		if !IsValidChannelID(tc) {
			t.Errorf("Expected IsValidChannelID(%q) to be true", tc)
		}
	}
}

func TestUnidentifiedChannelWithNumericIDCanBeDisabled(t *testing.T) {
	s := testService(t)
	handler := s.Handler("test-token")

	// 1. Multiple scanned / unidentified channels discovered with default invalid ID "0"
	u1 := Channel{
		Key:     "scan-239.45.1.1:5140",
		ID:      InvalidChannelID,
		Name:    "未知频道 239.45.1.1:5140",
		Group:   "待识别",
		URL:     "rtp://239.45.1.1:5140",
		Enabled: true,
		Source:  "scan",
	}
	u2 := Channel{
		Key:     "scan-239.45.1.2:5140",
		ID:      InvalidChannelID,
		Name:    "未知频道 239.45.1.2:5140",
		Group:   "待识别",
		URL:     "rtp://239.45.1.2:5140",
		Enabled: true,
		Source:  "scan",
	}

	if err := s.Store.Discover(u1); err != nil {
		t.Fatalf("Discover u1 failed: %v", err)
	}
	if err := s.Store.Discover(u2); err != nil {
		t.Fatalf("Discover u2 failed: %v", err)
	}

	// Verify both channels exist despite sharing default InvalidChannelID "0"
	list, err := s.Channels()
	if err != nil {
		t.Fatalf("Channels() error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Expected 2 discovered channels, got %d", len(list))
	}

	putReq := func(ref string, c Channel) (int, string) {
		b, _ := json.Marshal(c)
		r := httptest.NewRequest("PUT", "/api/panel/channels/"+ref, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code, w.Body.String()
	}

	// 2. Unidentified channel can be disabled directly while having default invalid ID "0"
	u1Disabled := u1
	u1Disabled.Enabled = false
	code, body := putReq("scan-239.45.1.1:5140", u1Disabled)
	if code != 200 {
		t.Fatalf("Failed to disable u1 with invalid ID: HTTP %d %s", code, body)
	}
	gotU1, err := s.Channel("scan-239.45.1.1:5140")
	if err != nil || gotU1.Enabled {
		t.Fatalf("u1 was not disabled: enabled=%v, err=%v", gotU1.Enabled, err)
	}

	// Re-enable u1
	u1Disabled.Enabled = true
	code, body = putReq("scan-239.45.1.1:5140", u1Disabled)
	if code != 200 {
		t.Fatalf("Failed to re-enable u1: HTTP %d %s", code, body)
	}

	// 3. User assigns a numeric ID to the unidentified channel (e.g. ID = "99")
	u1WithNumber := u1
	u1WithNumber.ID = "99"
	u1WithNumber.Name = "体育综合"
	u1WithNumber.Group = "体育"
	code, body = putReq("scan-239.45.1.1:5140", u1WithNumber)
	if code != 200 {
		t.Fatalf("Failed to give numeric ID to u1: HTTP %d %s", code, body)
	}

	gotAfterNumber, err := s.Channel("scan-239.45.1.1:5140")
	if err != nil {
		t.Fatalf("Channel not found by key after setting numeric ID: %v", err)
	}
	if gotAfterNumber.ID != "99" {
		t.Fatalf("Expected ID 99, got %s", gotAfterNumber.ID)
	}

	// 4. THE CORE BUG REPRODUCTION:
	// Once an unidentified channel is given a numeric ID, can it be disabled?
	// User edits or toggles disabled on this channel
	u1ToDisable := gotAfterNumber
	u1ToDisable.Enabled = false
	// Frontend sends channelRef (which is key: "scan-239.45.1.1:5140")
	code, body = putReq("scan-239.45.1.1:5140", u1ToDisable)
	if code != 200 {
		t.Fatalf("CRITICAL BUG: Unidentified channel given a numeric ID failed to disable! HTTP %d: %s", code, body)
	}

	gotDisabled, err := s.Channel("scan-239.45.1.1:5140")
	if err != nil {
		t.Fatalf("Lookup failed after disable: %v", err)
	}
	if gotDisabled.Enabled {
		t.Fatalf("Channel is still enabled, expected disabled!")
	}

	// Verify M3U export excludes disabled channel
	allChannels, _ := s.Channels()
	m3u := s.M3U(allChannels)
	if strings.Contains(m3u, "体育综合") {
		t.Fatalf("Disabled channel appeared in M3U: %s", m3u)
	}

	// Verify channel can also be disabled/enabled referencing by its current numeric ID "99"
	u1ToEnable := gotDisabled
	u1ToEnable.Enabled = true
	code, body = putReq("99", u1ToEnable)
	if code != 200 {
		t.Fatalf("Failed to re-enable using ID 99: HTTP %d %s", code, body)
	}
	gotReEnabled, err := s.Channel("99")
	if err != nil || !gotReEnabled.Enabled {
		t.Fatalf("Channel failed to re-enable: enabled=%v, err=%v", gotReEnabled.Enabled, err)
	}
}

func TestInvalidChannelIDExcludedFromXMLTVAndM3UTvgID(t *testing.T) {
	s := testService(t)
	cValid := Channel{ID: "51", Name: "CCTV-1", URL: "rtp://239.45.1.1:5140", Enabled: true, Source: "iptv"}
	cInvalid0 := Channel{Key: "scan-1", ID: "0", Name: "未知1", URL: "rtp://239.45.1.2:5140", Enabled: true, Source: "scan"}
	cInvalidNeg := Channel{Key: "scan-2", ID: "-1", Name: "未知2", URL: "rtp://239.45.1.3:5140", Enabled: true, Source: "scan"}

	_ = s.Store.Import([]Channel{cValid})
	_ = s.Store.Discover(cInvalid0)
	_ = s.Store.Discover(cInvalidNeg)

	now := time.Now()
	p := Programme{Title: "新闻联播", Start: now.Add(-time.Hour).Unix(), End: now.Add(time.Hour).Unix()}
	cfg, _ := s.Store.Snapshot()
	_ = s.saveEPG(map[string][]Programme{"51": {p}}, cfg.EPG)

	w := httptest.NewRecorder()
	s.ServeEPG(w, httptest.NewRequest("GET", "/api/epg", nil))
	if w.Code != 200 {
		t.Fatalf("ServeEPG failed: %d", w.Code)
	}

	var tvDoc struct {
		Channels []struct {
			ID   string `xml:"id,attr"`
			Name string `xml:"display-name"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &tvDoc); err != nil {
		t.Fatalf("XML parse failed: %v", err)
	}

	for _, ch := range tvDoc.Channels {
		if ch.ID == "0" || ch.ID == "-1" {
			t.Fatalf("Invalid channel ID %q found in XMLTV export!", ch.ID)
		}
	}

	channels, _ := s.Channels()
	m3u := s.M3U(channels)
	if strings.Contains(m3u, `tvg-id="0"`) || strings.Contains(m3u, `tvg-id="-1"`) {
		t.Fatalf("Invalid channel ID found in tvg-id of M3U: %s", m3u)
	}
}
