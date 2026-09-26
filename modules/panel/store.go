package panel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type ChannelMapping struct {
	Keyword    string `json:"keyword"`
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
	OperatorID string `json:"operator_id,omitempty"`
	Group      string `json:"group,omitempty"`
	Logo       string `json:"logo,omitempty"`
}

type Suggestion struct {
	Name              string  `json:"name"`
	Group             string  `json:"group"`
	Confidence        float64 `json:"confidence"`
	Reason            string  `json:"reason"`
	MatchedID         string  `json:"matched_id,omitempty"`
	MatchedName       string  `json:"matched_name,omitempty"`
	MatchedOperatorID string  `json:"matched_operator_id,omitempty"`
	MatchedGroup      string  `json:"matched_group,omitempty"`
	Resolution        string  `json:"resolution,omitempty"`
}

// InvalidChannelID represents the default sentinel for unassigned or invalid digital channel IDs.
const InvalidChannelID = "0"

// IsInvalidChannelID returns true if id is considered invalid, empty, or not a positive integer channel number.
func IsInvalidChannelID(id string) bool {
	id = strings.TrimSpace(id)
	return id == "" || id == InvalidChannelID || id == "-1" || id == "none" || id == "unknown" || strings.HasPrefix(id, "scan-")
}

// IsValidChannelID returns true if id is a valid positive integer digital channel ID.
func IsValidChannelID(id string) bool {
	return !IsInvalidChannelID(id)
}

type Channel struct {
	Key            string      `json:"key,omitempty"`
	ID             string      `json:"id"`
	OriginalID     string      `json:"original_id,omitempty"`
	Name           string      `json:"name"`
	Group          string      `json:"group"`
	LogoRegion     string      `json:"logo_region,omitempty"`
	Logo           string      `json:"logo"`
	LogoAutomatic  bool        `json:"logo_automatic,omitempty"`
	URL            string      `json:"url"`
	Enabled        bool        `json:"enabled"`
	Source         string      `json:"source"`
	Suggestion     *Suggestion `json:"suggestion,omitempty"`
	PlayURL        string      `json:"play_url,omitempty"`
	UnicastURL     string      `json:"unicast_url,omitempty"`
	OperatorID     string      `json:"operator_id,omitempty"`
	CatchupDays    int         `json:"catchup_days"`
	PlaybackCustom bool        `json:"playback_custom,omitempty"`
	CatchupSource  string      `json:"catchup_source,omitempty"`
	Resolution     string      `json:"resolution,omitempty"`
}

func (c *Channel) EnsureKey() string {
	if c.Key != "" {
		return c.Key
	}
	if c.OperatorID != "" {
		c.Key = c.OperatorID
		return c.Key
	}
	if c.OriginalID != "" {
		c.Key = c.OriginalID
		return c.Key
	}
	if c.URL != "" && IsInvalidChannelID(c.ID) {
		c.Key = c.URL
		return c.Key
	}
	if c.ID != "" {
		c.Key = c.ID
		return c.Key
	}
	if c.URL != "" {
		c.Key = c.URL
		return c.Key
	}
	return ""
}

func (c *Channel) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		c.ID = InvalidChannelID
	}
	// LogoRegion is retained for backward compatibility, no longer enforced
	if c.CatchupDays < 0 || c.CatchupDays > 30 || (c.UnicastURL != "" && !unicastURL(c.UnicastURL)) {
		return errors.New("单播地址必须为 HTTP(S)/RTSP，回看天数为 0–30（0 关闭回看）")
	}
	if c.ID == "" || len(c.ID) > 160 || strings.ContainsAny(c.ID, "/\\?#\r\n") || c.Name == "" || len(c.Name) > 200 || len(c.Group) > 200 || strings.ContainsAny(c.Name+c.Group+c.Logo, "\r\n") {
		return errors.New("频道 ID、名称或分组无效")
	}
	if c.Logo != "" && !httpURL(c.Logo) && !isLocalLogo(c.Logo) {
		return errors.New("Logo 必须为 HTTP(S) 地址或本地相对路径")
	}
	u, err := url.Parse(c.URL)
	if err != nil || strings.ContainsAny(c.URL, "\r\n") {
		return errors.New("无效的频道 URL")
	}
	if u.Scheme == "rtp" || u.Scheme == "udp" || u.Scheme == "igmp" {
		ip, e := netip.ParseAddr(u.Hostname())
		p, pe := strconv.Atoi(u.Port())
		if e != nil || !ip.Is4() || !ip.IsMulticast() || pe != nil || p < 1 || p > 65535 || u.User != nil {
			return errors.New("频道需要有效的 IPv4 组播地址及端口")
		}
	} else if !unicastURL(c.URL) {
		return errors.New("频道协议只支持 HTTP(S)、RTSP、RTP 和 UDP")
	}
	return nil
}

