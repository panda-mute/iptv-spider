package panel

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:embed web/*
var web embed.FS

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status int, err error) {
	jsonResponse(w, status, map[string]string{"error": err.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("请求需要 application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("请求必须为单个 JSON 对象")
	}
	return nil
}

func (s *Service) Handler(token string) http.Handler {
	mux := http.NewServeMux()
	assets, _ := fs.Sub(web, "web")
	mux.HandleFunc("GET /logos/{file...}", s.ServeLocalLogo)
	mux.HandleFunc("GET /api/panel/logos/local/{file...}", s.ServeLocalLogo)
	mux.Handle("/", http.FileServer(http.FS(assets)))
	api := http.NewServeMux()
	api.HandleFunc("GET /api/panel/logos/status", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, s.LogoSourcesStatus()) })
	api.HandleFunc("POST /api/panel/logos/download", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URL  string `json:"url"`
			Name string `json:"name"`
		}
		if err := decode(w, r, &body); err != nil {
			failure(w, 400, err)
			return
		}
		localURL, err := s.DownloadLogo(r.Context(), body.URL, body.Name)
		if err != nil {
			failure(w, 422, err)
			return
		}
		jsonResponse(w, 200, map[string]string{"url": localURL})
	})
	api.HandleFunc("POST /api/panel/logos/upload", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			failure(w, 400, fmt.Errorf("解析上传文件失败: %v", err))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			failure(w, 400, errors.New("请选择要上传的台标图片 (file)"))
			return
		}
		defer file.Close()

		name := r.FormValue("name")
		localURL, err := s.SaveUploadedLogo(name, header.Filename, file)
		if err != nil {
			failure(w, 422, err)
			return
		}
		jsonResponse(w, 200, map[string]string{"url": localURL})
	})
	api.HandleFunc("POST /api/panel/logos/refresh", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		jsonResponse(w, 200, s.RefreshLogoSources(ctx))
	})
	api.HandleFunc("GET /api/panel/epg/status", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, s.EPGStatus()) })
	api.HandleFunc("GET /api/panel/epg/programmes", s.ServeEPGPrograms)
	api.HandleFunc("POST /api/panel/epg/refresh", func(w http.ResponseWriter, r *http.Request) {
		if err := s.RefreshEPG(); err != nil {
			failure(w, 409, err)
			return
		}
		jsonResponse(w, 202, s.EPGStatus())
	})
	api.HandleFunc("POST /api/panel/logos/match", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name   string `json:"name"`
			Group  string `json:"group"`
			Region string `json:"region"`
		}
		if err := decode(w, r, &body); err != nil {
			failure(w, 400, err)
			return
		}
		result, err := s.MatchLogoRegion(r.Context(), body.Name, body.Group, body.Region)
		if err != nil {
			failure(w, 422, err)
			return
		}
		jsonResponse(w, 200, result)
	})
	api.HandleFunc("GET /api/panel/settings", func(w http.ResponseWriter, r *http.Request) {
		v, _ := s.Store.Snapshot()
		hasKey := v.AI.APIKey != ""
		v.AI.APIKey = ""
		jsonResponse(w, 200, map[string]any{"settings": v, "has_api_key": hasKey})
	})
	api.HandleFunc("PUT /api/panel/settings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Settings    Settings `json:"settings"`
			ClearAPIKey bool     `json:"clear_api_key"`
		}
		if err := decode(w, r, &body); err != nil {
			failure(w, 400, err)
			return
		}
		old, _ := s.Store.Snapshot()
		if body.ClearAPIKey {
			body.Settings.AI.APIKey = ""
		} else if body.Settings.AI.APIKey == "" {
			body.Settings.AI.APIKey = old.AI.APIKey
		}
		if err := s.Store.SaveSettings(body.Settings); err != nil {
			failure(w, 400, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"saved": true})
	})
	api.HandleFunc("GET /api/panel/channels/match", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		cands := s.MatchChannelCandidates(name, 10)
		if len(cands) == 0 {
			jsonResponse(w, 200, map[string]any{"matched": false, "candidates": []ChannelMatchCandidate{}})
			return
		}
		top := cands[0]
		jsonResponse(w, 200, map[string]any{
			"matched":     true,
			"id":          top.ID,
			"name":        top.Name,
			"operator_id": top.OperatorID,
			"group":       top.Group,
			"is_saved":    top.IsSaved,
			"candidates":  cands,
		})
	})
	api.HandleFunc("GET /api/panel/mappings", func(w http.ResponseWriter, r *http.Request) {
		if s.Store == nil {
			jsonResponse(w, 200, []ChannelMapping{})
			return
		}
		jsonResponse(w, 200, s.Store.Mappings())
	})
	api.HandleFunc("POST /api/panel/mappings", func(w http.ResponseWriter, r *http.Request) {
		if s.Store == nil {
			failure(w, 503, errors.New("存储服务未就绪"))
			return
		}
		var m ChannelMapping
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			failure(w, 400, errors.New("无效的映射数据"))
			return
		}
		if err := s.Store.SaveMapping(m); err != nil {
			failure(w, 400, err)
			return
		}
		jsonResponse(w, 200, map[string]any{"saved": true, "mapping": m})
	})
	api.HandleFunc("DELETE /api/panel/mappings/{keyword}", func(w http.ResponseWriter, r *http.Request) {
		if s.Store == nil {
			failure(w, 503, errors.New("存储服务未就绪"))
			return
		}
		kw := r.PathValue("keyword")
		if err := s.Store.DeleteMapping(kw); err != nil {
			failure(w, 400, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"deleted": true})
	})
	api.HandleFunc("GET /api/panel/channels", func(w http.ResponseWriter, r *http.Request) {
		channels, err := s.Channels()
		if err != nil {
			failure(w, 503, err)
			return
		}
		jsonResponse(w, 200, channels)
	})
	api.HandleFunc("PUT /api/panel/channels/{id}", func(w http.ResponseWriter, r *http.Request) {
		var c Channel
		if err := decode(w, r, &c); err != nil {
			failure(w, 400, err)
			return
		}
		if strings.TrimSpace(c.ID) == "" {
			c.ID = InvalidChannelID
		}
		oldID := r.PathValue("id")
		c.LogoAutomatic = false
		reqOpID := c.OperatorID
		old, err := s.Channel(oldID)
		c.OriginalID = ""
		c.CatchupSource = ""
		if err == nil {
			if reqOpID != "" {
				c.OperatorID = reqOpID
			} else {
				c.OperatorID = old.OperatorID
			}
			c.PlaybackCustom = old.PlaybackCustom || c.UnicastURL != old.UnicastURL || c.CatchupDays != old.CatchupDays || c.OperatorID != old.OperatorID
			if old.Source == "iptv" {
				c.OriginalID = old.OriginalID
				if c.OriginalID == "" && old.Key != "" && old.Key != c.ID {
					c.OriginalID = old.Key
				}
			}
			if c.Key == "" {
				c.Key = old.Key
			}
			c.Source = old.Source
			c.Suggestion = old.Suggestion
			if c.Resolution == "" {
				c.Resolution = old.Resolution
			}
		} else {
			list, _ := s.Channels()
			for _, ch := range list {
				if ch.Source == "iptv" && ch.OriginalID != "" && ch.OriginalID == oldID {
					failure(w, 409, errors.New("频道原 ID 已修改，请刷新后重试"))
					return
				}
			}
			if oldID != c.ID && oldID != c.Key {
				failure(w, 404, errors.New("频道不存在或已在其他窗口修改，请刷新后重试"))
				return
			}
			c.OperatorID = reqOpID
			c.PlaybackCustom = true
			c.Source = "manual"
			c.OriginalID = ""
			c.Suggestion = nil
			if c.Key == "" {
				c.Key = oldID
			}
		}
		if c.Key == "" {
			c.EnsureKey()
		}
		if err = s.Store.EditChannel(oldID, c); err != nil {
			failure(w, 400, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"saved": true})
	})
	api.HandleFunc("GET /api/panel/status", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, s.Status()) })
	api.HandleFunc("POST /api/panel/scan/start", func(w http.ResponseWriter, r *http.Request) {
		if err := s.StartScan(); err != nil {
			failure(w, 409, err)
			return
		}
		jsonResponse(w, 202, s.Status())
	})
	api.HandleFunc("POST /api/panel/scan/stop", func(w http.ResponseWriter, r *http.Request) { s.StopScan(); jsonResponse(w, 202, s.Status()) })
	api.HandleFunc("POST /api/panel/refresh", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Refresh(r.URL.Query().Get("epg") == "true"); err != nil {
			failure(w, 409, err)
			return
		}
		jsonResponse(w, 202, s.Status())
	})
	api.HandleFunc("POST /api/panel/probe/start", func(w http.ResponseWriter, r *http.Request) {
		onlyMissing := r.URL.Query().Get("only_missing") == "true"
		if err := s.StartProbe(r.Context(), onlyMissing); err != nil {
			failure(w, 409, err)
			return
		}
		jsonResponse(w, 202, s.ProbeStatus())
	})
	api.HandleFunc("POST /api/panel/probe/stop", func(w http.ResponseWriter, r *http.Request) {
		s.StopProbe()
		jsonResponse(w, 200, s.ProbeStatus())
	})
	api.HandleFunc("GET /api/panel/probe/status", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, s.ProbeStatus())
	})
	api.HandleFunc("POST /api/panel/channels/{id}/probe", func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Channel(r.PathValue("id"))
		if err != nil {
			failure(w, 404, err)
			return
		}
		uri, err := s.LiveURL(r.Context(), c)
		if err != nil {
			failure(w, 422, err)
			return
		}
		res, err := ProbeResolutionFromURL(r.Context(), uri)
		if err != nil || res == "" {
			failure(w, 422, fmt.Errorf("未能探测到有效视频分辨率: %v", err))
			return
		}
		key := c.EnsureKey()
		_ = s.Store.update(func(d *state) error {
			if latest, ok := d.Channels[key]; ok {
				latest.Resolution = res
				d.Channels[key] = latest
			}
			return nil
		})
		jsonResponse(w, 200, map[string]any{"resolution": res, "key": key})
	})
	api.HandleFunc("POST /api/panel/channels/{id}/identify", func(w http.ResponseWriter, r *http.Request) {
		result, err := s.Identify(r.Context(), r.PathValue("id"))
		if err != nil {
			failure(w, 422, err)
			return
		}
		jsonResponse(w, 200, result)
	})
	api.HandleFunc("GET /api/panel/channels/{id}/snapshot", func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Channel(r.PathValue("id"))
		if err != nil {
			failure(w, 404, err)
			return
		}
		uri, err := s.LiveURL(r.Context(), c)
		if err != nil {
			failure(w, 422, err)
			return
		}
		b, err := Capture(r.Context(), uri)
		if err != nil {
			failure(w, 422, err)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(b)
	})
	mux.Handle("/api/panel/", AdminOnly(token, api))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob: http: https:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		mux.ServeHTTP(w, r)
	})
}

func AdminOnly(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				failure(w, 401, errors.New("请输入正确的面板访问令牌"))
				return
			}
		} else {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				failure(w, 403, errors.New("远程管理请先设置 IPTV_PANEL_TOKEN 环境变量"))
				return
			}
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
				failure(w, 403, errors.New("不允许跨站管理请求"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
