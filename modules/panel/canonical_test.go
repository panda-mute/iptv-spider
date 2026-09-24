package panel

import "testing"

func TestCanonicalName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"cctv1", "CCTV-1"},
		{"CCTV1", "CCTV-1"},
		{"CCTV-1", "CCTV-1"},
		{"CCTV-1HD", "CCTV-1"},
		{"CCTV-1 高清", "CCTV-1"},
		{"CCTV-1 (高清)", "CCTV-1"},
		{"CCTV-4K", "CCTV-4K"},
		{"CCTV4K", "CCTV-4K"},
		{"CCTV-5+", "CCTV-5+"},
		{"CCTV-5+HD", "CCTV-5+"},
		{"湖南卫视", "湖南卫视"},
		{"湖南卫视高清", "湖南卫视"},
		{"湖南卫视HD", "湖南卫视"},
		{"湖南卫视4K", "湖南卫视"},
		{"湖南卫视 4K", "湖南卫视"},
		{"湖南卫视-4K超高清", "湖南卫视"},
		{"东方卫视", "东方卫视"},
		{"东方卫视HD", "东方卫视"},
		{"东方卫视高清", "东方卫视"},
		{"东方卫视 4K", "东方卫视"},
		{"浙江卫视(高清)", "浙江卫视"},
		{"广东卫视4K", "广东卫视"},
	}

	for _, tc := range cases {
		got := CanonicalName(tc.input)
		if got != tc.want {
			t.Errorf("CanonicalName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