type state struct {
	Version  int                       `json:"version"`
	Settings Settings                  `json:"settings"`
	Channels map[string]Channel        `json:"channels"`
	Imported []Channel                 `json:"imported"`
	Mappings map[string]ChannelMapping `json:"mappings"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data state
}

func OpenStore(path string, defaults Settings) (*Store, error) {
	s := &Store{path: path, data: state{Version: 2, Settings: defaults, Channels: map[string]Channel{}}}
	b, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("读取面板数据: %w", err)
		}
		if s.data.Version != 1 && s.data.Version != 2 {
			return nil, errors.New("不支持的面板数据版本")
		}
		if s.data.Version == 1 && s.data.Settings.Forward.Address == "" {
			s.data.Settings.Forward.Address = defaults.Forward.Address
		}
		if s.data.Settings.Forward.CatchupTemplate == "" {
			s.data.Settings.Forward.CatchupTemplate = defaults.Forward.CatchupTemplate
		}
		if len(s.data.Settings.Forward.FCCList) == 0 {
			s.data.Settings.Forward.FCCList = defaults.Forward.FCCList
			if s.data.Settings.Forward.FCC != "" && endpoint(s.data.Settings.Forward.FCC) {
				found := false
				for _, f := range s.data.Settings.Forward.FCCList {
					if f == s.data.Settings.Forward.FCC {
						found = true
						break
					}
				}
				if !found {
					s.data.Settings.Forward.FCCList = append(s.data.Settings.Forward.FCCList, s.data.Settings.Forward.FCC)
				}
			}
		}
		if err = s.data.Settings.Validate(); err != nil {
			return nil, err
		}
		if s.data.Channels == nil {
			s.data.Channels = map[string]Channel{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s.data.Mappings == nil {
		s.data.Mappings = map[string]ChannelMapping{}
		for _, m := range DefaultMappings() {
			s.data.Mappings[normalizeMappingKey(m.Keyword)] = m
		}
	}
	for k, ch := range s.data.Channels {
		m1 := s.ApplyMapping(&ch)
		m2 := PreclassifyChannel(&ch)
		if m1 || m2 {
			s.data.Channels[k] = ch
		}
	}
	for i := range s.data.Imported {
		s.ApplyMapping(&s.data.Imported[i])
		PreclassifyChannel(&s.data.Imported[i])
	}
	if s.data.Version == 1 {
		if err := s.update(func(d *state) error { d.Version = 2; return nil }); err != nil {
			return nil, fmt.Errorf("升级面板配置: %w", err)
		}
	}
	return s, nil
}

func (s *Store) Snapshot() (Settings, []Channel) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := make([]Channel, 0, len(s.data.Channels))
	for k, c := range s.data.Channels {
		if c.Key == "" {
			c.Key = k
		}
		if c.Suggestion != nil {
			v := *c.Suggestion
			c.Suggestion = &v
		}
		channels = append(channels, c)
	}
	settings := s.data.Settings
	return settings, channels
}

// Commit publishes state only after an atomic, permission-restricted file replacement.
func (s *Store) update(fn func(*state) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Channels = make(map[string]Channel, len(s.data.Channels))
	for k, v := range s.data.Channels {
		next.Channels[k] = v
	}
	next.Mappings = make(map[string]ChannelMapping, len(s.data.Mappings))
	for k, v := range s.data.Mappings {
		next.Mappings[k] = v
	}
	if err := fn(&next); err != nil {
		return err
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".panel-*.tmp")
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
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.data = next
	return nil
}

func (s *Store) SaveSettings(v Settings) error {
	if err := v.Validate(); err != nil {
		return err
	}
	return s.update(func(d *state) error {
		fccMap := map[string]bool{}
		var merged []string
		candidates := append([]string{}, DefaultFCCList()...)
		candidates = append(candidates, d.Settings.Forward.FCCList...)
		candidates = append(candidates, v.Forward.FCCList...)
		if v.Forward.FCC != "" {
			candidates = append(candidates, v.Forward.FCC)
		}
		for _, f := range candidates {
			f = strings.TrimSpace(f)
			if f != "" && endpoint(f) && !fccMap[f] {
				fccMap[f] = true
				merged = append(merged, f)
			}
		}
		v.Forward.FCCList = merged
		d.Settings = v
		return nil
	})
}

func (s *Store) AddFCCs(fccs ...string) error {
	return s.update(func(d *state) error {
		fccMap := map[string]bool{}
		for _, f := range d.Settings.Forward.FCCList {
			fccMap[f] = true
		}
		for _, f := range fccs {
			f = strings.TrimSpace(f)
			if f != "" && endpoint(f) && !fccMap[f] {
				fccMap[f] = true
				d.Settings.Forward.FCCList = append(d.Settings.Forward.FCCList, f)
			}
		}
		return nil
	})
}

func (s *Store) PutChannel(c Channel) error {
	c.PlayURL = ""
	if err := c.Validate(); err != nil {
		return err
	}
	c.EnsureKey()
	return s.update(func(d *state) error {
		for k, v := range d.Channels {
			if k != c.Key && IsValidChannelID(c.ID) && v.ID != "" && v.ID == c.ID {
				if c.OperatorID == "" || v.OperatorID == "" || c.OperatorID == v.OperatorID {
					delete(d.Channels, k)
				}
			}
		}
		d.Channels[c.Key] = c
		return nil
	})
}

// EditChannel moves the override atomically while retaining its catalog identity.
func (s *Store) EditChannel(oldID string, c Channel) error {
	c.PlayURL = ""
	if err := c.Validate(); err != nil {
		return err
	}
	return s.update(func(d *state) error {
		if c.Key == "" {
			c.EnsureKey()
		}
		if oldID != "" && oldID != c.Key {
			delete(d.Channels, oldID)
		}
		if c.OriginalID != "" && c.OriginalID != c.Key {
			delete(d.Channels, c.OriginalID)
		}
		d.Channels[c.Key] = c
		return nil
	})
}

func (s *Store) Imported() []Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Channel(nil), s.data.Imported...)
}

func (s *Store) Import(channels []Channel) error {
	channels = append([]Channel(nil), channels...)
	for i := range channels {
		channels[i].EnsureKey()
		s.ApplyMapping(&channels[i])
		PreclassifyChannel(&channels[i])
		if g := PreclassifyMulticastGroup(channels[i].URL); g != "" && g != "待识别" {
			if channels[i].Group == "" || channels[i].Group == "待识别" || channels[i].Group == "未分组" || (g == "4K" && channels[i].Group != "4K") || (g == "央视" && channels[i].Group == "高清") {
				channels[i].Group = g
			}
		}
		if err := channels[i].Validate(); err != nil {
			return fmt.Errorf("频道 %s: %w", channels[i].ID, err)
		}
	}
	return s.update(func(d *state) error { d.Imported = channels; return nil })
}

func (s *Store) UpdateResolution(key string, resolution string) error {
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return nil
	}
	return s.update(func(d *state) error {
		for k, c := range d.Channels {
			if c.Key == key || c.ID == key || c.URL == key || ("scan-"+multicastHost(c.URL)) == key {
				if c.Resolution == "" {
					c.Resolution = resolution
					d.Channels[k] = c
				}
				return nil
			}
		}
		return nil
	})
}

func (s *Store) Discover(c Channel) error {
	return s.update(func(d *state) error {
		for _, existing := range d.Channels {
			if existing.URL == c.URL {
				return nil
			}
		}
		s.ApplyMapping(&c)
		PreclassifyChannel(&c)
		c.EnsureKey()
		if _, ok := d.Channels[c.Key]; !ok {
			d.Channels[c.Key] = c
		}
		return nil
	})
}

func normalizeMappingKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(s)
	return s
}

func (s *Store) SaveMapping(m ChannelMapping) error {
	m.Keyword = strings.TrimSpace(m.Keyword)
	key := normalizeMappingKey(m.Keyword)
	if key == "" {
		return errors.New("关键词不能为空")
	}
	m.TargetID = strings.TrimSpace(m.TargetID)
	if m.TargetID == "" || IsInvalidChannelID(m.TargetID) {
		return errors.New("目标台号不能为空或无效")
	}
	m.TargetName = strings.TrimSpace(m.TargetName)
	if m.TargetName == "" {
		return errors.New("目标频道名称不能为空")
	}
	m.OperatorID = strings.TrimSpace(m.OperatorID)
	m.Group = strings.TrimSpace(m.Group)
	m.Logo = strings.TrimSpace(m.Logo)

	return s.update(func(d *state) error {
		if d.Mappings == nil {
			d.Mappings = map[string]ChannelMapping{}
		}
		d.Mappings[key] = m
		return nil
	})
}

func (s *Store) DeleteMapping(keyword string) error {
	key := normalizeMappingKey(keyword)
	if key == "" {
		return errors.New("无效的关键词")
	}
	return s.update(func(d *state) error {
		if d.Mappings != nil {
			delete(d.Mappings, key)
		}
		return nil
	})
}

func (s *Store) Mappings() []ChannelMapping {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ChannelMapping, 0, len(s.data.Mappings))
	for _, m := range s.data.Mappings {
		res = append(res, m)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Keyword < res[j].Keyword
	})
	return res
}

func (s *Store) FindMapping(keyword string) (ChannelMapping, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.Mappings == nil {
		return ChannelMapping{}, false
	}
	key := normalizeMappingKey(keyword)
	if m, ok := s.data.Mappings[key]; ok {
		return m, true
	}
	sim := SimplifyForMatch(keyword)
	if m, ok := s.data.Mappings[normalizeMappingKey(sim)]; ok {
		return m, true
	}
	return ChannelMapping{}, false
}

// DefaultMappings returns the preset channel keyword mappings.
func DefaultMappings() []ChannelMapping {
	return []ChannelMapping{
		{
			Keyword:    "体育频道",
			TargetName: "五星体育",
			TargetID:   "8",
			OperatorID: "ch00000000000000001346",
			Group:      "本地",
			Logo:       "/logos/五星体育.png",
		},
		{
			Keyword:    "体育频道HD",
			TargetName: "五星体育HD",
			TargetID:   "108",
			OperatorID: "ch00000000000000001182",
			Group:      "本地",
			Logo:       "/logos/五星体育.png",
		},
		{
			Keyword:    "卡酷卡通",
			TargetName: "卡酷少儿",
			TargetID:   "237",
			OperatorID: "ch00000000000000001102",
			Group:      "标清",
			Logo:       "/logos/卡酷少儿.png",
		},
		{
			Keyword:    "卡酷卡通HD",
			TargetName: "卡酷少儿HD",
			TargetID:   "167",
			OperatorID: "ch00000000000000001344",
			Group:      "高清",
			Logo:       "/logos/卡酷少儿.png",
		},
	}
}

// ResetMappings restores the channel keyword mappings to system defaults.
func (s *Store) ResetMappings() error {
	return s.update(func(d *state) error {
		d.Mappings = map[string]ChannelMapping{}
		for _, m := range DefaultMappings() {
			d.Mappings[normalizeMappingKey(m.Keyword)] = m
		}
		return nil
	})
}

// ApplyMapping applies configured keyword mapping to a channel.
func (s *Store) ApplyMapping(c *Channel) bool {
	if s == nil || c == nil {
		return false
	}
	m, found := s.FindMapping(c.Name)
	if !found {
		return false
	}
	changed := false
	if m.TargetName != "" && c.Name != m.TargetName {
		c.Name = m.TargetName
		changed = true
	}
	if m.TargetID != "" && IsInvalidChannelID(c.ID) && IsValidChannelID(m.TargetID) {
		c.ID = m.TargetID
		changed = true
	}
	if m.OperatorID != "" && c.OperatorID == "" {
		c.OperatorID = m.OperatorID
		changed = true
	}
	if m.Group != "" && (c.Group == "" || c.Group == "待识别" || c.Group == "未分组") {
		c.Group = m.Group
		changed = true
	}
	if m.Logo != "" && (c.Logo == "" || strings.Contains(c.Logo, "_") || strings.HasPrefix(c.Logo, "/logos/未知频道") || c.Logo == "/logos/体育频道.png") {
		if c.Logo != m.Logo {
			c.Logo = m.Logo
			changed = true
		}
	}
	return changed
}
