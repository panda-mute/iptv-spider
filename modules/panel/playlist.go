package panel

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s *Service) M3U(channels []Channel) string {
	epgURL := s.EPGURL
	if epgURL == "" {
		epgURL = "/api/epg"
	}
	return s.M3UWithEPG(channels, epgURL)
}

func (s *Service) M3UWithEPG(channels []Channel, epgURL string) string {
	text := Playlist(channels, "m3u")
	headerAttrs := ""
	if epgURL != "" {
		cleanURL := cleanM3U(epgURL)
		headerAttrs += fmt.Sprintf(` x-tvg-url="%s" url-tvg="%s"`, cleanURL, cleanURL)
	}
	headerAttrs += ` catchup="default"`
	return strings.Replace(text, "#EXTM3U\n", "#EXTM3U"+headerAttrs+"\n", 1)
}

func (s *Service) Export(udpxy, scheme, xteve, all string) ([]Channel, error) {
	channels, err := s.Channels()
	if err != nil {
		return nil, err
	}
	settings, _ := s.Store.Snapshot()
	if udpxy != "" && !endpoint(udpxy) {
		return nil, errors.New("udpxy 参数必须为 host:port")
	}
	if scheme != "" && scheme != "rtp" && scheme != "udp" && scheme != "igmp" {
		return nil, errors.New("scheme 只支持 rtp、udp、igmp")
	}
	list := make([]Channel, 0, len(channels))
	for _, c := range channels {
		c.PlayURL = PlaybackURL(settings.Forward, c.URL)
		if !c.Enabled || (all != "true" && c.Group == "购物") {
			continue
		}
		u, e := url.Parse(c.URL)
		if e == nil && (u.Scheme == "igmp" || u.Scheme == "rtp" || u.Scheme == "udp") {
			switch {
			case xteve == "true":
				c.PlayURL = "udp://@" + u.Host
			case udpxy != "":
				f := settings.Forward
				f.Address = udpxy
				c.PlayURL = PlaybackURL(f, c.URL)
			case scheme != "":
				u.Scheme = scheme
				c.PlayURL = u.String()
			}
		}
		list = append(list, c)
	}
	return list, nil
}

func (s *Service) ServePlaylist(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	channels, err := s.Export(q.Get("udpxy"), q.Get("scheme"), q.Get("xteve"), q.Get("all"))
	if err != nil {
		failure(w, 503, err)
		return
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	channels, err = s.playlistPlayback(channels, q.Get("mode"), scheme+"://"+r.Host)
	if err != nil {
		failure(w, 400, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	switch q.Get("fmt") {
	case "json":
		jsonResponse(w, 200, channels)
	case "txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(Playlist(channels, "txt")))
	case "", "m3u":
		w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=iptv.m3u")
		epgURL := q.Get("epg")
		if epgURL == "" {
			epgURL = s.EPGURL
		}
		if epgURL == "" {
			host := r.Host
			if host == "" {
				host = "127.0.0.1:8888"
			}
			epgURL = scheme + "://" + host + "/api/epg"
		}
		text := s.M3UWithEPG(channels, epgURL)
		_, _ = w.Write([]byte(text))
	default:
		failure(w, 400, errors.New("fmt 只支持 m3u、txt、json"))
	}
}
