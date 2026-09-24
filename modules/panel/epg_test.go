package panel

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEPGStorageAndRenamedChannelXML(t *testing.T) {
	for _, backend := range []string{"file", "mysql"} {
		t.Run(backend, func(t *testing.T) {
			s := testService(t)
			var saved []byte
			if backend == "mysql" {
				s.EPGBackend = backend
				s.SaveEPGData = func(b []byte) error { saved = append([]byte(nil), b...); return nil }
				s.LoadEPGData = func() ([]byte, error) {
					if saved == nil {
						return nil, os.ErrNotExist
					}
					return saved, nil
				}
			}
			c := Channel{ID: "51", Name: "台名 & <频道>", OperatorID: "op1", URL: "rtp://239.1.1.1:5140", Enabled: true}
			s.Store.Import([]Channel{c})
			now := time.Now()
			p := Programme{Title: "节目 & <标题>", Start: now.Add(-time.Hour).Unix(), End: now.Add(time.Hour).Unix(), Desc: "精彩节目详情 & 剧情介绍"}
			cfg, _ := s.Store.Snapshot()
			if err := s.saveEPG(map[string][]Programme{"op1": {p, p, {Title: "bad", Start: 5, End: 1}}}, cfg.EPG); err != nil {
				t.Fatal(err)
			}
			if err := s.OpenEPG(); err != nil {
				t.Fatal(err)
			}
			c.ID = "custom"
			c.OriginalID = "51"
			s.Store.PutChannel(c)
			w := httptest.NewRecorder()
			s.ServeEPG(w, httptest.NewRequest("GET", "/api/epg", nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), `channel="custom"`) || strings.Contains(w.Body.String(), `channel="51"`) {
				t.Fatal(w.Code, w.Body.String())
			}
			var doc struct {
				Programmes []struct {
					Title string `xml:"title"`
					Desc  struct {
						Lang  string `xml:"lang,attr"`
						Value string `xml:",chardata"`
					} `xml:"desc"`
				} `xml:"programme"`
			}
			if err := xml.Unmarshal(w.Body.Bytes(), &doc); err != nil || len(doc.Programmes) != 1 || doc.Programmes[0].Title != p.Title || doc.Programmes[0].Desc.Value != p.Desc || doc.Programmes[0].Desc.Lang != "zh" {
				t.Fatal(doc, err)
			}
			s.SaveEPGData = func([]byte) error { return errors.New("disk/db error") }
			if err := s.saveEPG(map[string][]Programme{"op1": {{Title: "replacement", Start: p.Start, End: p.End}}}, cfg.EPG); err == nil {
				t.Fatal("write failure accepted")
			}
			if got := s.Programmes(c, p.Start, p.End); len(got) != 1 || got[0].Title != p.Title {
				t.Fatal("failed commit changed state", got)
			}
			if backend == "file" {
				fresh := NewService(s.Store)
				if err := fresh.OpenEPG(); err != nil {
					t.Fatal(err)
				}
				if fresh.EPGStatus().Programmes != 1 {
					t.Fatal("restart lost EPG")
				}
				st, _ := os.Stat(filepath.Join(filepath.Dir(s.Store.path), "epg.json"))
				if st.Mode().Perm() != 0600 {
					t.Fatal(st.Mode())
				}
			}
		})
	}
}
func TestEPGPartialRefreshAndSchedule(t *testing.T) {
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.IPTV = IPTV{Enabled: true, UID: "u", SN: "s", MAC: "m", IP: "10.0.0.1", AuthHost: "host:7001"}
	s.Store.SaveSettings(cfg)
	now := time.Now().Unix()
	p := Programme{Title: "old", Start: now - 50, End: now + 50}
	s.saveEPG(map[string][]Programme{"failed": {p}}, cfg.EPG)
	block := make(chan struct{})
	s.FetchEPG = func(ctx context.Context, cfg Settings, progress func(int, int, int)) (map[string][]Programme, error) {
		<-block
		progress(2, 2, 1)
		return map[string][]Programme{"ok": {{Title: "new", Start: now - 50, End: now + 50}}}, errors.New("one channel failed")
	}
	if err := s.RefreshEPG(); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshEPG(); err == nil {
		t.Fatal("duplicate job")
	}
	close(block)
	for i := 0; i < 100 && s.EPGStatus().State == "running"; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	status := s.EPGStatus()
	if status.State != "failed" || status.Channels != 2 || status.Done != 2 {
		t.Fatal(status)
	}
	s.MaybeRefreshEPG()
	if s.EPGStatus().State == "running" {
		t.Fatal("retried immediately")
	}
	w := httptest.NewRecorder()
	s.ServeEPG(w, httptest.NewRequest("GET", "/api/epg?daysAgo=-1", nil))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}

