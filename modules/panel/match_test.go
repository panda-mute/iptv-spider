package panel

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestMatchChannelName(t *testing.T) {
	candidates := []Channel{
		{ID: "51", Name: "CCTV-1", Source: "iptv", OperatorID: "ch1001", Group: "央视"},
		{ID: "51", Name: "CCTV-1HD", Source: "iptv", OperatorID: "ch1001hd", Group: "央视"},
		{ID: "52", Name: "CCTV-2HD", Source: "iptv", OperatorID: "ch1002hd", Group: "央视"},
		{ID: "54", Name: "CCTV-4HD", Source: "iptv", OperatorID: "ch1004hd", Group: "央视"},
		{ID: "70", Name: "CCTV-5+HD", Source: "iptv", OperatorID: "ch1005plus", Group: "央视"},
		{ID: "99", Name: "CCTV-4K", Source: "iptv", OperatorID: "ch1004k", Group: "央视"},
		{ID: "139", Name: "CCTV-16HD", Source: "iptv", OperatorID: "ch1016hd", Group: "央视"},
		{ID: "161", Name: "金鹰纪实HD", Source: "iptv", OperatorID: "ch1161", Group: "高清"},
		{ID: "169", Name: "中国教育-4HD", Source: "iptv", OperatorID: "ch1169", Group: "高清"},
		{ID: "1227", Name: "湖南卫视HD", Source: "iptv", OperatorID: "ch1227", Group: "高清"},
		{ID: "1058", Name: "东方卫视HD", Source: "iptv", OperatorID: "ch1058", Group: "高清"},
		// Invalid ID channels that should be ignored
		{ID: "0", Name: "未知频道 233.18.204.158:5140", Source: "scan"},
		{ID: "scan-233.18.204.160:5140", Name: "未知频道", Source: "scan"},
	}

	tests := []struct {
		name        string
		query       string
		wantID      string
		wantName    string
		wantMatched bool
	}{
		{
			name:        "cctv16 matches CCTV-16HD",
			query:       "cctv16",
			wantID:      "139",
			wantName:    "CCTV-16HD",
			wantMatched: true,
		},
		{
			name:        "CCTV16 matches CCTV-16HD",
			query:       "CCTV16",
			wantID:      "139",
			wantName:    "CCTV-16HD",
			wantMatched: true,
		},
		{
			name:        "CCTV-16 matches CCTV-16HD",
			query:       "CCTV-16",
			wantID:      "139",
			wantName:    "CCTV-16HD",
			wantMatched: true,
		},
		{
			name:        "CCTV-16 奥林匹克 matches CCTV-16HD",
			query:       "CCTV-16 奥林匹克",
			wantID:      "139",
			wantName:    "CCTV-16HD",
			wantMatched: true,
		},
		{
			name:        "cctv1 matches CCTV-1HD",
			query:       "cctv1",
			wantID:      "51",
			wantName:    "CCTV-1HD",
			wantMatched: true,
		},
		{
			name:        "cctv1 does not match CCTV-16HD",
			query:       "cctv1",
			wantID:      "51",
			wantMatched: true,
		},
		{
			name:        "cctv5+ matches CCTV-5+HD",
			query:       "cctv5+",
			wantID:      "70",
			wantName:    "CCTV-5+HD",
			wantMatched: true,
		},
		{
			name:        "cctv4 does not match CCTV-4K",
			query:       "cctv4",
			wantID:      "54",
			wantName:    "CCTV-4HD",
			wantMatched: true,
		},
		{
			name:        "cctv4k matches CCTV-4K",
			query:       "cctv4k",
			wantID:      "99",
			wantName:    "CCTV-4K",
			wantMatched: true,
		},
		{
			name:        "湖南卫视 matches 湖南卫视HD",
			query:       "湖南卫视",
			wantID:      "1227",
			wantName:    "湖南卫视HD",
			wantMatched: true,
		},
		{
			name:        "hunan matches 湖南卫视HD",
			query:       "湖南",
			wantID:      "1227",
			wantName:    "湖南卫视HD",
			wantMatched: true,
		},
		{
			name:        "东方卫视 matches 东方卫视HD",
			query:       "东方卫视",
			wantID:      "1058",
			wantName:    "东方卫视HD",
			wantMatched: true,
		},
		{
			name:        "金鹰纪实 matches 金鹰纪实HD",
			query:       "金鹰纪实",
			wantID:      "161",
			wantName:    "金鹰纪实HD",
			wantMatched: true,
		},
		{
			name:        "cetv4 matches 中国教育-4HD",
			query:       "cetv4",
			wantID:      "169",
			wantName:    "中国教育-4HD",
			wantMatched: true,
		},
		{
			name:        "unrelated channel returns nil",
			query:       "Discovery Channel 探索频道",
			wantMatched: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchChannelName(tc.query, candidates)
			if !tc.wantMatched {
				if got != nil {
					t.Fatalf("expected nil for query %q, got %+v", tc.query, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected match for query %q, got nil", tc.query)
			}
			if got.ID != tc.wantID {
				t.Errorf("got ID %q, want %q", got.ID, tc.wantID)
			}
			if tc.wantName != "" && got.Name != tc.wantName {
				t.Errorf("got Name %q, want %q", got.Name, tc.wantName)
			}
		})
	}
}

func TestMatchChannelEndpointAndService(t *testing.T) {
	s := testService(t)
	handler := s.Handler("test-token")

	// Set up imported channels in store
	imported := []Channel{
		{Key: "ch1016", ID: "139", Name: "CCTV-16HD", Group: "央视", OperatorID: "ch1016hd", Source: "iptv", URL: "rtp://239.1.1.1:5140", Enabled: true},
		{Key: "ch1001", ID: "51", Name: "CCTV-1HD", Group: "央视", OperatorID: "ch1001hd", Source: "iptv", URL: "rtp://239.1.1.2:5140", Enabled: true},
		{Key: "ch1227", ID: "1227", Name: "湖南卫视HD", Group: "高清", OperatorID: "ch1227hd", Source: "iptv", URL: "rtp://239.1.1.3:5140", Enabled: true},
	}
	if err := s.Store.Import(imported); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// 1. Service method s.MatchChannel("cctv16")
	matched := s.MatchChannel("cctv16")
	if matched == nil {
		t.Fatalf("Expected s.MatchChannel(\"cctv16\") to match, got nil")
	}
	if matched.ID != "139" {
		t.Errorf("Expected matched ID \"139\", got %q", matched.ID)
	}
	if matched.Name != "CCTV-16HD" {
		t.Errorf("Expected matched Name \"CCTV-16HD\", got %q", matched.Name)
	}

	// 2. HTTP Endpoint GET /api/panel/channels/match?name=cctv16
	req := httptest.NewRequest("GET", "/api/panel/channels/match?name=cctv16", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Matched    bool   `json:"matched"`
		ID         string `json:"id"`
		Name       string `json:"name"`
		OperatorID string `json:"operator_id"`
		Group      string `json:"group"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !resp.Matched {
		t.Fatalf("Expected matched == true")
	}
	if resp.ID != "139" {
		t.Errorf("Expected ID \"139\", got %q", resp.ID)
	}
	if resp.Name != "CCTV-16HD" {
		t.Errorf("Expected Name \"CCTV-16HD\", got %q", resp.Name)
	}
	if resp.OperatorID != "ch1016hd" {
		t.Errorf("Expected OperatorID \"ch1016hd\", got %q", resp.OperatorID)
	}

	// 3. HTTP Endpoint GET /api/panel/channels/match for non-existent channel
	req2 := httptest.NewRequest("GET", "/api/panel/channels/match?name=unknown999", nil)
	req2.Header.Set("Authorization", "Bearer test-token")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w2.Code)
	}
	var resp2 struct {
		Matched bool `json:"matched"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if resp2.Matched {
		t.Errorf("Expected matched == false for unknown channel")
	}
}

func TestMatchChannelCandidatesAndKeywordMappings(t *testing.T) {
	s := testService(t)
	s.Store.PutChannel(Channel{
		ID:         "139",
		Name:       "CCTV-16 4K",
		OperatorID: "ch1016_4k",
		Group:      "央视频道",
		URL:        "rtp://239.45.1.16:5140",
		Enabled:    true,
	})
	s.Store.PutChannel(Channel{
		ID:         "139",
		Name:       "CCTV-16HD",
		OperatorID: "ch1016_hd",
		Group:      "央视频道",
		URL:        "rtp://239.45.1.17:5140",
		Enabled:    true,
	})
	s.Store.PutChannel(Channel{
		ID:         "37",
		Name:       "上海第一财经",
		OperatorID: "ch1037",
		Group:      "本地频道",
		URL:        "rtp://239.45.1.37:5140",
		Enabled:    true,
	})

	// 1. Match candidates returns multiple options for cctv16
	cands := s.MatchChannelCandidates("cctv16", 10)
	if len(cands) < 2 {
		t.Fatalf("expected at least 2 candidates for cctv16, got %d", len(cands))
	}
	if cands[0].ID != "139" || cands[1].ID != "139" {
		t.Errorf("expected candidates to have ID 139, got %s, %s", cands[0].ID, cands[1].ID)
	}

	// 2. Province prefix matching: 第一财经 -> 上海第一财经
	cands1cai := s.MatchChannelCandidates("第一财经", 5)
	if len(cands1cai) == 0 || cands1cai[0].Name != "上海第一财经" || cands1cai[0].ID != "37" {
		t.Fatalf("expected 第一财经 to match 上海第一财经 (ID 37), got %+v", cands1cai)
	}

	// 3. Save keyword mapping: mapping "cctv16-olympic" -> CCTV-16 4K (ID 139)
	err := s.Store.SaveMapping(ChannelMapping{
		Keyword:    "cctv16-olympic",
		TargetID:   "139",
		TargetName: "CCTV-16 4K",
		OperatorID: "ch1016_4k",
		Group:      "央视频道",
	})
	if err != nil {
		t.Fatalf("failed to save mapping: %v", err)
	}

	// 4. Verify FindMapping and Mappings
	m, found := s.Store.FindMapping("cctv16-olympic")
	if !found || m.TargetID != "139" || m.TargetName != "CCTV-16 4K" {
		t.Fatalf("expected mapping to be found, got %+v", m)
	}
	allMappings := s.Store.Mappings()
	if len(allMappings) != 1 || allMappings[0].Keyword != "cctv16-olympic" {
		t.Fatalf("expected 1 mapping, got %+v", allMappings)
	}

	// 5. Querying with mapped keyword returns saved candidate at top with IsSaved == true
	mappedCands := s.MatchChannelCandidates("cctv16-olympic", 5)
	if len(mappedCands) == 0 || !mappedCands[0].IsSaved || mappedCands[0].ID != "139" {
		t.Fatalf("expected top candidate to be saved mapping, got %+v", mappedCands)
	}

	// 6. Test HTTP endpoints
	handler := s.Handler("test-token")

	// GET /api/panel/mappings
	req := httptest.NewRequest("GET", "/api/panel/mappings", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("GET /mappings returned %d", w.Code)
	}

	// POST /api/panel/mappings
	postBody, _ := json.Marshal(ChannelMapping{
		Keyword:    "yicai",
		TargetID:   "37",
		TargetName: "上海第一财经",
		OperatorID: "ch1037",
		Group:      "本地频道",
	})
	reqPost := httptest.NewRequest("POST", "/api/panel/mappings", bytes.NewReader(postBody))
	reqPost.Header.Set("Authorization", "Bearer test-token")
	wPost := httptest.NewRecorder()
	handler.ServeHTTP(wPost, reqPost)
	if wPost.Code != 200 {
		t.Fatalf("POST /mappings returned %d: %s", wPost.Code, wPost.Body.String())
	}

	// DELETE /api/panel/mappings/yicai
	reqDel := httptest.NewRequest("DELETE", "/api/panel/mappings/yicai", nil)
	reqDel.Header.Set("Authorization", "Bearer test-token")
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, reqDel)
	if wDel.Code != 200 {
		t.Fatalf("DELETE /mappings/yicai returned %d", wDel.Code)
	}
}
