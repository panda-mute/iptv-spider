package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelLogoMatchCatalogValidation(t *testing.T) {
	for _, tc := range []struct {
		name, answer        string
		wantLogo, wantError bool
	}{
		{"valid", "```json\n{\"name\":\"CCTV1\",\"confidence\":0.96,\"reason\":\"相同频道\"}\n```", true, false},
		{"low", `{"name":"CCTV1","confidence":0.6,"reason":"不确定"}`, false, false},
		{"none", `{"name":"","confidence":0,"reason":"没有匹配"}`, false, false},
		{"invented", `{"name":"https://evil.test/a.png","confidence":1}`, false, true},
		{"missing confidence", `{"name":"CCTV1"}`, false, true},
		{"invalid", `{"name":"CCTV1","confidence":2}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer configured-key" {
					t.Error("wrong configured API/key")
				}
				var payload struct {
					Model    string
					Messages []struct{ Content string }
				}
				if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.Model != "configured-model" || len(payload.Messages) != 2 || !strings.Contains(payload.Messages[1].Content, "CCTV1") {
					t.Error("model/catalog missing")
				}
				jsonResponse(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": tc.answer}}}})
			}))
			defer server.Close()
			cfg := AIConfig{BaseURL: server.URL + "/v1", APIKey: "configured-key", Model: "configured-model"}
			got, err := matchLogo(context.Background(), cfg, "中央一套", "央视", server.Client())
			if (err != nil) != tc.wantError {
				t.Fatal(got, err)
			}
			if err == nil && (got.Logo != "") != tc.wantLogo {
				t.Fatal(got)
			}
			if err == nil && got.Logo != "" && got.Logo != channelLogo("CCTV-1") {
				t.Fatal("not catalog URL", got)
			}
		})
	}
}

func TestLogoMatchEndpointDoesNotOverwriteChannel(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"name":"CCTV1","confidence":0.99,"reason":"别名一致"}`}}}})
	}))
	defer server.Close()
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.AI = AIConfig{BaseURL: server.URL, Model: "test"}
	if err := s.Store.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	c := Channel{ID: "51", Name: "中央一套", Logo: "https://example.com/manual.png", URL: "rtp://239.1.1.1:5140", Enabled: true}
	s.Store.PutChannel(c)
	for _, token := range []string{"", "test"} {
		r := httptest.NewRequest("POST", "/api/panel/logos/match", bytes.NewBufferString(`{"name":"中央一套","group":"央视"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.Handler("test").ServeHTTP(w, r)
		if token == "" && w.Code != 401 {
			t.Fatal(w.Code)
		}
		if token != "" && (w.Code != 200 || !strings.Contains(w.Body.String(), "raw.githubusercontent.com")) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	got, _ := s.Channel(c.ID)
	if got.Logo != c.Logo {
		t.Fatal("manual logo overwritten")
	}
	s.identify <- struct{}{}
	if _, err := s.MatchLogo(context.Background(), c.Name, ""); err == nil {
		t.Fatal("concurrent task allowed")
	}
	<-s.identify
	if _, err := s.MatchLogo(context.Background(), "", ""); err == nil {
		t.Fatal("empty name allowed")
	}
}
