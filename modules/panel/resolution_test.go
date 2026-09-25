package panel

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestResolutionLabelAndScore(t *testing.T) {
	cases := []struct {
		res       string
		wantLabel string
		wantScore int
	}{
		{"3840x2160", "4K", 3840 * 2160},
		{"1920x1080", "1080P", 1920 * 1080},
		{"1280x720", "720P", 1280 * 720},
		{"720x576", "576P", 720 * 576},
		{"640x480", "480P", 640 * 480},
	}

	for _, tc := range cases {
		label := ResolutionLabel(tc.res)
		if label != tc.wantLabel {
			t.Errorf("ResolutionLabel(%q) = %q, want %q", tc.res, label, tc.wantLabel)
		}
		score := ResolutionScore(tc.res)
		if score != tc.wantScore {
			t.Errorf("ResolutionScore(%q) = %d, want %d", tc.res, score, tc.wantScore)
		}
	}

	if CompareResolution("3840x2160", "1920x1080") >= 0 {
		t.Error("expected 4K to compare higher than 1080P")
	}
	if CompareResolution("1920x1080", "3840x2160") <= 0 {
		t.Error("expected 1080P to compare lower than 4K")
	}
	if CompareResolution("1920x1080", "1920x1080") != 0 {
		t.Error("expected identical resolutions to compare equal")
	}
}

func TestExtractResolutionFromFFmpegStderr(t *testing.T) {
	log := `
Input #0, mpegts, from 'http://192.168.1.1:4022/rtp/239.1.1.1:5140':
  Duration: N/A, start: 1.400000, bitrate: N/A
  Stream #0:0[0x100]: Video: h264 (High) ([27][0][0][0] / 0x001B), yuv420p(tv, bt709, progressive), 1920x1080 [SAR 1:1 DAR 16:9], 25 fps, 50 tbr, 90k tbn
  Stream #0:1[0x101]: Audio: mp2 ([3][0][0][0] / 0x0003), 48000 Hz, stereo, s16p, 192 kb/s
`
	res := ExtractResolutionFromFFmpegStderr(log)
	if res != "1920x1080" {
		t.Errorf("ExtractResolutionFromFFmpegStderr got %q, want 1920x1080", res)
	}

	log4K := `Stream #0:1: Video: hevc, yuv420p10le(tv), 3840x2160 [SAR 1:1 DAR 16:9], 50 fps`
	res4K := ExtractResolutionFromFFmpegStderr(log4K)
	if res4K != "3840x2160" {
		t.Errorf("ExtractResolutionFromFFmpegStderr 4K got %q, want 3840x2160", res4K)
	}
	log4KReal := "Stream #0:0[0x1022]: Video: hevc (Main 10) ([36][0][0][0] / 0x0024), yuv420p10le(tv, bt2020nc/bt2020/arib-std-b67), 3840x2160 [SAR 1:1 DAR 16:9], 50 fps, 50 tbr, 90k tbn, start 1820.831433"
	if res4KReal := ExtractResolutionFromFFmpegStderr(log4KReal); res4KReal != "3840x2160" {
		t.Errorf("ExtractResolutionFromFFmpegStderr 4K real got %q, want 3840x2160", res4KReal)
	}

	noVideo := `Stream #0:0: Audio: aac, 44100 Hz, stereo`
	if resEmpty := ExtractResolutionFromFFmpegStderr(noVideo); resEmpty != "" {
		t.Errorf("ExtractResolutionFromFFmpegStderr no video got %q, want empty", resEmpty)
	}
}

func TestProbeResolutionFromMPEGTSPipe(t *testing.T) {
	// Generate a short 1-frame test MPEG-TS stream and pipe into ProbeResolutionFromMPEGTS
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-loglevel", "quiet", "-f", "lavfi", "-i", "testsrc=size=1920x1080:rate=25", "-t", "0.2", "-f", "mpegts", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Wait() }()

	res, err := ProbeResolutionFromMPEGTS(ctx, stdout)
	if err != nil {
		t.Fatalf("ProbeResolutionFromMPEGTS failed: %v", err)
	}
	if res != "1920x1080" {
		t.Errorf("ProbeResolutionFromMPEGTS got %q, want 1920x1080", res)
	}
}

func TestRealProbe4K(t *testing.T) {
	if os.Getenv("TEST_LIVE_STREAM") != "1" {
		t.Skip("skipping live probe test without TEST_LIVE_STREAM=1")
	}
	urls := map[string]string{
		"东方卫视4K": "http://192.168.190.1:4022/rtp/233.18.204.224:5140?fcc=124.75.26.151%3A15970",
		"江苏卫视4K": "http://192.168.190.1:4022/rtp/233.18.204.226:5140?fcc=124.75.26.151%3A15970",
		"湖南卫视4K": "http://192.168.190.1:4022/rtp/233.18.204.227:5140?fcc=124.75.26.151%3A15970",
		"山东卫视4K": "http://192.168.190.1:4022/rtp/233.18.204.228:5140?fcc=124.75.26.151%3A15970",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for name, u := range urls {
		t.Run(name, func(t *testing.T) {
			res, err := ProbeResolutionFromURL(ctx, u)
			if err != nil {
				t.Fatalf("%s probe failed: %v", name, err)
			}
			t.Logf("%s resolved resolution: %s (label: %s)", name, res, ResolutionLabel(res))
			if res != "3840x2160" {
				t.Errorf("%s expected 3840x2160, got %s", name, res)
			}
		})
	}
}
