package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ffmpegVideoResRegex = regexp.MustCompile(`Stream #0:\d+.*?: Video: .*?, (\d{3,5})x(\d{3,5})`)
	resStringRegex      = regexp.MustCompile(`^(\d{3,5})[xX*](\d{3,5})$`)
)

// ExtractResolutionFromFFmpegStderr parses the input video stream resolution from ffmpeg log output.
func ExtractResolutionFromFFmpegStderr(stderr string) string {
	m := ffmpegVideoResRegex.FindStringSubmatch(stderr)
	if len(m) >= 3 {
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if w > 0 && h > 0 {
			return fmt.Sprintf("%dx%d", w, h)
		}
	}
	return ""
}

// ResolutionLabel returns a user-friendly classification label for a resolution string.
func ResolutionLabel(res string) string {
	res = strings.TrimSpace(res)
	if res == "" {
		return ""
	}
	m := resStringRegex.FindStringSubmatch(res)
	if len(m) < 3 {
		return res
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	if h >= 2160 || w >= 3840 {
		return "4K"
	}
	if h >= 1080 || w >= 1920 {
		return "1080P"
	}
	if h >= 720 || w >= 1280 {
		return "720P"
	}
	if h >= 576 {
		return "576P"
	}
	if h >= 480 {
		return "480P"
	}
	return res
}

// ResolutionScore calculates a numeric score based on pixel count for resolution comparison.
func ResolutionScore(res string) int {
	res = strings.TrimSpace(res)
	if res == "" {
		return 0
	}
	m := resStringRegex.FindStringSubmatch(res)
	if len(m) < 3 {
		return 0
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	return w * h
}

// CompareResolution compares two resolution strings. Higher resolution returns -1, equal returns 0, lower returns 1.
func CompareResolution(a, b string) int {
	scoreA := ResolutionScore(a)
	scoreB := ResolutionScore(b)
	if scoreA != scoreB {
		if scoreA > scoreB {
			return -1
		}
		return 1
	}
	return 0
}

type ffprobeJSONOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

// ProbeResolutionFromMPEGTS reads MPEG-TS data from reader and probes video resolution via ffprobe pipe.
func ProbeResolutionFromMPEGTS(ctx context.Context, r io.Reader) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-hide_banner",
		"-v", "error",
		"-analyzeduration", "6000000",
		"-probesize", "8000000",
		"-select_streams", "v",
		"-show_entries", "stream=width,height",
		"-of", "json",
		"pipe:0",
	)

	cmd.Stdin = io.LimitReader(r, 8<<20)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	var output ffprobeJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return "", err
	}
	for _, stream := range output.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			return fmt.Sprintf("%dx%d", stream.Width, stream.Height), nil
		}
	}
	return "", errors.New("未检测到有效视频流分辨率")
}

// ProbeResolutionFromURL probes video resolution directly from an HTTP stream URL using ffprobe,
// with an ffmpeg decode probe fallback for high-bitrate 4K streams.
func ProbeResolutionFromURL(ctx context.Context, uri string) (string, error) {
	if !httpURL(uri) {
		return "", errors.New("无效的 HTTP 流地址")
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	res, err := probeViaFFprobe(ctx, uri)
	if err == nil && res != "" {
		return res, nil
	}

	fallbackRes, fallbackErr := probeViaFFmpeg(ctx, uri)
	if fallbackErr == nil && fallbackRes != "" {
		return fallbackRes, nil
	}

	if err != nil {
		return "", err
	}
	return "", errors.New("未检测到有效视频流分辨率")
}

func probeViaFFprobe(ctx context.Context, uri string) (string, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-hide_banner",
		"-v", "error",
		"-protocol_whitelist", "http,https,tcp,tls,crypto",
		"-rw_timeout", "10000000",
		"-analyzeduration", "6000000",
		"-probesize", "8000000",
		"-select_streams", "v",
		"-show_entries", "stream=width,height",
		"-of", "json",
		uri,
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	var output ffprobeJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return "", err
	}
	for _, stream := range output.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			return fmt.Sprintf("%dx%d", stream.Width, stream.Height), nil
		}
	}
	return "", errors.New("未检测到有效视频流分辨率")
}

func probeViaFFmpeg(ctx context.Context, uri string) (string, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-nostdin",
		"-hide_banner",
		"-loglevel", "info",
		"-protocol_whitelist", "http,https,tcp,tls,crypto",
		"-rw_timeout", "10000000",
		"-analyzeduration", "6000000",
		"-probesize", "8000000",
		"-i", uri,
		"-frames:v", "1",
		"-an",
		"-f", "null",
		"-",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run()
	res := ExtractResolutionFromFFmpegStderr(stderr.String())
	if res != "" {
		return res, nil
	}
	return "", errors.New("未从 ffmpeg 日志检测到分辨率")
}
