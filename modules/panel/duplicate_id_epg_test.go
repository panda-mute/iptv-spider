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

func TestDuplicateNumericChannelIDsAndEPGDeduplication(t *testing.T) {
	s := testService(t)

	// Two imported channels from operator: CCTV-1 (51) and CCTV-1HD (102)
	c1 := Channel{ID: "51", Name: "CCTV-1", OperatorID: "ch00000000000000001372", URL: "rtp://239.45.1.1:5140", Enabled: true, Source: "iptv"}
	c2 := Channel{ID: "102", Name: "CCTV-1HD", OperatorID: "ch00000000000000001062", URL: "rtp://239.45.1.2:5140", Enabled: true, Source: "iptv"}
	// Hunan variants: Hunan SD (203) and Hunan 4K (301)
	h1 := Channel{ID: "203", Name: "湖南卫视", OperatorID: "ch00000000000000001501", URL: "rtp://239.45.2.1:5140", Enabled: true, Source: "iptv"}
	h2 := Channel{ID: "301", Name: "湖南卫视4K", OperatorID: "ch00000000000000001888", URL: "rtp://239.45.2.2:5140", Enabled: true, Source: "iptv"}

	if err := s.Store.Import([]Channel{c1, c2, h1, h2}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Seed EPG program for canonical CCTV-1 and 湖南卫视
	now := time.Now()
	pCCTV := Programme{Title: "新闻联播", Start: now.Add(-30 * time.Minute).Unix(), End: now.Add(30 * time.Minute).Unix()}
	pHunan := Programme{Title: "快乐大本营", Start: now.Add(-30 * time.Minute).Unix(), End: now.Add(30 * time.Minute).Unix()}

	cfg, _ := s.Store.Snapshot()
	if err := s.saveEPG(map[string][]Programme{
		"ch00000000000000001372": {pCCTV},
		"ch00000000000000001501": {pHunan},
	}, cfg.EPG); err != nil {
		t.Fatalf("saveEPG failed: %v", err)
	}

	handler := s.Handler("test-token")
	putChannel := func(ref string, c Channel) int {
		t.Helper()
		b, _ := json.Marshal(c)
		r := httptest.NewRequest("PUT", "/api/panel/channels/"+ref, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	// 1. Edit CCTV-1 to digital ID "1"
	c1Edited := c1
	c1Edited.ID = "1"
	if code := putChannel("51", c1Edited); code != 200 {
		t.Fatalf("Failed to edit c1: HTTP %d", code)
	}

	// 2. Edit CCTV-1HD to digital ID "1" as well ("数字频道ID可以一样")
	c2Edited := c2
	c2Edited.ID = "1"
	if code := putChannel("102", c2Edited); code != 200 {
		t.Fatalf("Failed to edit c2 to same ID '1': HTTP %d", code)
	}

	// 3. Edit Hunan 4K to same digital ID "203" as Hunan SD
	h2Edited := h2
	h2Edited.ID = "203"
	if code := putChannel("301", h2Edited); code != 200 {
		t.Fatalf("Failed to edit h2 to same ID '203': HTTP %d", code)
	}

	// Verify all 4 channels remain in the list
	channels, err := s.Channels()
	if err != nil {
		t.Fatalf("s.Channels error: %v", err)
	}
	if len(channels) != 4 {
		t.Fatalf("Expected 4 channels, got %d", len(channels))
	}

	countID1 := 0
	countID203 := 0
	for _, ch := range channels {
		if ch.ID == "1" {
			countID1++
		}
		if ch.ID == "203" {
			countID203++
		}
	}
	if countID1 != 2 {
		t.Fatalf("Expected 2 channels with ID '1', got %d", countID1)
	}
	if countID203 != 2 {
		t.Fatalf("Expected 2 channels with ID '203', got %d", countID203)
	}

	// Verify EPG cross-definition resolution:
	// CCTV-1HD should find EPG from canonical CCTV-1
	progsCCTVHD := s.Programmes(c2Edited, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
	if len(progsCCTVHD) == 0 || progsCCTVHD[0].Title != "新闻联播" {
		t.Fatalf("CCTV-1HD failed to inherit EPG: %+v", progsCCTVHD)
	}

	// Hunan 4K should find EPG from canonical 湖南卫视
	progsHunan4K := s.Programmes(h2Edited, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
	if len(progsHunan4K) == 0 || progsHunan4K[0].Title != "快乐大本营" {
		t.Fatalf("Hunan 4K failed to inherit EPG: %+v", progsHunan4K)
	}

	// Verify M3U export includes both channels with shared tvg-id="1"
	m3u := s.M3U(channels)
	if !strings.Contains(m3u, `tvg-id="1" tvg-name="CCTV-1"`) {
		t.Errorf("M3U missing CCTV-1 tvg-id: %s", m3u)
	}
	if !strings.Contains(m3u, `tvg-id="1" tvg-name="CCTV-1HD"`) {
		t.Errorf("M3U missing CCTV-1HD tvg-id: %s", m3u)
	}

	// Verify XMLTV export deduplicates <channel id="1"> so it only appears once
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
		Programmes []struct {
			Channel string `xml:"channel,attr"`
			Title   string `xml:"title"`
		} `xml:"programme"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &tvDoc); err != nil {
		t.Fatalf("Failed to unmarshal XMLTV: %v", err)
	}

	c1XmlCount := 0
	for _, ch := range tvDoc.Channels {
		if ch.ID == "1" {
			c1XmlCount++
		}
	}
	if c1XmlCount != 1 {
		t.Errorf("Expected XMLTV <channel id=\"1\"> to appear exactly once, appeared %d times", c1XmlCount)
	}
}
