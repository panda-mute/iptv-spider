package panel

import (
	"testing"
)

func TestCCTVNumericSorting(t *testing.T) {
	channels := []Channel{
		{Name: "CCTV-16HD", Group: "央视频道", URL: "rtp://239.1.1.16:5140"},
		{Name: "CCTV-1HD", Group: "央视频道", URL: "rtp://239.1.1.1:5140"},
		{Name: "CCTV-2HD", Group: "央视频道", URL: "rtp://239.1.1.2:5140"},
		{Name: "CCTV-10", Group: "央视频道", URL: "rtp://239.1.1.10:5140"},
		{Name: "CCTV-5+HD", Group: "央视频道", URL: "rtp://239.1.1.55:5140"},
		{Name: "CCTV-5HD", Group: "央视频道", URL: "rtp://239.1.1.5:5140"},
		{Name: "CCTV-4K", Group: "央视频道", URL: "rtp://239.1.1.44:5140"},
		{Name: "CCTV-8K", Group: "央视频道", URL: "rtp://239.1.1.88:5140"},
		{Name: "CCTV-1", Group: "央视频道", URL: "rtp://239.1.1.101:5140"},
		{Name: "CCTV-17", Group: "央视频道", URL: "rtp://239.1.1.17:5140"},
		{Name: "CCTV-3", Group: "央视频道", URL: "rtp://239.1.1.3:5140"},
		{Name: "CCTV-风云足球", Group: "央视频道", URL: "rtp://239.1.1.99:5140"},
		{Name: "CCTV新闻", Group: "央视频道", URL: "rtp://239.1.1.13:5140"},
		{Name: "中央一台", Group: "央视频道", URL: "rtp://239.1.1.102:5140"},
		{Name: "CCTV奥林匹克", Group: "央视频道", URL: "rtp://239.1.1.161:5140"},
		{Name: "CCTV-兵器科技", Group: "央视频道", URL: "rtp://239.1.1.98:5140"},
	}

	SortChannels(channels, DefaultGroupOrder())

	// Collect names
	names := make([]string, len(channels))
	for i, c := range channels {
		names[i] = c.Name
	}

	indexOf := func(name string) int {
		for i, n := range names {
			if n == name {
				return i
			}
		}
		return -1
	}

	// Verify numeric ordering
	// 1. CCTV-1 / CCTV-1HD / 中央一台 must precede CCTV-2
	if indexOf("CCTV-1") >= indexOf("CCTV-2HD") || indexOf("CCTV-1HD") >= indexOf("CCTV-2HD") {
		t.Errorf("CCTV-1 should come before CCTV-2: %v", names)
	}

	// 2. CCTV-2HD before CCTV-3
	if indexOf("CCTV-2HD") >= indexOf("CCTV-3") {
		t.Errorf("CCTV-2HD should come before CCTV-3: %v", names)
	}

	// 3. CCTV-5HD before CCTV-5+HD
	if indexOf("CCTV-5HD") >= indexOf("CCTV-5+HD") {
		t.Errorf("CCTV-5HD should come before CCTV-5+HD: %v", names)
	}

	// 4. CCTV-5+HD before CCTV-10
	if indexOf("CCTV-5+HD") >= indexOf("CCTV-10") {
		t.Errorf("CCTV-5+HD should come before CCTV-10: %v", names)
	}

	// 5. CCTV-10 before CCTV新闻 (13)
	if indexOf("CCTV-10") >= indexOf("CCTV新闻") {
		t.Errorf("CCTV-10 should come before CCTV新闻: %v", names)
	}

	// 6. CCTV新闻 before CCTV-16HD and CCTV奥林匹克
	if indexOf("CCTV新闻") >= indexOf("CCTV-16HD") || indexOf("CCTV新闻") >= indexOf("CCTV奥林匹克") {
		t.Errorf("CCTV新闻 should come before CCTV-16: %v", names)
	}

	// 7. CCTV-16 before CCTV-17
	if indexOf("CCTV-16HD") >= indexOf("CCTV-17") {
		t.Errorf("CCTV-16HD should come before CCTV-17: %v", names)
	}

	// 8. CCTV-17 before CCTV-4K
	if indexOf("CCTV-17") >= indexOf("CCTV-4K") {
		t.Errorf("CCTV-17 should come before CCTV-4K: %v", names)
	}

	// 9. CCTV-4K before CCTV-8K
	if indexOf("CCTV-4K") >= indexOf("CCTV-8K") {
		t.Errorf("CCTV-4K should come before CCTV-8K: %v", names)
	}

	// 10. CCTV-8K before non-numbered specialized channels (风云足球, 兵器科技)
	if indexOf("CCTV-8K") >= indexOf("CCTV-风云足球") || indexOf("CCTV-8K") >= indexOf("CCTV-兵器科技") {
		t.Errorf("CCTV-8K should come before unnumbered CCTV: %v", names)
	}
}

