package panel

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (s *Service) LiveURL(ctx context.Context, c Channel) (string, error) {
	cfg, _ := s.Store.Snapshot()
	if cfg.Forward.PlayMode == "http" && c.OperatorID != "" {
		if s.ResolveHTTP == nil || !cfg.IPTV.Enabled {
			return "", errors.New("请配置并启用 IPTV 认证")
		}
		return s.ResolveHTTP(ctx, cfg, c.OperatorID, time.Time{}, time.Time{})
	}
	return c.PlayURL, nil
}

func unicastURL(value string) bool {
	if strings.ContainsAny(value, "\r\n") {
		return false
	}
	if httpURL(value) {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "rtsp" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

func playbackLink(id, mode string) string {
	return "/api/play?id=" + url.QueryEscape(id) + "&mode=" + mode
}

func parseSingleTime(s string, defaultLoc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("时间参数为空")
	}

	allDigits := true
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			allDigits = false
			break
		}
	}

	if allDigits {
		switch len(s) {
		case 14: // 20060102150405 (China Standard Time format, popular in RTSP/Televizo)
			return time.ParseInLocation("20060102150405", s, defaultLoc)
		case 12: // 200601021504
			return time.ParseInLocation("200601021504", s, defaultLoc)
		case 13: // Millisecond epoch timestamp
			ms, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.UnixMilli(ms), nil
		case 10: // Unix timestamp in seconds (TiviMate {utc} & Televizo {utc}/${start})
			sec, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(sec, 0), nil
		default:
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			if val > 100000000000 {
				return time.UnixMilli(val), nil
			}
			return time.Unix(val, 0), nil
		}
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, defaultLoc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("无法解析的时间格式: " + s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parsePlaybackTimes(q url.Values, catchupDays int) (start, end time.Time, isReplay bool, err error) {
	cstZone := time.FixedZone("CST", 8*3600)
	now := time.Now()

	// 1. Playseek parameter (RTSP playseek query e.g. 20260921120000-20260921130000 or 1726915200-1726918800)
	if playseek := strings.TrimSpace(q.Get("playseek")); playseek != "" {
		isReplay = true
		parts := strings.Split(playseek, "-")
		if len(parts) >= 1 && parts[0] != "" {
			s, err := parseSingleTime(parts[0], cstZone)
			if err != nil {
				return time.Time{}, time.Time{}, true, errors.New("无效的 playseek 开始时间: " + err.Error())
			}
			start = s
		}
		if len(parts) >= 2 && parts[1] != "" {
			e, err := parseSingleTime(parts[1], cstZone)
			if err != nil {
				return time.Time{}, time.Time{}, true, errors.New("无效的 playseek 结束时间: " + err.Error())
			}
			end = e
		}
	}

	// 2. Start time aliases: start, utc, begin, time, timestamp
	startRaw := firstNonEmpty(q.Get("start"), q.Get("utc"), q.Get("begin"), q.Get("time"), q.Get("timestamp"))
	if startRaw != "" && start.IsZero() {
		isReplay = true
		s, err := parseSingleTime(startRaw, cstZone)
		if err != nil {
			return time.Time{}, time.Time{}, true, errors.New("无效的回看开始时间: " + err.Error())
		}
		start = s
	}

	// 3. End time aliases: end, lutc, utcend, lutcend, stop
	endRaw := firstNonEmpty(q.Get("end"), q.Get("lutc"), q.Get("utcend"), q.Get("lutcend"), q.Get("stop"))
	if endRaw != "" && end.IsZero() {
		isReplay = true
		e, err := parseSingleTime(endRaw, cstZone)
		if err != nil {
			return time.Time{}, time.Time{}, true, errors.New("无效的回看结束时间: " + err.Error())
		}
		end = e
	}

	// 4. Duration aliases: duration, dur
	durRaw := firstNonEmpty(q.Get("duration"), q.Get("dur"))
	if durRaw != "" && !start.IsZero() && end.IsZero() {
		isReplay = true
		if durSec, err := strconv.ParseInt(durRaw, 10, 64); err == nil && durSec > 0 {
			end = start.Add(time.Duration(durSec) * time.Second)
		}
	}

	if !isReplay {
		return time.Time{}, time.Time{}, false, nil
	}

	if start.IsZero() {
		return time.Time{}, time.Time{}, true, errors.New("缺少回看开始时间")
	}

	if end.IsZero() {
		end = start.Add(2 * time.Hour)
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, true, errors.New("回看结束时间不能早于开始时间")
	}

	if catchupDays <= 0 {
		return time.Time{}, time.Time{}, true, errors.New("该频道未启用回看")
	}

	minAllowed := now.Add(-time.Duration(catchupDays+1)*24*time.Hour - 2*time.Hour)
	if start.Before(minAllowed) {
		return time.Time{}, time.Time{}, true, errors.New("回看时间超出频道可回看天数范围")
	}

	if start.After(now.Add(5 * time.Minute)) {
		return time.Time{}, time.Time{}, true, errors.New("回看开始时间不能为未来时间")
	}

	// Timeshift support (e.g. playing ongoing live broadcast from start):
	// If program ends in the future, clamp replay end to near current time to ensure RTSP/HTTP compatibility
	if end.After(now.Add(5 * time.Minute)) {
		end = now.Add(5 * time.Minute)
	}

	return start, end, true, nil
}

func (s *Service) ServePlay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	q := r.URL.Query()
	ref := firstNonEmpty(q.Get("id"), q.Get("channel"), q.Get("channel_id"))
	c, err := s.Channel(ref)
	if err != nil || !c.Enabled {
		failure(w, 404, errors.New("频道不存在或已停用"))
		return
	}
	cfg, _ := s.Store.Snapshot()
	mode := q.Get("mode")
	if mode == "" {
		if c.OperatorID != "" && cfg.Forward.PlayMode != "unicast" {
			mode = "http"
		} else {
			mode = "unicast"
		}
	}
	if mode != "http" && mode != "unicast" {
		failure(w, 400, errors.New("播放方式只支持 http 或 unicast"))
		return
	}

	start, end, isReplay, err := parsePlaybackTimes(q, c.CatchupDays)
	if err != nil {
		failure(w, 400, err)
		return
	}

	var target string
	if mode == "http" && c.OperatorID != "" {
		if s.ResolveHTTP == nil || !cfg.IPTV.Enabled {
			failure(w, 503, errors.New("请配置并启用 IPTV 认证"))
			return
		}
		target, err = s.ResolveHTTP(r.Context(), cfg, c.OperatorID, start, end)
	} else {
		source := c.UnicastURL
		if source == "" && unicastURL(c.URL) {
			source = c.URL
		}
		if source == "" {
			failure(w, 404, errors.New("该频道没有单播地址"))
			return
		}
		target = PlaybackURL(cfg.Forward, source)
		if isReplay && !start.IsZero() {
			u, e := url.Parse(target)
			if e != nil {
				failure(w, 400, e)
				return
			}
			// Shanghai Telecom RTSP replay uses China Standard Time, regardless of host timezone.
			zone := time.FixedZone("CST", 8*3600)
			v := u.Query()
			v.Set("playseek", start.In(zone).Format("20060102150405")+"-"+end.In(zone).Format("20060102150405"))
			u.RawQuery = v.Encode()
			target = u.String()
		}
	}
	if err != nil {
		failure(w, 502, errors.New("获取运营商 HTTP 播放地址失败: "+err.Error()))
		return
	}
	if !unicastURL(target) {
		failure(w, 502, errors.New("无效的播放地址"))
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Service) playlistPlayback(channels []Channel, mode, base string) ([]Channel, error) {
	cfg, _ := s.Store.Snapshot()
	if mode == "" {
		mode = cfg.Forward.PlayMode
	}
	if mode == "" {
		mode = "multicast"
	}
	if mode != "multicast" && mode != "unicast" && mode != "http" {
		return nil, errors.New("mode 只支持 multicast、unicast、http")
	}
	idCounts := map[string]int{}
	for _, c := range channels {
		if c.ID != "" {
			idCounts[c.ID]++
		}
	}
	list := make([]Channel, 0, len(channels))
	for _, c := range channels {
		if strings.HasPrefix(c.Logo, "/") {
			c.Logo = strings.TrimRight(base, "/") + c.Logo
		}
		ref := c.ID
		if IsInvalidChannelID(ref) || ref == "" || idCounts[c.ID] > 1 {
			if c.OperatorID != "" {
				ref = c.OperatorID
			} else if c.Key != "" {
				ref = c.Key
			}
		}
		switch mode {
		case "unicast":
			source := c.UnicastURL
			if source == "" && unicastURL(c.URL) {
				source = c.URL
			}
			if source == "" {
				continue
			}
			c.PlayURL = PlaybackURL(cfg.Forward, source)
		case "http":
			if c.OperatorID != "" {
				c.PlayURL = base + playbackLink(ref, "http")
			} else if httpURL(c.UnicastURL) {
				c.PlayURL = c.UnicastURL
			} else if httpURL(c.URL) {
				c.PlayURL = c.URL
			} else {
				continue
			}
		}
		if c.CatchupDays > 0 && (c.UnicastURL != "" || c.OperatorID != "") {
			replayMode := "unicast"
			if mode != "unicast" && c.OperatorID != "" {
				replayMode = "http"
			}
			tpl := strings.TrimSpace(cfg.Forward.CatchupTemplate)
			if tpl == "" {
				tpl = DefaultCatchupTemplate
			}
			if strings.HasPrefix(tpl, "?") {
				tpl = "&" + strings.TrimPrefix(tpl, "?")
			} else if !strings.HasPrefix(tpl, "&") {
				tpl = "&" + tpl
			}
			c.CatchupSource = base + playbackLink(ref, replayMode) + tpl
		}
		list = append(list, c)
	}
	return list, nil
}
