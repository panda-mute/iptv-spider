package panel

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type EPGConfig struct {
	Enabled       bool `json:"enabled"`
	IntervalHours int  `json:"interval_hours"`
	PastDays      int  `json:"past_days"`
	FutureDays    int  `json:"future_days"`
}
type Programme struct {
	Title string `json:"title"`
	Start int64  `json:"start"` // Unix seconds
	End   int64  `json:"end"`
	Desc  string `json:"desc,omitempty"`
}
type epgData struct {
	UpdatedAt time.Time              `json:"updated_at"`
	Channels  map[string][]Programme `json:"channels"` // Stable operator IDs.
}
type EPGStatus struct {
	Job
	Backend    string    `json:"backend"`
	UpdatedAt  time.Time `json:"updated_at"`
	Channels   int       `json:"channels"`
	Programmes int       `json:"programmes"`
}

func (s *Service) OpenEPG() error {
	load := s.LoadEPGData
	if load == nil {
		load = func() ([]byte, error) { return os.ReadFile(filepath.Join(filepath.Dir(s.Store.path), "epg.json")) }
	}
	b, err := load()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var data epgData
	if err = json.Unmarshal(b, &data); err != nil {
		return err
	}
	s.epgMu.Lock()
	s.epgData = data
	s.epgMu.Unlock()
	return nil
}
func (s *Service) EPGStatus() EPGStatus {
	s.mu.Lock()
	job := s.epg
	s.mu.Unlock()
	s.epgMu.RLock()
	defer s.epgMu.RUnlock()
	out := EPGStatus{Job: job, UpdatedAt: s.epgData.UpdatedAt, Channels: len(s.epgData.Channels)}
	out.Backend = s.EPGBackend
	if out.Backend == "" {
		out.Backend = "file"
	}
	for _, p := range s.epgData.Channels {
		out.Programmes += len(p)
	}
	return out
}
func (s *Service) MaybeRefreshEPG() {
	cfg, _ := s.Store.Snapshot()
	if !cfg.EPG.Enabled || !cfg.IPTV.Enabled {
		return
	}
	status := s.EPGStatus()
	if status.State == "running" {
		return
	}
	last := status.UpdatedAt
	if status.StartedAt.After(last) {
		last = status.StartedAt
	}
	if last.IsZero() || time.Since(last) >= time.Duration(cfg.EPG.IntervalHours)*time.Hour {
		_ = s.RefreshEPG()
	}
}
func (s *Service) StopEPG() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.epgCancel != nil {
		s.epgCancel()
	}
}
func (s *Service) RefreshEPG() error {
	cfg, _ := s.Store.Snapshot()
	if !cfg.IPTV.Enabled {
		return errors.New("请先配置并启用 IPTV 认证")
	}
	if s.FetchEPG == nil {
		return errors.New("节目单同步服务尚未初始化")
	}
	s.mu.Lock()
	if s.epg.State == "running" || s.refresh.State == "running" {
		s.mu.Unlock()
		return errors.New("频道或节目单同步正在运行，请完成后再试")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	s.epgCancel = cancel
	s.epg = Job{State: "running", StartedAt: time.Now()}
	s.mu.Unlock()
	go func() {
		defer cancel()
		var err error
		defer func() {
			if recover() != nil {
				err = errors.New("节目单同步异常，已保留现有数据")
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			s.epg.State = "completed"
			s.epgCancel = nil
			if err != nil {
				s.epg.State = "failed"
				s.epg.Error = err.Error()
			}
		}()
		data, fetchErr := s.FetchEPG(ctx, cfg, func(done, total, count int) {
			s.mu.Lock()
			s.epg.Done = done
			s.epg.Total = total
			s.epg.Found = count
			s.mu.Unlock()
		})
		if len(data) > 0 {
			err = s.saveEPG(data, cfg.EPG)
		} else if fetchErr == nil {
			err = errors.New("运营商未返回节目单，已保留现有数据")
		}
		if err == nil {
			err = fetchErr
		}
	}()
	return nil
}
func (s *Service) saveEPG(channels map[string][]Programme, cfg EPGConfig) error {
	s.epgMu.Lock()
	defer s.epgMu.Unlock()
	now := time.Now()
	min := now.Add(-time.Duration(cfg.PastDays+1) * 24 * time.Hour).Unix()
	max := now.Add(time.Duration(cfg.FutureDays+1) * 24 * time.Hour).Unix()
	next := epgData{UpdatedAt: now, Channels: map[string][]Programme{}}
	for id, p := range s.epgData.Channels {
		next.Channels[id] = p
	}
	for id, p := range channels {
		next.Channels[id] = p
	}
	for id, programmes := range next.Channels {
		cleaned := make([]Programme, 0, len(programmes))
		seen := map[string]int{}
		for _, p := range programmes {
			if p.Title == "" || p.End <= p.Start || p.End <= min || p.Start >= max {
				continue
			}
			key := fmt.Sprint(p.Start, "/", p.End, "/", p.Title)
			if idx, ok := seen[key]; ok {
				if cleaned[idx].Desc == "" && p.Desc != "" {
					cleaned[idx].Desc = p.Desc
				}
				continue
			}
			seen[key] = len(cleaned)
			cleaned = append(cleaned, p)
		}
		sort.Slice(cleaned, func(i, j int) bool { return cleaned[i].Start < cleaned[j].Start })
		if len(cleaned) == 0 {
			delete(next.Channels, id)
		} else {
			next.Channels[id] = cleaned
		}
	}
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if s.SaveEPGData != nil {
		if err := s.SaveEPGData(b); err != nil {
			return err
		}
		s.epgData = next
		return nil
	}
	path := filepath.Join(filepath.Dir(s.Store.path), "epg.json")
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".epg-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	s.epgData = next
	return nil
}
func (s *Service) Programmes(c Channel, start, end int64) []Programme {
	out := []Programme{}
	s.epgMu.RLock()
	defer s.epgMu.RUnlock()
	list := s.epgData.Channels[c.ID]
	if len(list) == 0 && c.OperatorID != "" {
		list = s.epgData.Channels[c.OperatorID]
	}
	if len(list) == 0 && c.OriginalID != "" {
		list = s.epgData.Channels[c.OriginalID]
	}
	if len(list) == 0 && c.Key != "" {
		list = s.epgData.Channels[c.Key]
	}
	can := CanonicalName(c.Name)
	if len(list) == 0 && can != "" {
		list = s.epgData.Channels[can]
	}
	if len(list) == 0 && s.Store != nil && IsValidChannelID(c.ID) {
		for _, imp := range s.Store.Imported() {
			if (IsValidChannelID(imp.ID) && imp.ID == c.ID) || (c.OriginalID != "" && imp.ID == c.OriginalID) || (can != "" && CanonicalName(imp.Name) == can) {
				if imp.OperatorID != "" && len(s.epgData.Channels[imp.OperatorID]) > 0 {
					list = s.epgData.Channels[imp.OperatorID]
					break
				}
				if len(s.epgData.Channels[imp.ID]) > 0 {
					list = s.epgData.Channels[imp.ID]
					break
				}
				if imp.OriginalID != "" && len(s.epgData.Channels[imp.OriginalID]) > 0 {
					list = s.epgData.Channels[imp.OriginalID]
					break
				}
			}
		}
	}
	for _, p := range list {
		if p.End > start && p.Start < end {
			out = append(out, p)
		}
	}
	return out
}

var chinaTime = time.FixedZone("CST", 8*3600)

func (s *Service) ServeEPGPrograms(w http.ResponseWriter, r *http.Request) {
	c, err := s.Channel(r.URL.Query().Get("id"))
	if err != nil {
		failure(w, 404, err)
		return
	}
	day := time.Now().In(chinaTime).Format("2006-01-02")
	if v := r.URL.Query().Get("date"); v != "" {
		day = v
	}
	start, err := time.ParseInLocation("2006-01-02", day, chinaTime)
	if err != nil {
		failure(w, 400, errors.New("日期格式应为 YYYY-MM-DD"))
		return
	}
	progs := s.Programmes(c, start.Unix(), start.AddDate(0, 0, 1).Unix())
	jsonResponse(w, 200, map[string]any{"programmes": progs, "operator": c.OperatorID != "" || len(progs) > 0, "date": day})
}
func (s *Service) ServeEPG(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.Store.Snapshot()
	past := cfg.EPG.PastDays
	if raw := r.URL.Query().Get("daysAgo"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 14 {
			failure(w, 400, errors.New("daysAgo 为 0–14"))
			return
		}
		past = v
	}
	status := s.EPGStatus()
	if status.UpdatedAt.IsZero() {
		failure(w, 503, errors.New("节目单尚未同步，请在面板 EPG 中点击立即同步"))
		return
	}
	channels, err := s.Channels()
	if err != nil {
		failure(w, 503, err)
		return
	}
	type icon struct {
		Src string `xml:"src,attr"`
	}
	type xmlChannel struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"display-name"`
		Icon *icon  `xml:"icon,omitempty"`
	}
	type xmlDesc struct {
		Lang  string `xml:"lang,attr"`
		Value string `xml:",chardata"`
	}
	type xmlProgramme struct {
		Channel string   `xml:"channel,attr"`
		Start   string   `xml:"start,attr"`
		Stop    string   `xml:"stop,attr"`
		Title   string   `xml:"title"`
		Desc    *xmlDesc `xml:"desc,omitempty"`
	}
	tv := struct {
		XMLName    xml.Name       `xml:"tv"`
		Generator  string         `xml:"generator-info-name,attr"`
		Channels   []xmlChannel   `xml:"channel"`
		Programmes []xmlProgramme `xml:"programme"`
	}{Generator: "iptv-spider"}
	now := time.Now().In(chinaTime)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, chinaTime)
	start := midnight.AddDate(0, 0, -past).Unix()
	end := midnight.AddDate(0, 0, cfg.EPG.FutureDays+1).Unix()
	seenXMLIDs := map[string]bool{}
	for _, c := range channels {
		if !c.Enabled {
			continue
		}
		if IsInvalidChannelID(c.ID) {
			continue
		}
		if seenXMLIDs[c.ID] {
			continue
		}
		seenXMLIDs[c.ID] = true
		progs := s.Programmes(c, start, end)
		if len(progs) == 0 && c.OperatorID == "" {
			continue
		}
		ch := xmlChannel{ID: c.ID, Name: c.Name}
		if c.Logo != "" {
			logoURL := c.Logo
			if strings.HasPrefix(logoURL, "/") {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				logoURL = scheme + "://" + r.Host + logoURL
			}
			ch.Icon = &icon{logoURL}
		}
		tv.Channels = append(tv.Channels, ch)
		for _, p := range progs {
			var desc *xmlDesc
			if p.Desc != "" {
				desc = &xmlDesc{Lang: "zh", Value: p.Desc}
			} else {
				desc = &xmlDesc{Lang: "zh"}
			}
			tv.Programmes = append(tv.Programmes, xmlProgramme{
				Channel: c.ID,
				Start:   time.Unix(p.Start, 0).In(chinaTime).Format("20060102150405 -0700"),
				Stop:    time.Unix(p.End, 0).In(chinaTime).Format("20060102150405 -0700"),
				Title:   p.Title,
				Desc:    desc,
			})
		}
	}
	b, err := xml.Marshal(tv)
	if err != nil {
		failure(w, 500, err)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(xml.Header))
	w.Write(b)
}
