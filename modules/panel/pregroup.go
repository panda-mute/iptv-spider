package panel

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// MulticastPreInfo contains preset channel metadata classified by multicast address.
type MulticastPreInfo struct {
	Octet      int    `json:"octet"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Group      string `json:"group"`
	Name       string `json:"name"`
	ChannelID  string `json:"channel_id"`
	OperatorID string `json:"operator_id"`
	Resolution string `json:"resolution"`
	Logo       string `json:"logo"`
}

// DefaultPreGroups returns the default pre-grouping order.
func DefaultPreGroups() []string {
	return []string{
		"4K",
		"央视",
		"卫视",
		"高清",
		"本地",
		"少儿",
		"标清",
		"其它",
		"待识别",
	}
}

// ParseMulticastAddress parses a multicast URL, host:port, or IP into bare IP and port.
func ParseMulticastAddress(raw string) (ip string, port int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, false
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil && u.Host != "" {
			raw = u.Host
		}
	}
	host, portStr, err := net.SplitHostPort(raw)
	if err == nil {
		p, pe := strconv.Atoi(portStr)
		if pe == nil && p > 0 && p <= 65535 {
			return host, p, true
		}
		return host, 0, true
	}
	if net.ParseIP(raw) != nil {
		return raw, 0, true
	}
	return "", 0, false
}

// PreclassifyMulticastChannel looks up preset metadata for a multicast address.
func PreclassifyMulticastChannel(raw string) (MulticastPreInfo, bool) {
	ip, _, ok := ParseMulticastAddress(raw)
	if !ok {
		return MulticastPreInfo{}, false
	}
	info, found := shanghaiTelecomMulticast[ip]
	return info, found
}

// PreclassifyMulticastGroup returns the pre-classified group for a multicast URL or IP.
// Returns empty string if the address is not recognized.
func PreclassifyMulticastGroup(raw string) string {
	info, found := PreclassifyMulticastChannel(raw)
	if !found {
		return ""
	}
	return info.Group
}

// PreclassifyChannel applies pre-grouping to a channel based on its multicast address.
// It fills in the Group if empty, "待识别", or "未分组".
// If the channel name is empty or a generic "未知频道...", it pre-fills the known name.
// If digital ID is unassigned, it pre-fills the known channel ID.
// If logo is empty or has an old hash suffix, it pre-fills the canonical logo.
// Returns true if any field was updated.
func PreclassifyChannel(c *Channel) bool {
	if c == nil {
		return false
	}
	info, found := PreclassifyMulticastChannel(c.URL)
	if !found {
		return false
	}
	changed := false
	if info.Group != "" && (c.Group == "" || c.Group == "待识别" || c.Group == "未分组") {
		c.Group = info.Group
		changed = true
	}
	if info.Name != "" && (c.Name == "" || strings.HasPrefix(c.Name, "未知频道")) {
		if c.Name != info.Name {
			c.Name = info.Name
			changed = true
		}
	}
	if info.ChannelID != "" && IsInvalidChannelID(c.ID) && IsValidChannelID(info.ChannelID) {
		c.ID = info.ChannelID
		changed = true
	}
	if info.Resolution != "" && c.Resolution == "" {
		c.Resolution = info.Resolution
		changed = true
	}
	if info.Logo != "" && (c.Logo == "" || strings.Contains(c.Logo, "_") || strings.HasPrefix(c.Logo, "/logos/未知频道")) {
		if c.Logo != info.Logo {
			c.Logo = info.Logo
			changed = true
		}
	}
	return changed
}

var shanghaiTelecomMulticast = map[string]MulticastPreInfo{
	"233.18.204.1":   {Octet: 1, IP: "233.18.204.1", Port: 5140, Group: "本地", Name: "东方卫视", ChannelID: "2", OperatorID: "ch00000000000000001058", Resolution: "720x576", Logo: "/logos/东方卫视.png"},
	"233.18.204.2":   {Octet: 2, IP: "233.18.204.2", Port: 5140, Group: "本地", Name: "都市频道", ChannelID: "3", OperatorID: "ch00000000000000001143", Resolution: "720x576", Logo: "/logos/都市频道.png"},
	"233.18.204.4":   {Octet: 4, IP: "233.18.204.4", Port: 5140, Group: "本地", Name: "东方影视", ChannelID: "5", OperatorID: "ch00000000000000001462", Resolution: "720x576", Logo: "/logos/东方影视.png"},
	"233.18.204.5":   {Octet: 5, IP: "233.18.204.5", Port: 5140, Group: "本地", Name: "第一财经", ChannelID: "7", OperatorID: "ch00000000000000001028", Resolution: "720x576", Logo: "/logos/第一财经.png"},
	"233.18.204.6":   {Octet: 6, IP: "233.18.204.6", Port: 5140, Group: "本地", Name: "五星体育", ChannelID: "8", OperatorID: "ch00000000000000001346", Resolution: "720x576", Logo: "/logos/五星体育.png"},
	"233.18.204.7":   {Octet: 7, IP: "233.18.204.7", Port: 5140, Group: "少儿", Name: "哈哈炫动", ChannelID: "10", OperatorID: "ch00000000000000001226", Resolution: "720x576", Logo: "/logos/哈哈炫动.png"},
	"233.18.204.9":   {Octet: 9, IP: "233.18.204.9", Port: 5140, Group: "本地", Name: "东方购物-1", ChannelID: "12", OperatorID: "ch00000000000000001322", Resolution: "720x576", Logo: "/logos/东方购物.png"},
	"233.18.204.10":  {Octet: 10, IP: "233.18.204.10", Port: 5140, Group: "本地", Name: "东方购物-2", ChannelID: "13", OperatorID: "ch00000000000000001332", Resolution: "720x576", Logo: "/logos/东方购物.png"},
	"233.18.204.11":  {Octet: 11, IP: "233.18.204.11", Port: 5140, Group: "本地", Name: "上海教育", ChannelID: "16", OperatorID: "ch00000000000000001432", Resolution: "720x576", Logo: "/logos/上海教育.png"},
	"233.18.204.12":  {Octet: 12, IP: "233.18.204.12", Port: 5140, Group: "标清", Name: "法治天地", ChannelID: "18", OperatorID: "ch00000000000000001295", Resolution: "720x576", Logo: "/logos/法治天地.png"},
	"233.18.204.13":  {Octet: 13, IP: "233.18.204.13", Port: 5140, Group: "标清", Name: "游戏风云", ChannelID: "19", OperatorID: "ch00000000000000001472", Resolution: "720x576", Logo: "/logos/游戏风云.png"},
	"233.18.204.14":  {Octet: 14, IP: "233.18.204.14", Port: 5140, Group: "本地", Name: "金色学堂", ChannelID: "20", OperatorID: "ch00000000000000001132", Resolution: "720x576", Logo: "/logos/金色学堂.png"},
	"233.18.204.15":  {Octet: 15, IP: "233.18.204.15", Port: 5140, Group: "标清", Name: "央广购物", ChannelID: "21", OperatorID: "ch00000000000000001145", Resolution: "720x576", Logo: "/logos/央广购物.png"},
	"233.18.204.20":  {Octet: 20, IP: "233.18.204.20", Port: 5140, Group: "央视", Name: "CCTV-1", ChannelID: "51", OperatorID: "ch00000000000000001372", Resolution: "720x576", Logo: "/logos/CCTV-1.png"},
	"233.18.204.21":  {Octet: 21, IP: "233.18.204.21", Port: 5140, Group: "央视", Name: "CCTV-2", ChannelID: "52", OperatorID: "ch00000000000000001392", Resolution: "720x576", Logo: "/logos/CCTV-2.png"},
	"233.18.204.22":  {Octet: 22, IP: "233.18.204.22", Port: 5140, Group: "央视", Name: "CCTV-3", ChannelID: "53", OperatorID: "ch00000000000000001044", Resolution: "720x576", Logo: "/logos/CCTV-3.png"},
	"233.18.204.23":  {Octet: 23, IP: "233.18.204.23", Port: 5140, Group: "央视", Name: "CCTV-4", ChannelID: "54", OperatorID: "ch00000000000000001282", Resolution: "720x576", Logo: "/logos/CCTV-4.png"},
	"233.18.204.24":  {Octet: 24, IP: "233.18.204.24", Port: 5140, Group: "央视", Name: "CCTV-5", ChannelID: "55", OperatorID: "ch00000000000000001313", Resolution: "720x576", Logo: "/logos/CCTV-5.png"},
	"233.18.204.25":  {Octet: 25, IP: "233.18.204.25", Port: 5140, Group: "央视", Name: "CCTV-6", ChannelID: "56", OperatorID: "ch00000000000000001323", Resolution: "720x576", Logo: "/logos/CCTV-6.png"},
	"233.18.204.26":  {Octet: 26, IP: "233.18.204.26", Port: 5140, Group: "央视", Name: "CCTV-7", ChannelID: "57", OperatorID: "ch00000000000000001253", Resolution: "720x576", Logo: "/logos/CCTV-7.png"},
	"233.18.204.27":  {Octet: 27, IP: "233.18.204.27", Port: 5140, Group: "央视", Name: "CCTV-8", ChannelID: "58", OperatorID: "ch00000000000000001452", Resolution: "720x576", Logo: "/logos/CCTV-8.png"},
	"233.18.204.28":  {Octet: 28, IP: "233.18.204.28", Port: 5140, Group: "央视", Name: "CGTN", ChannelID: "67", OperatorID: "ch00000000000000001157", Resolution: "720x576", Logo: "/logos/CGTN.png"},
	"233.18.204.29":  {Octet: 29, IP: "233.18.204.29", Port: 5140, Group: "央视", Name: "CCTV-10", ChannelID: "60", OperatorID: "ch00000000000000001163", Resolution: "720x576", Logo: "/logos/CCTV-10.png"},
	"233.18.204.30":  {Octet: 30, IP: "233.18.204.30", Port: 5140, Group: "央视", Name: "CCTV-11", ChannelID: "61", OperatorID: "ch00000000000000001373", Resolution: "720x576", Logo: "/logos/CCTV-11.png"},
	"233.18.204.31":  {Octet: 31, IP: "233.18.204.31", Port: 5140, Group: "央视", Name: "CCTV-12", ChannelID: "62", OperatorID: "ch00000000000000001029", Resolution: "720x576", Logo: "/logos/CCTV-12.png"},
	"233.18.204.32":  {Octet: 32, IP: "233.18.204.32", Port: 5140, Group: "央视", Name: "CCTV-13", ChannelID: "63", OperatorID: "ch00000000000000001347", Resolution: "720x576", Logo: "/logos/CCTV-13.png"},
	"233.18.204.33":  {Octet: 33, IP: "233.18.204.33", Port: 5140, Group: "央视", Name: "CCTV-14", ChannelID: "64", OperatorID: "ch00000000000000001095", Resolution: "720x576", Logo: "/logos/CCTV-14.png"},
	"233.18.204.34":  {Octet: 34, IP: "233.18.204.34", Port: 5140, Group: "央视", Name: "CCTV-15", ChannelID: "65", OperatorID: "ch00000000000000001242", Resolution: "720x576", Logo: "/logos/CCTV-15.png"},
	"233.18.204.35":  {Octet: 35, IP: "233.18.204.35", Port: 5140, Group: "央视", Name: "CCTV-17", ChannelID: "66", OperatorID: "ch00000000000000001401", Resolution: "720x576", Logo: "/logos/CCTV-17.png"},
	"233.18.204.36":  {Octet: 36, IP: "233.18.204.36", Port: 5140, Group: "标清", Name: "好享购物", ChannelID: "22", OperatorID: "ch00000000000000001125", Resolution: "720x576", Logo: "/logos/好享购物.png"},
	"233.18.204.39":  {Octet: 39, IP: "233.18.204.39", Port: 5140, Group: "高清", Name: "央广购物HD", ChannelID: "74", OperatorID: "ch00000000000000001793", Resolution: "1920x1080", Logo: "/logos/央广购物.png"},
	"233.18.204.40":  {Octet: 40, IP: "233.18.204.40", Port: 5140, Group: "央视", Name: "CCTV-9", ChannelID: "59", OperatorID: "ch00000000000000001343", Resolution: "720x576", Logo: "/logos/CCTV-9.png"},
	"233.18.204.41":  {Octet: 41, IP: "233.18.204.41", Port: 5140, Group: "标清", Name: "财富天下", ChannelID: "75", OperatorID: "ch00000000000000001412", Resolution: "720x576", Logo: "/logos/财富天下.png"},
	"233.18.204.42":  {Octet: 42, IP: "233.18.204.42", Port: 5140, Group: "高清", Name: "快乐垂钓HD", ChannelID: "83", OperatorID: "ch00000000000000001431", Resolution: "1920x1080", Logo: "/logos/快乐垂钓HD.png"},
	"233.18.204.44":  {Octet: 44, IP: "233.18.204.44", Port: 5140, Group: "高清", Name: "游戏风云HD", ChannelID: "93", OperatorID: "ch00000000000000001031", Resolution: "1920x1080", Logo: "/logos/游戏风云.png"},
	"233.18.204.45":  {Octet: 45, IP: "233.18.204.45", Port: 5140, Group: "高清", Name: "生活时尚HD", ChannelID: "94", OperatorID: "ch00000000000000001021", Resolution: "1920x1080", Logo: "/logos/生活时尚HD.png"},
	"233.18.204.46":  {Octet: 46, IP: "233.18.204.46", Port: 5140, Group: "高清", Name: "动漫秀场HD", ChannelID: "96", OperatorID: "ch00000000000000001011", Resolution: "1920x1080", Logo: "/logos/动漫秀场HD.png"},
	"233.18.204.47":  {Octet: 47, IP: "233.18.204.47", Port: 5140, Group: "高清", Name: "乐游HD", ChannelID: "97", OperatorID: "ch00000000000000001484", Resolution: "1920x1080", Logo: "/logos/乐游HD.png"},
	"233.18.204.48":  {Octet: 48, IP: "233.18.204.48", Port: 5140, Group: "高清", Name: "都市剧场HD", ChannelID: "98", OperatorID: "ch00000000000000001077", Resolution: "1920x1080", Logo: "/logos/都市剧场HD.png"},
	"233.18.204.49":  {Octet: 49, IP: "233.18.204.49", Port: 5140, Group: "高清", Name: "法治天地HD", ChannelID: "99", OperatorID: "ch00000000000000001032", Resolution: "1920x1080", Logo: "/logos/法治天地.png"},
	"233.18.204.50":  {Octet: 50, IP: "233.18.204.50", Port: 5140, Group: "高清", Name: "多彩文体HD", ChannelID: "100", OperatorID: "ch00000000000000001066", Resolution: "1920x1080", Logo: "/logos/多彩文体HD.png"},
	"233.18.204.51":  {Octet: 51, IP: "233.18.204.51", Port: 5140, Group: "本地", Name: "东方卫视HD", ChannelID: "101", OperatorID: "ch00000000000000001067", Resolution: "1920x1080", Logo: "/logos/东方卫视.png"},
	"233.18.204.52":  {Octet: 52, IP: "233.18.204.52", Port: 5140, Group: "央视", Name: "CCTV-1HD", ChannelID: "102", OperatorID: "ch00000000000000001062", Resolution: "1920x1080", Logo: "/logos/CCTV-1.png"},
	"233.18.204.53":  {Octet: 53, IP: "233.18.204.53", Port: 5140, Group: "本地", Name: "都市频道", ChannelID: "103", OperatorID: "ch00000000000000001215", Resolution: "1920x1080", Logo: "/logos/都市频道.png"},
	"233.18.204.54":  {Octet: 54, IP: "233.18.204.54", Port: 5140, Group: "高清", Name: "哈哈炫动HD", ChannelID: "104", OperatorID: "ch00000000000000001012", Resolution: "1920x1080", Logo: "/logos/哈哈炫动.png"},
	"233.18.204.55":  {Octet: 55, IP: "233.18.204.55", Port: 5140, Group: "本地", Name: "东方影视HD", ChannelID: "105", OperatorID: "ch00000000000000001297", Resolution: "1920x1080", Logo: "/logos/东方影视.png"},
	"233.18.204.57":  {Octet: 57, IP: "233.18.204.57", Port: 5140, Group: "本地", Name: "新闻综合HD", ChannelID: "107", OperatorID: "ch00000000000000001403", Resolution: "1920x1080", Logo: "/logos/新闻综合.png"},
	"233.18.204.58":  {Octet: 58, IP: "233.18.204.58", Port: 5140, Group: "本地", Name: "五星体育HD", ChannelID: "108", OperatorID: "ch00000000000000001182", Resolution: "1920x1080", Logo: "/logos/五星体育.png"},
	"233.18.204.59":  {Octet: 59, IP: "233.18.204.59", Port: 5140, Group: "本地", Name: "第一财经HD", ChannelID: "109", OperatorID: "ch00000000000000001057", Resolution: "1920x1080", Logo: "/logos/第一财经.png"},
	"233.18.204.61":  {Octet: 61, IP: "233.18.204.61", Port: 5140, Group: "本地", Name: "东方购物-1HD", ChannelID: "112", OperatorID: "ch00000000000000001041", Resolution: "1920x1080", Logo: "/logos/东方购物.png"},
	"233.18.204.62":  {Octet: 62, IP: "233.18.204.62", Port: 5140, Group: "本地", Name: "东方购物-2HD", ChannelID: "113", OperatorID: "ch00000000000000001061", Resolution: "1920x1080", Logo: "/logos/东方购物.png"},
	"233.18.204.63":  {Octet: 63, IP: "233.18.204.63", Port: 5140, Group: "本地", Name: "上海教育HD", ChannelID: "116", OperatorID: "ch00000000000000001583", Resolution: "1920x1080", Logo: "/logos/上海教育.png"},
	"233.18.204.65":  {Octet: 65, IP: "233.18.204.65", Port: 5140, Group: "本地", Name: "东方财经HD", ChannelID: "118", OperatorID: "ch00000000000000001623", Resolution: "1920x1080", Logo: "/logos/东方财经HD.png"},
	"233.18.204.66":  {Octet: 66, IP: "233.18.204.66", Port: 5140, Group: "本地", Name: "金色学堂HD", ChannelID: "120", OperatorID: "ch00000000000000001022", Resolution: "1920x1080", Logo: "/logos/金色学堂.png"},
	"233.18.204.67":  {Octet: 67, IP: "233.18.204.67", Port: 5140, Group: "央视", Name: "CCTV-5+HD", ChannelID: "123", OperatorID: "ch00000000000000001783", Resolution: "1920x1080", Logo: "/logos/CCTV-5+.png"},
	"233.18.204.68":  {Octet: 68, IP: "233.18.204.68", Port: 5140, Group: "央视", Name: "CCTV-2HD", ChannelID: "125", OperatorID: "ch00000000000000001391", Resolution: "1920x1080", Logo: "/logos/CCTV-2.png"},
	"233.18.204.69":  {Octet: 69, IP: "233.18.204.69", Port: 5140, Group: "央视", Name: "CCTV-3HD", ChannelID: "126", OperatorID: "ch00000000000000001233", Resolution: "1920x1080", Logo: "/logos/CCTV-3.png"},
	"233.18.204.70":  {Octet: 70, IP: "233.18.204.70", Port: 5140, Group: "央视", Name: "CCTV-4HD", ChannelID: "127", OperatorID: "ch00000000000000001174", Resolution: "1920x1080", Logo: "/logos/CCTV-4.png"},
	"233.18.204.71":  {Octet: 71, IP: "233.18.204.71", Port: 5140, Group: "央视", Name: "CCTV-5HD", ChannelID: "128", OperatorID: "ch00000000000000001361", Resolution: "1920x1080", Logo: "/logos/CCTV-5.png"},
	"233.18.204.72":  {Octet: 72, IP: "233.18.204.72", Port: 5140, Group: "央视", Name: "CCTV-6HD", ChannelID: "129", OperatorID: "ch00000000000000001133", Resolution: "1920x1080", Logo: "/logos/CCTV-6.png"},
	"233.18.204.73":  {Octet: 73, IP: "233.18.204.73", Port: 5140, Group: "央视", Name: "CCTV-7HD", ChannelID: "130", OperatorID: "ch00000000000000001381", Resolution: "1920x1080", Logo: "/logos/CCTV-7.png"},
	"233.18.204.74":  {Octet: 74, IP: "233.18.204.74", Port: 5140, Group: "央视", Name: "CCTV-8HD", ChannelID: "131", OperatorID: "ch00000000000000001371", Resolution: "1920x1080", Logo: "/logos/CCTV-8.png"},
	"233.18.204.75":  {Octet: 75, IP: "233.18.204.75", Port: 5140, Group: "央视", Name: "CCTV-9HD", ChannelID: "132", OperatorID: "ch00000000000000001224", Resolution: "1920x1080", Logo: "/logos/CCTV-9.png"},
	"233.18.204.76":  {Octet: 76, IP: "233.18.204.76", Port: 5140, Group: "央视", Name: "CCTV-10HD", ChannelID: "133", OperatorID: "ch00000000000000001123", Resolution: "1920x1080", Logo: "/logos/CCTV-10.png"},
	"233.18.204.77":  {Octet: 77, IP: "233.18.204.77", Port: 5140, Group: "央视", Name: "CCTV-11HD", ChannelID: "134", OperatorID: "ch00000000000000001633", Resolution: "1920x1080", Logo: "/logos/CCTV-11.png"},
	"233.18.204.78":  {Octet: 78, IP: "233.18.204.78", Port: 5140, Group: "央视", Name: "CCTV-12HD", ChannelID: "135", OperatorID: "ch00000000000000001063", Resolution: "1920x1080", Logo: "/logos/CCTV-12.png"},
	"233.18.204.79":  {Octet: 79, IP: "233.18.204.79", Port: 5140, Group: "央视", Name: "CCTV-13HD", ChannelID: "136", OperatorID: "ch00000000000000001643", Resolution: "1920x1080", Logo: "/logos/CCTV-13.png"},
	"233.18.204.80":  {Octet: 80, IP: "233.18.204.80", Port: 5140, Group: "央视", Name: "CCTV-14HD", ChannelID: "137", OperatorID: "ch00000000000000001214", Resolution: "1920x1080", Logo: "/logos/CCTV-14.png"},
	"233.18.204.81":  {Octet: 81, IP: "233.18.204.81", Port: 5140, Group: "央视", Name: "CCTV-15HD", ChannelID: "138", OperatorID: "ch00000000000000001653", Resolution: "1920x1080", Logo: "/logos/CCTV-15.png"},
	"233.18.204.82":  {Octet: 82, IP: "233.18.204.82", Port: 5140, Group: "央视", Name: "CCTV-16HD", ChannelID: "139", OperatorID: "ch00000000000000001563", Resolution: "1920x1080", Logo: "/logos/CCTV-16.png"},
	"233.18.204.83":  {Octet: 83, IP: "233.18.204.83", Port: 5140, Group: "央视", Name: "CCTV-17HD", ChannelID: "140", OperatorID: "ch00000000000000001035", Resolution: "1920x1080", Logo: "/logos/CCTV-17.png"},
	"233.18.204.84":  {Octet: 84, IP: "233.18.204.84", Port: 5140, Group: "卫视", Name: "浙江卫视HD", ChannelID: "141", OperatorID: "ch00000000000000001082", Resolution: "1920x1080", Logo: "/logos/浙江卫视.png"},
	"233.18.204.85":  {Octet: 85, IP: "233.18.204.85", Port: 5140, Group: "卫视", Name: "江苏卫视HD", ChannelID: "142", OperatorID: "ch00000000000000001072", Resolution: "1920x1080", Logo: "/logos/江苏卫视.png"},
	"233.18.204.86":  {Octet: 86, IP: "233.18.204.86", Port: 5140, Group: "卫视", Name: "湖南卫视HD", ChannelID: "143", OperatorID: "ch00000000000000001424", Resolution: "1920x1080", Logo: "/logos/湖南卫视.png"},
	"233.18.204.87":  {Octet: 87, IP: "233.18.204.87", Port: 5140, Group: "卫视", Name: "北京卫视HD", ChannelID: "144", OperatorID: "ch00000000000000001333", Resolution: "1920x1080", Logo: "/logos/北京卫视.png"},
	"233.18.204.88":  {Octet: 88, IP: "233.18.204.88", Port: 5140, Group: "卫视", Name: "广东卫视HD", ChannelID: "145", OperatorID: "ch00000000000000001434", Resolution: "1920x1080", Logo: "/logos/广东卫视.png"},
	"233.18.204.89":  {Octet: 89, IP: "233.18.204.89", Port: 5140, Group: "卫视", Name: "深圳卫视HD", ChannelID: "146", OperatorID: "ch00000000000000001415", Resolution: "1920x1080", Logo: "/logos/深圳卫视.png"},
	"233.18.204.90":  {Octet: 90, IP: "233.18.204.90", Port: 5140, Group: "卫视", Name: "黑龙江卫视HD", ChannelID: "147", OperatorID: "ch00000000000000001353", Resolution: "1920x1080", Logo: "/logos/黑龙江卫视.png"},
	"233.18.204.91":  {Octet: 91, IP: "233.18.204.91", Port: 5140, Group: "卫视", Name: "山东卫视HD", ChannelID: "148", OperatorID: "ch00000000000000001158", Resolution: "1920x1080", Logo: "/logos/山东卫视.png"},
	"233.18.204.92":  {Octet: 92, IP: "233.18.204.92", Port: 5140, Group: "卫视", Name: "湖北卫视HD", ChannelID: "149", OperatorID: "ch00000000000000001146", Resolution: "1920x1080", Logo: "/logos/湖北卫视.png"},
	"233.18.204.93":  {Octet: 93, IP: "233.18.204.93", Port: 5140, Group: "卫视", Name: "安徽卫视HD", ChannelID: "150", OperatorID: "ch00000000000000001151", Resolution: "1920x1080", Logo: "/logos/安徽卫视.png"},
	"233.18.204.94":  {Octet: 94, IP: "233.18.204.94", Port: 5140, Group: "卫视", Name: "东南卫视HD", ChannelID: "151", OperatorID: "ch00000000000000001161", Resolution: "1920x1080", Logo: "/logos/东南卫视.png"},
	"233.18.204.95":  {Octet: 95, IP: "233.18.204.95", Port: 5140, Group: "卫视", Name: "江西卫视HD", ChannelID: "152", OperatorID: "ch00000000000000001131", Resolution: "1920x1080", Logo: "/logos/江西卫视.png"},
	"233.18.204.96":  {Octet: 96, IP: "233.18.204.96", Port: 5140, Group: "卫视", Name: "辽宁卫视HD", ChannelID: "153", OperatorID: "ch00000000000000001141", Resolution: "1920x1080", Logo: "/logos/辽宁卫视.png"},
	"233.18.204.97":  {Octet: 97, IP: "233.18.204.97", Port: 5140, Group: "卫视", Name: "天津卫视HD", ChannelID: "154", OperatorID: "ch00000000000000001121", Resolution: "1920x1080", Logo: "/logos/天津卫视.png"},
	"233.18.204.98":  {Octet: 98, IP: "233.18.204.98", Port: 5140, Group: "高清", Name: "中国教育-1HD", ChannelID: "155", OperatorID: "ch00000000000000001091", Resolution: "1920x1080", Logo: "/logos/中国教育-1.png"},
	"233.18.204.99":  {Octet: 99, IP: "233.18.204.99", Port: 5140, Group: "卫视", Name: "四川卫视HD", ChannelID: "156", OperatorID: "ch00000000000000001191", Resolution: "1920x1080", Logo: "/logos/四川卫视.png"},
	"233.18.204.100": {Octet: 100, IP: "233.18.204.100", Port: 5140, Group: "卫视", Name: "重庆卫视HD", ChannelID: "157", OperatorID: "ch00000000000000001181", Resolution: "1920x1080", Logo: "/logos/重庆卫视.png"},
	"233.18.204.101": {Octet: 101, IP: "233.18.204.101", Port: 5140, Group: "卫视", Name: "贵州卫视HD", ChannelID: "158", OperatorID: "ch00000000000000001201", Resolution: "1920x1080", Logo: "/logos/贵州卫视.png"},
	"233.18.204.102": {Octet: 102, IP: "233.18.204.102", Port: 5140, Group: "卫视", Name: "海南卫视HD", ChannelID: "159", OperatorID: "ch00000000000000001042", Resolution: "1920x1080", Logo: "/logos/海南卫视.png"},
	"233.18.204.103": {Octet: 103, IP: "233.18.204.103", Port: 5140, Group: "卫视", Name: "河北卫视HD", ChannelID: "160", OperatorID: "ch00000000000000001192", Resolution: "1920x1080", Logo: "/logos/河北卫视.png"},
	"233.18.204.104": {Octet: 104, IP: "233.18.204.104", Port: 5140, Group: "高清", Name: "金鹰纪实HD", ChannelID: "161", OperatorID: "ch00000000000000001052", Resolution: "1920x1080", Logo: "/logos/金鹰纪实HD.png"},
	"233.18.204.105": {Octet: 105, IP: "233.18.204.105", Port: 5140, Group: "卫视", Name: "河南卫视HD", ChannelID: "163", OperatorID: "ch00000000000000001411", Resolution: "1920x1080", Logo: "/logos/河南卫视.png"},
	"233.18.204.106": {Octet: 106, IP: "233.18.204.106", Port: 5140, Group: "卫视", Name: "云南卫视HD", ChannelID: "164", OperatorID: "ch00000000000000001001", Resolution: "1920x1080", Logo: "/logos/云南卫视.png"},
	"233.18.204.107": {Octet: 107, IP: "233.18.204.107", Port: 5140, Group: "卫视", Name: "广西卫视HD", ChannelID: "165", OperatorID: "ch00000000000000001025", Resolution: "1920x1080", Logo: "/logos/广西卫视.png"},
	"233.18.204.108": {Octet: 108, IP: "233.18.204.108", Port: 5140, Group: "卫视", Name: "吉林卫视HD", ChannelID: "166", OperatorID: "ch00000000000000001421", Resolution: "1920x1080", Logo: "/logos/吉林卫视.png"},
	"233.18.204.109": {Octet: 109, IP: "233.18.204.109", Port: 5140, Group: "高清", Name: "卡酷少儿HD", ChannelID: "167", OperatorID: "ch00000000000000001344", Resolution: "1920x1080", Logo: "/logos/卡酷少儿.png"},
	"233.18.204.110": {Octet: 110, IP: "233.18.204.110", Port: 5140, Group: "卫视", Name: "甘肃卫视HD", ChannelID: "168", OperatorID: "ch00000000000000001603", Resolution: "1920x1080", Logo: "/logos/甘肃卫视.png"},
	"233.18.204.111": {Octet: 111, IP: "233.18.204.111", Port: 5140, Group: "高清", Name: "中国教育-4HD", ChannelID: "169", OperatorID: "ch00000000000000001663", Resolution: "1920x1080", Logo: "/logos/中国教育-4HD.png"},
	"233.18.204.112": {Octet: 112, IP: "233.18.204.112", Port: 5140, Group: "卫视", Name: "青海卫视HD", ChannelID: "170", OperatorID: "ch00000000000000001823", Resolution: "1920x1080", Logo: "/logos/青海卫视.png"},
	"233.18.204.113": {Octet: 113, IP: "233.18.204.113", Port: 5140, Group: "高清", Name: "金鹰卡通HD", ChannelID: "171", OperatorID: "ch00000000000000001803", Resolution: "1920x1080", Logo: "/logos/金鹰卡通.png"},
	"233.18.204.114": {Octet: 114, IP: "233.18.204.114", Port: 5140, Group: "4K", Name: "CCTV-16 4K", ChannelID: "139", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/CCTV-16 4K.png"},
	"233.18.204.115": {Octet: 115, IP: "233.18.204.115", Port: 5140, Group: "4K", Name: "多彩文体4K", ChannelID: "100", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/多彩文体4K.png"},
	"233.18.204.116": {Octet: 116, IP: "233.18.204.116", Port: 5140, Group: "高清", Name: "高清导视频道", ChannelID: "198", OperatorID: "ch00000000000000001298", Resolution: "", Logo: "/logos/高清导视频道.png"},
	"233.18.204.117": {Octet: 117, IP: "233.18.204.117", Port: 5140, Group: "卫视", Name: "宁夏卫视", ChannelID: "201", OperatorID: "ch00000000000000001481", Resolution: "720x576", Logo: "/logos/宁夏卫视.png"},
	"233.18.204.118": {Octet: 118, IP: "233.18.204.118", Port: 5140, Group: "标清", Name: "北京卫视", ChannelID: "202", OperatorID: "ch00000000000000001467", Resolution: "720x576", Logo: "/logos/北京卫视.png"},
	"233.18.204.119": {Octet: 119, IP: "233.18.204.119", Port: 5140, Group: "标清", Name: "湖南卫视", ChannelID: "203", OperatorID: "ch00000000000000001501", Resolution: "720x576", Logo: "/logos/湖南卫视.png"},
	"233.18.204.120": {Octet: 120, IP: "233.18.204.120", Port: 5140, Group: "标清", Name: "江苏卫视", ChannelID: "204", OperatorID: "ch00000000000000001147", Resolution: "720x576", Logo: "/logos/江苏卫视.png"},
	"233.18.204.121": {Octet: 121, IP: "233.18.204.121", Port: 5140, Group: "标清", Name: "浙江卫视", ChannelID: "205", OperatorID: "ch00000000000000001115", Resolution: "720x576", Logo: "/logos/浙江卫视.png"},
	"233.18.204.122": {Octet: 122, IP: "233.18.204.122", Port: 5140, Group: "标清", Name: "海南卫视", ChannelID: "206", OperatorID: "ch00000000000000001037", Resolution: "720x576", Logo: "/logos/海南卫视.png"},
	"233.18.204.123": {Octet: 123, IP: "233.18.204.123", Port: 5140, Group: "标清", Name: "广西卫视", ChannelID: "207", OperatorID: "ch00000000000000001362", Resolution: "720x576", Logo: "/logos/广西卫视.png"},
	"233.18.204.124": {Octet: 124, IP: "233.18.204.124", Port: 5140, Group: "标清", Name: "四川卫视", ChannelID: "208", OperatorID: "ch00000000000000001293", Resolution: "720x576", Logo: "/logos/四川卫视.png"},
	"233.18.204.125": {Octet: 125, IP: "233.18.204.125", Port: 5140, Group: "标清", Name: "山东卫视", ChannelID: "209", OperatorID: "ch00000000000000001374", Resolution: "720x576", Logo: "/logos/山东卫视.png"},
	"233.18.204.126": {Octet: 126, IP: "233.18.204.126", Port: 5140, Group: "标清", Name: "辽宁卫视", ChannelID: "210", OperatorID: "ch00000000000000001252", Resolution: "720x576", Logo: "/logos/辽宁卫视.png"},
	"233.18.204.127": {Octet: 127, IP: "233.18.204.127", Port: 5140, Group: "标清", Name: "安徽卫视", ChannelID: "211", OperatorID: "ch00000000000000001435", Resolution: "720x576", Logo: "/logos/安徽卫视.png"},
	"233.18.204.128": {Octet: 128, IP: "233.18.204.128", Port: 5140, Group: "标清", Name: "东南卫视", ChannelID: "212", OperatorID: "ch00000000000000001134", Resolution: "720x576", Logo: "/logos/东南卫视.png"},
	"233.18.204.129": {Octet: 129, IP: "233.18.204.129", Port: 5140, Group: "标清", Name: "天津卫视", ChannelID: "214", OperatorID: "ch00000000000000001413", Resolution: "720x576", Logo: "/logos/天津卫视.png"},
	"233.18.204.130": {Octet: 130, IP: "233.18.204.130", Port: 5140, Group: "标清", Name: "江西卫视", ChannelID: "215", OperatorID: "ch00000000000000001043", Resolution: "720x576", Logo: "/logos/江西卫视.png"},
	"233.18.204.131": {Octet: 131, IP: "233.18.204.131", Port: 5140, Group: "标清", Name: "吉林卫视", ChannelID: "216", OperatorID: "ch00000000000000001312", Resolution: "720x576", Logo: "/logos/吉林卫视.png"},
	"233.18.204.132": {Octet: 132, IP: "233.18.204.132", Port: 5140, Group: "卫视", Name: "山西卫视", ChannelID: "217", OperatorID: "ch00000000000000001055", Resolution: "720x576", Logo: "/logos/山西卫视.png"},
	"233.18.204.133": {Octet: 133, IP: "233.18.204.133", Port: 5140, Group: "标清", Name: "青海卫视", ChannelID: "218", OperatorID: "ch00000000000000001027", Resolution: "720x576", Logo: "/logos/青海卫视.png"},
	"233.18.204.134": {Octet: 134, IP: "233.18.204.134", Port: 5140, Group: "卫视", Name: "西藏卫视", ChannelID: "219", OperatorID: "ch00000000000000001294", Resolution: "720x576", Logo: "/logos/西藏卫视.png"},
	"233.18.204.135": {Octet: 135, IP: "233.18.204.135", Port: 5140, Group: "卫视", Name: "陕西卫视", ChannelID: "220", OperatorID: "ch00000000000000001154", Resolution: "720x576", Logo: "/logos/陕西卫视.png"},
	"233.18.204.136": {Octet: 136, IP: "233.18.204.136", Port: 5140, Group: "标清", Name: "云南卫视", ChannelID: "221", OperatorID: "ch00000000000000001014", Resolution: "720x576", Logo: "/logos/云南卫视.png"},
	"233.18.204.137": {Octet: 137, IP: "233.18.204.137", Port: 5140, Group: "标清", Name: "甘肃卫视", ChannelID: "222", OperatorID: "ch00000000000000001074", Resolution: "720x576", Logo: "/logos/甘肃卫视.png"},
	"233.18.204.138": {Octet: 138, IP: "233.18.204.138", Port: 5140, Group: "标清", Name: "广东卫视", ChannelID: "223", OperatorID: "ch00000000000000001046", Resolution: "720x576", Logo: "/logos/广东卫视.png"},
	"233.18.204.139": {Octet: 139, IP: "233.18.204.139", Port: 5140, Group: "标清", Name: "黑龙江卫视", ChannelID: "224", OperatorID: "ch00000000000000001302", Resolution: "720x576", Logo: "/logos/黑龙江卫视.png"},
	"233.18.204.140": {Octet: 140, IP: "233.18.204.140", Port: 5140, Group: "标清", Name: "河北卫视", ChannelID: "225", OperatorID: "ch00000000000000001461", Resolution: "720x576", Logo: "/logos/河北卫视.png"},
	"233.18.204.141": {Octet: 141, IP: "233.18.204.141", Port: 5140, Group: "卫视", Name: "内蒙古卫视", ChannelID: "226", OperatorID: "ch00000000000000001225", Resolution: "720x576", Logo: "/logos/内蒙古卫视.png"},
	"233.18.204.142": {Octet: 142, IP: "233.18.204.142", Port: 5140, Group: "标清", Name: "湖北卫视", ChannelID: "227", OperatorID: "ch00000000000000001272", Resolution: "720x576", Logo: "/logos/湖北卫视.png"},
	"233.18.204.143": {Octet: 143, IP: "233.18.204.143", Port: 5140, Group: "标清", Name: "重庆卫视", ChannelID: "228", OperatorID: "ch00000000000000001315", Resolution: "720x576", Logo: "/logos/重庆卫视.png"},
	"233.18.204.144": {Octet: 144, IP: "233.18.204.144", Port: 5140, Group: "标清", Name: "贵州卫视", ChannelID: "229", OperatorID: "ch00000000000000001451", Resolution: "720x576", Logo: "/logos/贵州卫视.png"},
	"233.18.204.145": {Octet: 145, IP: "233.18.204.145", Port: 5140, Group: "标清", Name: "河南卫视", ChannelID: "230", OperatorID: "ch00000000000000001054", Resolution: "720x576", Logo: "/logos/河南卫视.png"},
	"233.18.204.146": {Octet: 146, IP: "233.18.204.146", Port: 5140, Group: "标清", Name: "深圳卫视", ChannelID: "231", OperatorID: "ch00000000000000001345", Resolution: "720x576", Logo: "/logos/深圳卫视.png"},
	"233.18.204.147": {Octet: 147, IP: "233.18.204.147", Port: 5140, Group: "卫视", Name: "新疆卫视", ChannelID: "232", OperatorID: "ch00000000000000001471", Resolution: "720x576", Logo: "/logos/新疆卫视.png"},
	"233.18.204.148": {Octet: 148, IP: "233.18.204.148", Port: 5140, Group: "卫视", Name: "兵团卫视", ChannelID: "233", OperatorID: "ch00000000000000001064", Resolution: "720x576", Logo: "/logos/兵团卫视.png"},
	"233.18.204.149": {Octet: 149, IP: "233.18.204.149", Port: 5140, Group: "标清", Name: "三沙卫视", ChannelID: "234", OperatorID: "ch00000000000000001422", Resolution: "720x576", Logo: "/logos/三沙卫视.png"},
	"233.18.204.150": {Octet: 150, IP: "233.18.204.150", Port: 5140, Group: "标清", Name: "金鹰卡通", ChannelID: "235", OperatorID: "ch00000000000000001092", Resolution: "720x576", Logo: "/logos/金鹰卡通.png"},
	"233.18.204.151": {Octet: 151, IP: "233.18.204.151", Port: 5140, Group: "少儿", Name: "嘉佳卡通", ChannelID: "236", OperatorID: "ch00000000000000001172", Resolution: "720x576", Logo: "/logos/嘉佳卡通.png"},
	"233.18.204.152": {Octet: 152, IP: "233.18.204.152", Port: 5140, Group: "标清", Name: "卡酷少儿", ChannelID: "237", OperatorID: "ch00000000000000001102", Resolution: "720x576", Logo: "/logos/卡酷少儿.png"},
	"233.18.204.153": {Octet: 153, IP: "233.18.204.153", Port: 5140, Group: "标清", Name: "中国教育-1", ChannelID: "238", OperatorID: "ch00000000000000001112", Resolution: "720x576", Logo: "/logos/中国教育-1.png"},
	"233.18.204.154": {Octet: 154, IP: "233.18.204.154", Port: 5140, Group: "少儿", Name: "中国教育-2", ChannelID: "239", OperatorID: "ch00000000000000001122", Resolution: "720x576", Logo: "/logos/中国教育-2.png"},
	"233.18.204.155": {Octet: 155, IP: "233.18.204.155", Port: 5140, Group: "卫视", Name: "延边卫视", ChannelID: "243", OperatorID: "ch00000000000000001813", Resolution: "720x576", Logo: "/logos/延边卫视.png"},
	"233.18.204.157": {Octet: 157, IP: "233.18.204.157", Port: 5140, Group: "标清", Name: "家庭理财", ChannelID: "350", OperatorID: "ch00000000000000001593", Resolution: "720x576", Logo: "/logos/家庭理财.png"},
	"233.18.204.158": {Octet: 158, IP: "233.18.204.158", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "0", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.159": {Octet: 159, IP: "233.18.204.159", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "0", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.160": {Octet: 160, IP: "233.18.204.160", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.160:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.161": {Octet: 161, IP: "233.18.204.161", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.161:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.162": {Octet: 162, IP: "233.18.204.162", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.162:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.163": {Octet: 163, IP: "233.18.204.163", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.163:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.164": {Octet: 164, IP: "233.18.204.164", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.164:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.165": {Octet: 165, IP: "233.18.204.165", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.165:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.166": {Octet: 166, IP: "233.18.204.166", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.166:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.167": {Octet: 167, IP: "233.18.204.167", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.167:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.168": {Octet: 168, IP: "233.18.204.168", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.168:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.169": {Octet: 169, IP: "233.18.204.169", Port: 5140, Group: "4K", Name: "BesTV4K电影", ChannelID: "0", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/BesTV4K电影.png"},
	"233.18.204.170": {Octet: 170, IP: "233.18.204.170", Port: 5140, Group: "4K", Name: "BesTV4K记录", ChannelID: "0", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/BesTV4K记录.png"},
	"233.18.204.171": {Octet: 171, IP: "233.18.204.171", Port: 5140, Group: "4K", Name: "BesTV4K动画", ChannelID: "0", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/BesTV4K动画.png"},
	"233.18.204.172": {Octet: 172, IP: "233.18.204.172", Port: 5140, Group: "卫视", Name: "三沙卫视HD", ChannelID: "162", OperatorID: "ch00000000000000001854", Resolution: "1920x1080", Logo: "/logos/三沙卫视.png"},
	"233.18.204.173": {Octet: 173, IP: "233.18.204.173", Port: 5140, Group: "高清", Name: "CHC家庭影院HD", ChannelID: "76", OperatorID: "ch00000000000000001855", Resolution: "1920x1080", Logo: "/logos/CHC家庭影院HD.png"},
	"233.18.204.174": {Octet: 174, IP: "233.18.204.174", Port: 5140, Group: "高清", Name: "CHC动作电影HD", ChannelID: "77", OperatorID: "ch00000000000000001865", Resolution: "1920x1080", Logo: "/logos/CHC动作电影HD.png"},
	"233.18.204.175": {Octet: 175, IP: "233.18.204.175", Port: 5140, Group: "高清", Name: "CHC影迷电影HD", ChannelID: "78", OperatorID: "ch00000000000000001856", Resolution: "1920x1080", Logo: "/logos/CHC影迷电影HD.png"},
	"233.18.204.176": {Octet: 176, IP: "233.18.204.176", Port: 5140, Group: "央视", Name: "风云足球HD", ChannelID: "79", OperatorID: "ch00000000000000001866", Resolution: "1920x1080", Logo: "/logos/风云足球HD.png"},
	"233.18.204.177": {Octet: 177, IP: "233.18.204.177", Port: 5140, Group: "央视", Name: "央视台球HD", ChannelID: "80", OperatorID: "ch00000000000000001875", Resolution: "1920x1080", Logo: "/logos/央视台球HD.png"},
	"233.18.204.178": {Octet: 178, IP: "233.18.204.178", Port: 5140, Group: "央视", Name: "兵器科技HD", ChannelID: "81", OperatorID: "ch00000000000000001859", Resolution: "1920x1080", Logo: "/logos/兵器科技HD.png"},
	"233.18.204.179": {Octet: 179, IP: "233.18.204.179", Port: 5140, Group: "央视", Name: "世界地理HD", ChannelID: "82", OperatorID: "ch00000000000000001861", Resolution: "1920x1080", Logo: "/logos/世界地理HD.png"},
	"233.18.204.180": {Octet: 180, IP: "233.18.204.180", Port: 5140, Group: "央视", Name: "女性时尚HD", ChannelID: "85", OperatorID: "ch00000000000000001862", Resolution: "1920x1080", Logo: "/logos/女性时尚HD.png"},
	"233.18.204.181": {Octet: 181, IP: "233.18.204.181", Port: 5140, Group: "央视", Name: "高尔夫网球HD", ChannelID: "86", OperatorID: "ch00000000000000001885", Resolution: "1920x1080", Logo: "/logos/高尔夫网球HD.png"},
	"233.18.204.182": {Octet: 182, IP: "233.18.204.182", Port: 5140, Group: "央视", Name: "怀旧剧场HD", ChannelID: "87", OperatorID: "ch00000000000000001895", Resolution: "1920x1080", Logo: "/logos/怀旧剧场HD.png"},
	"233.18.204.183": {Octet: 183, IP: "233.18.204.183", Port: 5140, Group: "央视", Name: "风云剧场HD", ChannelID: "88", OperatorID: "ch00000000000000001863", Resolution: "1920x1080", Logo: "/logos/风云剧场HD.png"},
	"233.18.204.184": {Octet: 184, IP: "233.18.204.184", Port: 5140, Group: "央视", Name: "第一剧场HD", ChannelID: "89", OperatorID: "ch00000000000000001905", Resolution: "1920x1080", Logo: "/logos/第一剧场HD.png"},
	"233.18.204.185": {Octet: 185, IP: "233.18.204.185", Port: 5140, Group: "央视", Name: "风云音乐HD", ChannelID: "90", OperatorID: "ch00000000000000001915", Resolution: "1920x1080", Logo: "/logos/风云音乐HD.png"},
	"233.18.204.186": {Octet: 186, IP: "233.18.204.186", Port: 5140, Group: "央视", Name: "央视文化精品HD", ChannelID: "91", OperatorID: "ch00000000000000001886", Resolution: "1920x1080", Logo: "/logos/央视文化精品HD.png"},
	"233.18.204.187": {Octet: 187, IP: "233.18.204.187", Port: 5140, Group: "高清", Name: "早期教育HD", ChannelID: "92", OperatorID: "ch00000000000000001864", Resolution: "1920x1080", Logo: "/logos/早期教育HD.png"},
	"233.18.204.188": {Octet: 188, IP: "233.18.204.188", Port: 5140, Group: "4K", Name: "CCTV-4K", ChannelID: "124", OperatorID: "ch00000000000000001860", Resolution: "3840x2160", Logo: "/logos/CCTV-4K.png"},
	"233.18.204.189": {Octet: 189, IP: "233.18.204.189", Port: 5140, Group: "本地", Name: "新闻综合", ChannelID: "1", OperatorID: "ch00000000000000001485", Resolution: "720x576", Logo: "/logos/新闻综合.png"},
	"233.18.204.201": {Octet: 201, IP: "233.18.204.201", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.201:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.202": {Octet: 202, IP: "233.18.204.202", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.202:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.203": {Octet: 203, IP: "233.18.204.203", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.203:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.204": {Octet: 204, IP: "233.18.204.204", Port: 5140, Group: "4K", Name: "CCTV-4K", ChannelID: "124", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/CCTV-4K.png"},
	"233.18.204.205": {Octet: 205, IP: "233.18.204.205", Port: 5140, Group: "4K", Name: "多彩文体4K", ChannelID: "100", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/多彩文体4K.png"},
	"233.18.204.206": {Octet: 206, IP: "233.18.204.206", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.206:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.207": {Octet: 207, IP: "233.18.204.207", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.207:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.208": {Octet: 208, IP: "233.18.204.208", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "scan-233.18.204.208:5140", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.209": {Octet: 209, IP: "233.18.204.209", Port: 5140, Group: "4K", Name: "北京卫视4K", ChannelID: "144", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/北京卫视4K.png"},
	"233.18.204.210": {Octet: 210, IP: "233.18.204.210", Port: 5140, Group: "央视", Name: "CCTV-1HD", ChannelID: "0", OperatorID: "", Resolution: "1920x1080", Logo: "/logos/CCTV-1.png"},
	"233.18.204.211": {Octet: 211, IP: "233.18.204.211", Port: 5140, Group: "卫视", Name: "山西卫视HD", ChannelID: "172", OperatorID: "ch00000000000000001998", Resolution: "1920x1080", Logo: "/logos/山西卫视.png"},
	"233.18.204.215": {Octet: 215, IP: "233.18.204.215", Port: 5140, Group: "4K", Name: "CCTV-16 4K", ChannelID: "139", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/CCTV-16 4K.png"},
	"233.18.204.216": {Octet: 216, IP: "233.18.204.216", Port: 5140, Group: "待识别", Name: "未知频道", ChannelID: "0", OperatorID: "", Resolution: "", Logo: ""},
	"233.18.204.217": {Octet: 217, IP: "233.18.204.217", Port: 5140, Group: "卫视", Name: "内蒙古卫视HD", ChannelID: "173", OperatorID: "ch00000000000000002007", Resolution: "1920x1080", Logo: "/logos/内蒙古卫视.png"},
	"233.18.204.218": {Octet: 218, IP: "233.18.204.218", Port: 5140, Group: "4K", Name: "广东卫视4K", ChannelID: "145", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/广东卫视4K.png"},
	"233.18.204.219": {Octet: 219, IP: "233.18.204.219", Port: 5140, Group: "4K", Name: "深圳卫视4K", ChannelID: "146", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/深圳卫视4K.png"},
	"233.18.204.220": {Octet: 220, IP: "233.18.204.220", Port: 5140, Group: "卫视", Name: "新疆卫视HD", ChannelID: "174", OperatorID: "ch00000000000000002057", Resolution: "1920x1080", Logo: "/logos/新疆卫视.png"},
	"233.18.204.221": {Octet: 221, IP: "233.18.204.221", Port: 5140, Group: "卫视", Name: "兵团卫视HD", ChannelID: "175", OperatorID: "ch00000000000000002058", Resolution: "1920x1080", Logo: "/logos/兵团卫视.png"},
	"233.18.204.222": {Octet: 222, IP: "233.18.204.222", Port: 5140, Group: "卫视", Name: "西藏卫视HD", ChannelID: "176", OperatorID: "ch00000000000000002011", Resolution: "1920x1080", Logo: "/logos/西藏卫视.png"},
	"233.18.204.223": {Octet: 223, IP: "233.18.204.223", Port: 5140, Group: "卫视", Name: "陕西卫视HD", ChannelID: "177", OperatorID: "ch00000000000000002037", Resolution: "1920x1080", Logo: "/logos/陕西卫视.png"},
	"233.18.204.224": {Octet: 224, IP: "233.18.204.224", Port: 5140, Group: "4K", Name: "东方卫视4K", ChannelID: "101", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/东方卫视4K.png"},
	"233.18.204.225": {Octet: 225, IP: "233.18.204.225", Port: 5140, Group: "4K", Name: "四川卫视4K", ChannelID: "156", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/四川卫视4K.png"},
	"233.18.204.226": {Octet: 226, IP: "233.18.204.226", Port: 5140, Group: "4K", Name: "江苏卫视4K", ChannelID: "142", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/江苏卫视4K.png"},
	"233.18.204.227": {Octet: 227, IP: "233.18.204.227", Port: 5140, Group: "4K", Name: "湖南卫视4K", ChannelID: "143", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/湖南卫视4K.png"},
	"233.18.204.228": {Octet: 228, IP: "233.18.204.228", Port: 5140, Group: "4K", Name: "山东卫视4K", ChannelID: "148", OperatorID: "", Resolution: "3840x2160", Logo: "/logos/山东卫视4K.png"},
	"233.18.204.229": {Octet: 229, IP: "233.18.204.229", Port: 5140, Group: "4K", Name: "浙江卫视4K", ChannelID: "141", OperatorID: "", Resolution: "3840x2176", Logo: "/logos/浙江卫视4K.png"},
	"233.18.204.230": {Octet: 230, IP: "233.18.204.230", Port: 5140, Group: "卫视", Name: "宁夏卫视HD", ChannelID: "178", OperatorID: "ch00000000000000002117", Resolution: "1920x1080", Logo: "/logos/宁夏卫视.png"},
	"233.18.204.231": {Octet: 231, IP: "233.18.204.231", Port: 5140, Group: "高清", Name: "财富天下HD", ChannelID: "197", OperatorID: "ch00000000000000002107", Resolution: "1920x1080", Logo: "/logos/财富天下.png"},
}
