package panel

import (
	"testing"
)

func TestPreclassifyMulticastGroup(t *testing.T) {
	cases := []struct {
		url       string
		wantGroup string
	}{
		// 4K
		{"igmp://233.18.204.114:5140", "4K"},
		{"rtp://233.18.204.224:5140", "4K"},
		{"igmp://233.18.204.188:5140", "4K"},
		{"233.18.204.229:5140", "4K"},
		{"233.18.204.169", "4K"},

		// 央视
		{"igmp://233.18.204.20:5140", "央视"},
		{"igmp://233.18.204.52:5140", "央视"},
		{"igmp://233.18.204.176:5140", "央视"},
		{"rtp://233.18.204.210:5140", "央视"},

		// 卫视
		{"igmp://233.18.204.84:5140", "卫视"},
		{"igmp://233.18.204.100:5140", "卫视"},
		{"igmp://233.18.204.220:5140", "卫视"},

		// 高清
		{"igmp://233.18.204.173:5140", "高清"},
		{"igmp://233.18.204.44:5140", "高清"},
		{"igmp://233.18.204.231:5140", "高清"},

		// 本地
		{"igmp://233.18.204.1:5140", "本地"},
		{"igmp://233.18.204.6:5140", "本地"},
		{"igmp://233.18.204.57:5140", "本地"},

		// 少儿
		{"igmp://233.18.204.7:5140", "少儿"},
		{"igmp://233.18.204.151:5140", "少儿"},
		{"igmp://233.18.204.154:5140", "少儿"},

		// 标清
		{"igmp://233.18.204.12:5140", "标清"},
		{"igmp://233.18.204.118:5140", "标清"},

		// 待识别
		{"rtp://233.18.204.158:5140", "待识别"},
		{"rtp://233.18.204.201:5140", "待识别"},

		// Unknown IP outside Shanghai Telecom preset
		{"igmp://239.1.1.1:5140", ""},
		{"rtp://224.0.0.1:1234", ""},
		{"", ""},
	}

	for _, tc := range cases {
		got := PreclassifyMulticastGroup(tc.url)
		if got != tc.wantGroup {
			t.Errorf("PreclassifyMulticastGroup(%q) = %q; want %q", tc.url, got, tc.wantGroup)
		}
	}
}

func TestPreclassifyChannel(t *testing.T) {
	// 1. Scanned unknown 4K channel gets pre-classified
	c1 := Channel{
		URL:   "igmp://233.18.204.114:5140",
		Name:  "未知频道 233.18.204.114:5140",
		Group: "待识别",
	}
	if !PreclassifyChannel(&c1) {
		t.Errorf("expected PreclassifyChannel to return true for unknown 4K channel")
	}
	if c1.Group != "4K" {
		t.Errorf("expected Group '4K', got %q", c1.Group)
	}
	if c1.Name != "CCTV-16 4K" {
		t.Errorf("expected Name 'CCTV-16 4K', got %q", c1.Name)
	}
	if c1.Resolution != "3840x2160" {
		t.Errorf("expected Resolution '3840x2160', got %q", c1.Resolution)
	}

	// 2. Custom channel name and group are preserved
	c2 := Channel{
		URL:   "igmp://233.18.204.1:5140",
		Name:  "我的东方卫视",
		Group: "自定义分组",
	}
	_ = PreclassifyChannel(&c2)
	if c2.Name != "我的东方卫视" {
		t.Errorf("expected custom Name to be preserved, got %q", c2.Name)
	}
	if c2.Group != "自定义分组" {
		t.Errorf("expected custom Group to be preserved, got %q", c2.Group)
	}

	// 3. Channel with empty group gets pre-grouped
	c3 := Channel{
		URL:  "igmp://233.18.204.20:5140",
		Name: "CCTV-1",
	}
	if !PreclassifyChannel(&c3) {
		t.Errorf("expected PreclassifyChannel to return true for empty group")
	}
	if c3.Group != "央视" {
		t.Errorf("expected Group '央视', got %q", c3.Group)
	}

	// 4. nil channel
	if PreclassifyChannel(nil) {
		t.Errorf("expected false for nil channel")
	}

	// 5. Unknown channel with ip:port name is standardized to "未知频道"
	c5 := Channel{
		URL:   "rtp://233.18.204.158:5140",
		Name:  "未知频道 233.18.204.158:5140",
		Group: "待识别",
	}
	_ = PreclassifyChannel(&c5)
	if c5.Name != "未知频道" {
		t.Errorf("expected Name '未知频道', got %q", c5.Name)
	}
	if c5.Logo != "" {
		t.Errorf("expected empty Logo for unknown channel, got %q", c5.Logo)
	}

	// 6. CCTV-1HD and CCTV-1 share CCTV-1.png logo
	cCCTV := Channel{
		URL:  "igmp://233.18.204.52:5140",
		Name: "CCTV-1HD",
	}
	_ = PreclassifyChannel(&cCCTV)
	if cCCTV.Logo != "/logos/CCTV-1.png" {
		t.Errorf("expected Logo '/logos/CCTV-1.png', got %q", cCCTV.Logo)
	}

	// 7. 东方购物-1 and 东方购物-2 share 东方购物.png logo
	cShop1 := Channel{URL: "igmp://233.18.204.9:5140"}
	_ = PreclassifyChannel(&cShop1)
	if cShop1.Name != "东方购物-1" || cShop1.Logo != "/logos/东方购物.png" {
		t.Errorf("expected 东方购物-1 with /logos/东方购物.png, got name=%q logo=%q", cShop1.Name, cShop1.Logo)
	}

	cShop2 := Channel{URL: "igmp://233.18.204.10:5140"}
	_ = PreclassifyChannel(&cShop2)
	if cShop2.Name != "东方购物-2" || cShop2.Logo != "/logos/东方购物.png" {
		t.Errorf("expected 东方购物-2 with /logos/东方购物.png, got name=%q logo=%q", cShop2.Name, cShop2.Logo)
	}

	// 8. 卫视 and 卫视HD share logo, while 卫视 4K has distinct logo
	cZjHD := Channel{URL: "igmp://233.18.204.84:5140", Name: "浙江卫视HD"}
	_ = PreclassifyChannel(&cZjHD)
	if cZjHD.Logo != "/logos/浙江卫视.png" {
		t.Errorf("expected Logo '/logos/浙江卫视.png', got %q", cZjHD.Logo)
	}

	cZj4K := Channel{URL: "rtp://233.18.204.229:5140"}
	_ = PreclassifyChannel(&cZj4K)
	if cZj4K.Logo != "/logos/浙江卫视4K.png" {
		t.Errorf("expected Logo '/logos/浙江卫视4K.png', got %q", cZj4K.Logo)
	}

}

func TestDefaultPreGroups(t *testing.T) {
	groups := DefaultPreGroups()
	expected := []string{"4K", "央视", "卫视", "高清", "本地", "少儿", "标清", "其它", "待识别"}
	if len(groups) != len(expected) {
		t.Fatalf("expected %d groups, got %d", len(expected), len(groups))
	}
	for i, g := range groups {
		if g != expected[i] {
			t.Errorf("expected index %d group %q, got %q", i, expected[i], g)
		}
	}
}