func TestEPGAndChannelSyncDoNotAuthenticateConcurrently(t *testing.T) {
	s := testService(t)
	cfg, _ := s.Store.Snapshot()
	cfg.IPTV = IPTV{Enabled: true, UID: "u", SN: "s", MAC: "m", IP: "10.0.0.1", AuthHost: "host:7001"}
	s.Store.SaveSettings(cfg)
	s.FetchEPG = func(context.Context, Settings, func(int, int, int)) (map[string][]Programme, error) { return nil, nil }
	s.Fetch = func(Settings, bool) error { return nil }
	s.mu.Lock()
	s.refresh.State = "running"
	s.mu.Unlock()
	if err := s.RefreshEPG(); err == nil {
		t.Fatal("EPG overlapped channel authentication")
	}
	s.mu.Lock()
	s.refresh.State = "idle"
	s.epg.State = "running"
	s.mu.Unlock()
	if err := s.Refresh(false); err == nil {
		t.Fatal("channel authentication overlapped EPG")
	}
}

func TestEPGProgrammeDescriptionPreservationAndXMLTV(t *testing.T) {
	s := testService(t)
	c := Channel{ID: "c1", Name: "测试频道", OperatorID: "op_test", URL: "rtp://239.1.1.1:5140", Enabled: true}
	s.Store.Import([]Channel{c})

	now := time.Now()
	p1 := Programme{Title: "早间新闻", Start: now.Add(-2 * time.Hour).Unix(), End: now.Add(-time.Hour).Unix(), Desc: "播报全球早间要闻"}
	p2NoDesc := Programme{Title: "午间视点", Start: now.Add(-time.Hour).Unix(), End: now.Unix(), Desc: ""}
	p2WithDesc := Programme{Title: "午间视点", Start: now.Add(-time.Hour).Unix(), End: now.Unix(), Desc: "深度解析财经动态"}

	cfg, _ := s.Store.Snapshot()
	// Test that merging duplicate programme updates empty desc with richer desc
	if err := s.saveEPG(map[string][]Programme{"op_test": {p1, p2NoDesc, p2WithDesc}}, cfg.EPG); err != nil {
		t.Fatal(err)
	}

	progs := s.Programmes(c, now.Add(-3*time.Hour).Unix(), now.Add(time.Hour).Unix())
	if len(progs) != 2 {
		t.Fatalf("expected 2 programmes, got %d", len(progs))
	}
	if progs[0].Desc != "播报全球早间要闻" {
		t.Fatalf("unexpected p1 desc: %s", progs[0].Desc)
	}
	if progs[1].Desc != "深度解析财经动态" {
		t.Fatalf("unexpected p2 desc after merge: %s", progs[1].Desc)
	}

	w := httptest.NewRecorder()
	s.ServeEPG(w, httptest.NewRequest("GET", "/api/epg", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	xmlStr := w.Body.String()
	if !strings.Contains(xmlStr, `<desc lang="zh">播报全球早间要闻</desc>`) {
		t.Fatalf("XML missing p1 desc: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<desc lang="zh">深度解析财经动态</desc>`) {
		t.Fatalf("XML missing p2 desc: %s", xmlStr)
	}
}