func TestGroupSortingDefault(t *testing.T) {
	channels := []Channel{
		{Name: "未知频道 1", Group: "待识别", URL: "rtp://239.1.1.1:5140"},
		{Name: "东方卫视", Group: "卫视", URL: "rtp://239.1.1.2:5140"},
		{Name: "上海新闻综合", Group: "本地", URL: "rtp://239.1.1.3:5140"},
		{Name: "CCTV-1", Group: "央视", URL: "rtp://239.1.1.4:5140"},
		{Name: "CHC家庭影院", Group: "高清", URL: "rtp://239.1.1.5:5140"},
		{Name: "CCTV-4K", Group: "4K", URL: "rtp://239.1.1.6:5140"},
		{Name: "卡酷少儿", Group: "少儿", URL: "rtp://239.1.1.7:5140"},
		{Name: "法治天地", Group: "标清", URL: "rtp://239.1.1.8:5140"},
		{Name: "测试频道", Group: "其它", URL: "rtp://239.1.1.9:5140"},
	}

	SortChannels(channels, DefaultGroupOrder())

	expectedGroups := []string{"4K", "央视", "卫视", "高清", "本地", "少儿", "标清", "其它", "待识别"}
	for i, c := range channels {
		if c.Group != expectedGroups[i] {
			t.Errorf("expected index %d group %q, got %q (channel %s)", i, expectedGroups[i], c.Group, c.Name)
		}
	}
}

func TestGroupSortingLegacy(t *testing.T) {
	channels := []Channel{
		{Name: "未知频道 1", Group: "待识别", URL: "rtp://239.1.1.1:5140"},
		{Name: "东方卫视", Group: "卫视频道", URL: "rtp://239.1.1.2:5140"},
		{Name: "上海新闻综合", Group: "上海频道", URL: "rtp://239.1.1.3:5140"},
		{Name: "CCTV-1", Group: "央视频道", URL: "rtp://239.1.1.4:5140"},
		{Name: "CHC家庭影院", Group: "数字频道", URL: "rtp://239.1.1.5:5140"},
		{Name: "测试频道", Group: "其它", URL: "rtp://239.1.1.6:5140"},
	}

	legacyOrder := []string{"央视频道", "卫视频道", "上海频道", "数字频道", "其它", "待识别"}
	SortChannels(channels, legacyOrder)

	expectedGroups := []string{"央视频道", "卫视频道", "上海频道", "数字频道", "其它", "待识别"}
	for i, c := range channels {
		if c.Group != expectedGroups[i] {
			t.Errorf("expected index %d group %q, got %q (channel %s)", i, expectedGroups[i], c.Group, c.Name)
		}
	}
}

func TestGroupSortingCustom(t *testing.T) {
	channels := []Channel{
		{Name: "CCTV-1", Group: "央视频道", URL: "rtp://239.1.1.1:5140"},
		{Name: "东方卫视", Group: "卫视频道", URL: "rtp://239.1.1.2:5140"},
		{Name: "上海新闻综合", Group: "上海频道", URL: "rtp://239.1.1.3:5140"},
	}

	customOrder := []string{"上海频道", "央视频道", "卫视频道"}
	SortChannels(channels, customOrder)

	expectedGroups := []string{"上海频道", "央视频道", "卫视频道"}
	for i, c := range channels {
		if c.Group != expectedGroups[i] {
			t.Errorf("expected index %d group %q, got %q (channel %s)", i, expectedGroups[i], c.Group, c.Name)
		}
	}
}

func TestCompareNatural(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"CETV-1", "CETV-2", -1},
		{"CETV-2", "CETV-10", -1},
		{"CETV-10", "CETV-10", 0},
		{"上海-1", "上海-2", -1},
		{"上海-2", "上海-10", -1},
	}

	for _, tc := range cases {
		got := CompareNatural(tc.a, tc.b)
		if (tc.want < 0 && got >= 0) || (tc.want > 0 && got <= 0) || (tc.want == 0 && got != 0) {
			t.Errorf("CompareNatural(%q, %q) = %d; want sign %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestInGroupChannelSorting(t *testing.T) {
	channels := []Channel{
		{Key: "c-1", ID: "1", Name: "东方卫视", Group: "卫视频道", URL: "rtp://239.1.1.1:5140"},
		{Key: "c-2", ID: "2", Name: "湖南卫视", Group: "卫视频道", URL: "rtp://239.1.1.2:5140"},
		{Key: "c-3", ID: "3", Name: "浙江卫视", Group: "卫视频道", URL: "rtp://239.1.1.3:5140"},
		{Key: "c-4", ID: "4", Name: "江苏卫视", Group: "卫视频道", URL: "rtp://239.1.1.4:5140"},
	}

	// Custom order: 湖南卫视 (c-2) -> 江苏卫视 (c-4) -> 东方卫视 (c-1) -> unlisted (浙江卫视 c-3)
	inGroupOrder := map[string][]string{
		"卫视频道": {"c-2", "c-4", "c-1"},
	}

	SortChannels(channels, DefaultGroupOrder(), inGroupOrder)

	expectedOrder := []string{"湖南卫视", "江苏卫视", "东方卫视", "浙江卫视"}
	for i, c := range channels {
		if c.Name != expectedOrder[i] {
			t.Errorf("expected index %d to be %q, got %q", i, expectedOrder[i], c.Name)
		}
	}
}
