package panel

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type IPTV struct {
	Enabled  bool   `json:"enabled"`
	UID      string `json:"uid"`
	SN       string `json:"sn"`
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Type     string `json:"type"`
	AuthHost string `json:"auth_host"`
}

const DefaultCatchupTemplate = "&utc=${start}&lutc=${end}"

type Forward struct {
	PlayMode        string   `json:"play_mode"`
	Address         string   `json:"address"`
	Protocol        string   `json:"protocol"`
	FCC             string   `json:"fcc"`
	FCCList         []string `json:"fcc_list,omitempty"`
	CatchupTemplate string   `json:"catchup_template,omitempty"`
}

type ScanConfig struct {
	StartIP   string `json:"start_ip"`
	EndIP     string `json:"end_ip"`
	StartPort int    `json:"start_port"`
	EndPort   int    `json:"end_port"`
	Workers   int    `json:"workers"`
	Timeout   int    `json:"timeout"`
}

type AIConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type Settings struct {
	Logos             LogoSources         `json:"logos"`
	EPG               EPGConfig           `json:"epg"`
	IPTV              IPTV                `json:"iptv"`
	Forward           Forward             `json:"forward"`
	Scan              ScanConfig          `json:"scan"`
	AI                AIConfig            `json:"ai"`
	GroupOrder        []string            `json:"group_order,omitempty"`
	GroupChannelOrder map[string][]string `json:"group_channel_order,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{
		Logos:   defaultLogoSources(),
		EPG:     EPGConfig{Enabled: true, IntervalHours: 6, PastDays: 7, FutureDays: 3},
		IPTV:    IPTV{Type: "B860A", AuthHost: "222.68.208.73:7001"},
		Forward: Forward{
			Address:         "192.168.190.1:4022",
			Protocol:        "rtp",
			FCC:             "124.75.26.151:15970",
			FCCList:         DefaultFCCList(),
			CatchupTemplate: DefaultCatchupTemplate,
		},
		Scan:    ScanConfig{StartIP: "239.45.0.1", EndIP: "239.45.0.10", StartPort: 5140, EndPort: 5140, Workers: 4, Timeout: 5},
		AI:                AIConfig{BaseURL: "https://api.openai.com/v1"},
		GroupOrder:        DefaultGroupOrder(),
		GroupChannelOrder: map[string][]string{},
	}
}

func DefaultGroupOrder() []string {
	return []string{
		"央视频道",
		"卫视频道",
		"上海频道",
		"数字频道",
		"其它",
		"待识别",
	}
}

func DefaultFCCList() []string {
	return []string{
		"124.75.26.151:15970",
		"124.75.26.152:15970",
		"124.75.26.153:15970",
		"124.75.29.151:15970",
		"124.75.29.152:15970",
		"124.75.34.151:15970",
	}
}

func Endpoint(s string) bool {
	return endpoint(s)
}

func endpoint(s string) bool {
	host, port, err := net.SplitHostPort(s)
	p, e := strconv.Atoi(port)
	return err == nil && e == nil && host != "" && !strings.ContainsAny(host, "/?#@ \r\n\t") && p > 0 && p <= 65535
}

func httpURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

func (s Settings) Validate() error {
	if err := s.Logos.validate(); err != nil {
		return err
	}
	if s.EPG.IntervalHours < 1 || s.EPG.IntervalHours > 168 || s.EPG.PastDays < 0 || s.EPG.PastDays > 14 || s.EPG.FutureDays < 0 || s.EPG.FutureDays > 7 {
		return errors.New("EPG 更新间隔为 1–168 小时，历史 0–14 天，未来 0–7 天")
	}
	if s.Forward.PlayMode != "" && s.Forward.PlayMode != "multicast" && s.Forward.PlayMode != "unicast" && s.Forward.PlayMode != "http" {
		return errors.New("播放方式必须为 multicast、unicast 或 http")
	}
	if s.IPTV.Enabled && (s.IPTV.UID == "" || s.IPTV.SN == "" || s.IPTV.MAC == "" || net.ParseIP(s.IPTV.IP) == nil || !endpoint(s.IPTV.AuthHost)) {
		return errors.New("请填写完整的 IPTV 账号、SN、MAC、机顶盒 IP 和认证服务器")
	}
	if s.Forward.Address != "" && !endpoint(s.Forward.Address) {
		return errors.New("转发地址必须为 host:port")
	}
	if s.Forward.FCC != "" && !endpoint(s.Forward.FCC) {
		return errors.New("FCC 必须为 host:port")
	}
	for _, f := range s.Forward.FCCList {
		if f != "" && !endpoint(f) {
			return errors.New("FCC 列表项必须为 host:port")
		}
	}
	if s.Forward.Protocol != "udp" && s.Forward.Protocol != "rtp" {
		return errors.New("转发协议必须为 udp 或 rtp")
	}
	if _, err := s.Scan.Bounds(); err != nil {
		return err
	}
	if s.AI.BaseURL != "" && (!httpURL(s.AI.BaseURL) || strings.Contains(s.AI.BaseURL, "?")) {
		return errors.New("AI Base URL 必须为 HTTP(S) 地址，不含查询参数")
	}
	if len(s.AI.APIKey) > 4096 || strings.ContainsAny(s.AI.APIKey, "\r\n") {
		return errors.New("无效的 API key")
	}
	return nil
}

func (s ScanConfig) Bounds() (int, error) {
	a, err := netip.ParseAddr(s.StartIP)
	b, e := netip.ParseAddr(s.EndIP)
	if err != nil || e != nil || !a.Is4() || !b.Is4() || !a.IsMulticast() || !b.IsMulticast() || a.Compare(b) > 0 {
		return 0, errors.New("请填写递增的 IPv4 组播地址范围（224.0.0.0–239.255.255.255）")
	}
	if s.StartPort < 1 || s.EndPort > 65535 || s.StartPort > s.EndPort {
		return 0, errors.New("端口范围必须在 1–65535 之间且递增")
	}
	if s.Workers < 1 || s.Workers > 32 || s.Timeout < 1 || s.Timeout > 30 {
		return 0, errors.New("并发数为 1–32，探测超时为 1–30 秒")
	}
	ipNumber := func(ip netip.Addr) uint64 {
		v := ip.As4()
		return uint64(v[0])<<24 | uint64(v[1])<<16 | uint64(v[2])<<8 | uint64(v[3])
	}
	total := (ipNumber(b) - ipNumber(a) + 1) * uint64(s.EndPort-s.StartPort+1)
	if total > 65536 {
		return 0, errors.New("一次扫描最多 65536 个 IP/端口组合")
	}
	return int(total), nil
}

// PlaybackURL is shared by export, scanning, preview and recognition.
func PlaybackURL(f Forward, source string) string {
	u, err := url.Parse(strings.Replace(source, "://@", "://", 1))
	if err == nil && u.Scheme == "rtsp" && f.Address != "" {
		return "http://" + f.Address + "/rtsp/" + strings.TrimPrefix(source, "rtsp://")
	}
	if err != nil || (u.Scheme != "rtp" && u.Scheme != "udp" && u.Scheme != "igmp") || f.Address == "" {
		return source
	}
	out := url.URL{Scheme: "http", Host: f.Address, Path: "/" + f.Protocol + "/" + u.Host, RawQuery: u.RawQuery}
	q := out.Query()
	if q.Get("fcc") == "" && f.FCC != "" {
		q.Set("fcc", f.FCC)
	}
	out.RawQuery = q.Encode()
	return out.String()
}
