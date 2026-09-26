package panel

import (
	"context"
	"net/http"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Job struct {
	State     string    `json:"state"`
	Total     int       `json:"total"`
	Done      int       `json:"done"`
	Found     int       `json:"found"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
}

type Service struct {
	Store       *Store
	StreamClient *http.Client
	Load        func() ([]Channel, error)
	Fetch       func(Settings, bool) error
	ResolveHTTP func(context.Context, Settings, string, time.Time, time.Time) (string, error)
	FetchEPG    func(context.Context, Settings, func(int, int, int)) (map[string][]Programme, error)
	LoadEPGData func() ([]byte, error)
	SaveEPGData func([]byte) error
	EPGBackend  string
	epgMu       sync.RWMutex
	epgData     epgData
	epg         Job
	epgCancel   context.CancelFunc
	EPGURL      string
	LogosDir    string
	mu          sync.Mutex
	scan        Job
	refresh     Job
	cancel      context.CancelFunc
	scanDone    chan struct{}
	identify    chan struct{}
	probeMu     sync.Mutex
	probe       Job
	probeCancel context.CancelFunc
}

// Current is installed once during startup, before requests and cron jobs run.
var Current *Service

func NewService(store *Store) *Service {
	return &Service{Store: store, epg: Job{State: "idle"}, scan: Job{State: "idle"}, refresh: Job{State: "idle"}, probe: Job{State: "idle"}, identify: make(chan struct{}, 1)}
}

func (s *Service) Channels() ([]Channel, error) {
	settings, overrides := s.Store.Snapshot()
	byKey := map[string]Channel{}
	imported := s.Store.Imported()
	if s.Load != nil && len(imported) == 0 {
		base, err := s.Load()
		if err != nil {
			return nil, err
		}
		for _, c := range base {
			if s.Store != nil {
				s.Store.ApplyMapping(&c)
			}
			PreclassifyChannel(&c)
			key := c.EnsureKey()
			byKey[key] = c
		}
	}
	for _, c := range imported {
		key := c.EnsureKey()
		byKey[key] = c
	}
	for i, c := range overrides {
		key := c.EnsureKey()
		base, ok := byKey[key]
		if !ok && c.OriginalID != "" {
			base, ok = byKey[c.OriginalID]
		}
		if ok {
			c.OperatorID = base.OperatorID
			if !c.PlaybackCustom {
				c.UnicastURL = base.UnicastURL
				c.CatchupDays = base.CatchupDays
			}
		}
		overrides[i] = c
	}
	for _, c := range overrides {
		key := c.EnsureKey()
		byKey[key] = c
	}
	list := make([]Channel, 0, len(byKey))
	for _, c := range byKey {
		if c.ID == "" {
			c.ID = InvalidChannelID
		}
		if c.Source != "iptv" && c.OriginalID != "" {
			c.OriginalID = ""
		}
		if s.Store != nil {
			s.Store.ApplyMapping(&c)
		}
		if c.Logo == "" || c.LogoAutomatic {
			if local := s.FindLocalLogo(c.Name); local != "" {
				c.Logo = local
				c.LogoAutomatic = true
			} else if c.Logo == "" {
				c.Logo = configuredLogo(settings.Logos, c.Name, c.LogoRegion)
				c.LogoAutomatic = c.Logo != ""
			}
		}
		c.PlayURL = PlaybackURL(settings.Forward, c.URL)
		if settings.Forward.PlayMode == "unicast" && c.UnicastURL != "" {
			c.PlayURL = PlaybackURL(settings.Forward, c.UnicastURL)
		}
		list = append(list, c)
	}
	groupOrder := settings.GroupOrder
	if len(groupOrder) == 0 {
		groupOrder = DefaultGroupOrder()
	}
	SortChannels(list, groupOrder, settings.GroupChannelOrder)
	return list, nil
}

func (s *Service) Channel(ref string) (Channel, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Channel{}, errors.New("频道不存在")
	}
	list, err := s.Channels()
	if err != nil {
		return Channel{}, err
	}
	for _, c := range list {
		if c.OperatorID != "" && c.OperatorID == ref {
			return c, nil
		}
	}
	for _, c := range list {
		if c.Key != "" && c.Key == ref && (c.Source != "iptv" || c.OriginalID == "" || c.OriginalID != ref || c.ID == ref) {
			return c, nil
		}
	}
	for _, c := range list {
		if c.ID == ref {
			return c, nil
		}
	}
	for _, c := range list {
		if c.Name == ref {
			return c, nil
		}
	}
	for _, c := range list {
		if strings.EqualFold(c.ID, ref) {
			return c, nil
		}
	}
	for _, c := range list {
		if strings.EqualFold(c.Name, ref) {
			return c, nil
		}
	}
	canRef := CanonicalName(ref)
	if canRef != "" {
		for _, c := range list {
			if CanonicalName(c.Name) == canRef {
				return c, nil
			}
		}
	}
	return Channel{}, errors.New("频道不存在")
}

func (s *Service) Status() map[string]Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]Job{"scan": s.scan, "refresh": s.refresh}
}

func (s *Service) Refresh(epg bool) error {
	settings, _ := s.Store.Snapshot()
	if !settings.IPTV.Enabled {
		return errors.New("请先配置并启用 IPTV 认证")
	}
	if s.Fetch == nil {
		return errors.New("运营商同步服务尚未初始化")
	}
	s.mu.Lock()
	if s.refresh.State == "running" || s.epg.State == "running" {
		s.mu.Unlock()
		return errors.New("同步任务正在运行")
	}
	s.refresh = Job{State: "running", StartedAt: time.Now()}
	s.mu.Unlock()
	go func() {
		var err error
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("运营商认证失败，请检查专网与账号: %v", p)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			s.refresh.State = "completed"
			if err != nil {
				s.refresh.State = "failed"
				s.refresh.Error = err.Error()
			}
		}()
		err = s.Fetch(settings, epg)
	}()
	return nil
}

func cleanM3U(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", `"`, "'").Replace(s)
}

func Playlist(channels []Channel, format string) string {
	var b strings.Builder
	if format != "txt" {
		b.WriteString("#EXTM3U\n")
	}
	group := "\x00"
	for _, c := range channels {
		if !c.Enabled {
			continue
		}
		if format == "txt" {
			if c.Group != group {
				group = c.Group
				name := group
				if name == "" {
					name = "未分组"
				}
				fmt.Fprintf(&b, "%s,#genre#\n", cleanM3U(name))
			}
			fmt.Fprintf(&b, "%s,%s\n", cleanM3U(c.Name), c.PlayURL)
		} else {
			tvgID := c.ID
			if IsInvalidChannelID(tvgID) {
				tvgID = ""
			}
			chnoAttr := ""
			if tvgID != "" {
				if _, err := strconv.Atoi(tvgID); err == nil {
					chnoAttr = fmt.Sprintf(" tvg-chno=\"%s\"", cleanM3U(tvgID))
				}
			}
			fmt.Fprintf(&b, "#EXTINF:-1 tvg-id=\"%s\" tvg-name=\"%s\" tvg-logo=\"%s\"%s group-title=\"%s\"", cleanM3U(tvgID), cleanM3U(c.Name), cleanM3U(c.Logo), chnoAttr, cleanM3U(c.Group))
			if c.CatchupSource != "" && c.CatchupDays > 0 {
				fmt.Fprintf(&b, " catchup=\"default\" catchup-days=\"%d\" catchup-source=\"%s\" timeshift=\"%d\"", c.CatchupDays, cleanM3U(c.CatchupSource), c.CatchupDays)
			}
			fmt.Fprintf(&b, ",%s\n%s\n", cleanM3U(c.Name), c.PlayURL)
		}
	}
	return b.String()
}

func (s *Service) MatchChannelCandidates(query string, maxResults int) []ChannelMatchCandidate {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	var candidates []Channel
	if s.Store != nil {
		candidates = append(candidates, s.Store.Imported()...)
	}
	if channels, err := s.Channels(); err == nil {
		candidates = append(candidates, channels...)
	}

	rawMatches := MatchChannelCandidates(query, candidates, maxResults)

	if s.Store != nil {
		if m, found := s.Store.FindMapping(query); found {
			saved := ChannelMatchCandidate{
				ID:         m.TargetID,
				Name:       m.TargetName,
				OperatorID: m.OperatorID,
				Group:      m.Group,
				Score:      200,
				IsSaved:    true,
			}
			var filtered []ChannelMatchCandidate
			filtered = append(filtered, saved)
			for _, c := range rawMatches {
				if c.ID == m.TargetID && c.Name == m.TargetName {
					continue
				}
				filtered = append(filtered, c)
			}
			if len(filtered) > maxResults {
				filtered = filtered[:maxResults]
			}
			return filtered
		}
	}
	return rawMatches
}

func (s *Service) MatchChannel(query string) *Channel {
	cands := s.MatchChannelCandidates(query, 1)
	if len(cands) == 0 {
		return nil
	}
	top := cands[0]
	return &Channel{
		ID:         top.ID,
		Name:       top.Name,
		OperatorID: top.OperatorID,
		Group:      top.Group,
	}
}

func (s *Service) StartProbe(ctx context.Context, onlyMissing bool) error {
	s.probeMu.Lock()
	if s.probe.State == "running" {
		s.probeMu.Unlock()
		return errors.New("探测任务已在运行中")
	}

	channels, err := s.Channels()
	if err != nil {
		s.probeMu.Unlock()
		return err
	}

	var targets []Channel
	for _, c := range channels {
		if !c.Enabled {
			continue
		}
		if c.URL == "" && c.UnicastURL == "" {
			continue
		}
		if onlyMissing && c.Resolution != "" {
			continue
		}
		targets = append(targets, c)
	}

	if len(targets) == 0 {
		s.probeMu.Unlock()
		return errors.New("没有需要探测的有效频道")
	}

	pCtx, cancel := context.WithCancel(context.Background())
	s.probeCancel = cancel
	s.probe = Job{
		State:     "running",
		Total:     len(targets),
		Done:      0,
		Found:     0,
		StartedAt: time.Now(),
	}
	s.probeMu.Unlock()

	go func() {
		defer cancel()
		jobs := make(chan Channel, len(targets))
		for _, t := range targets {
			jobs <- t
		}
		close(jobs)

		workers := 3
		if workers > len(targets) {
			workers = len(targets)
		}
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for c := range jobs {
					select {
					case <-pCtx.Done():
						return
					default:
					}

					uri, err := s.LiveURL(pCtx, c)
					if err == nil && uri != "" {
						res, err := ProbeResolutionFromURL(pCtx, uri)
						if err == nil && res != "" {
							key := c.EnsureKey()
							_ = s.Store.update(func(d *state) error {
								if latest, ok := d.Channels[key]; ok {
									latest.Resolution = res
									d.Channels[key] = latest
								}
								return nil
							})
							s.probeMu.Lock()
							s.probe.Found++
							s.probeMu.Unlock()
						}
					}
					s.probeMu.Lock()
					s.probe.Done++
					s.probeMu.Unlock()
				}
			}()
		}
		wg.Wait()

		s.probeMu.Lock()
		if s.probe.State == "running" {
			s.probe.State = "completed"
		}
		s.probeMu.Unlock()
	}()

	return nil
}

func (s *Service) StopProbe() {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.probe.State == "running" {
		if s.probeCancel != nil {
			s.probeCancel()
		}
		s.probe.State = "stopped"
	}
}

func (s *Service) ProbeStatus() map[string]any {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	return map[string]any{
		"state":      s.probe.State,
		"total":      s.probe.Total,
		"done":       s.probe.Done,
		"found":      s.probe.Found,
		"error":      s.probe.Error,
		"started_at": s.probe.StartedAt,
	}
}
