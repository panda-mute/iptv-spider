package panel

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func isLocalLogo(s string) bool {
	if strings.ContainsAny(s, "\r\n") {
		return false
	}
	return strings.HasPrefix(s, "/logos/") || strings.HasPrefix(s, "/api/panel/logos/local/")
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || (r >= 0x4e00 && r <= 0x9fa5) {
			b.WriteRune(r)
			lastUnderscore = false
		} else if r == '_' || r == ' ' {
			if !lastUnderscore && b.Len() > 0 {
				b.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	res := strings.Trim(b.String(), "_")
	if len(res) > 50 {
		res = res[:50]
	}
	return res
}

func (s *Service) DownloadLogo(ctx context.Context, rawURL, name string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("台标 URL 不能为空")
	}
	if isLocalLogo(rawURL) {
		return rawURL, nil
	}
	if !httpURL(rawURL) {
		return "", errors.New("无效的台标 HTTP(S) 地址")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载台标失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载台标返回 HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("读取台标内容失败: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("台标文件内容为空")
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	sniffed := http.DetectContentType(data)
	ext := ".png"
	u, _ := url.Parse(rawURL)
	urlExt := strings.ToLower(filepath.Ext(u.Path))
	switch {
	case strings.Contains(contentType, "svg") || strings.Contains(sniffed, "svg") || urlExt == ".svg":
		ext = ".svg"
	case strings.Contains(contentType, "webp") || strings.Contains(sniffed, "webp") || urlExt == ".webp":
		ext = ".webp"
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") || strings.Contains(sniffed, "jpeg") || urlExt == ".jpg" || urlExt == ".jpeg":
		ext = ".jpg"
	case strings.Contains(contentType, "gif") || strings.Contains(sniffed, "gif") || urlExt == ".gif":
		ext = ".gif"
	case strings.Contains(contentType, "icon") || urlExt == ".ico":
		ext = ".ico"
	default:
		ext = ".png"
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(rawURL)))[:8]
	safeName := sanitizeFilename(name)
	if safeName == "" {
		safeName = "logo"
	}
	filename := fmt.Sprintf("%s_%s%s", safeName, hash, ext)

	dir := s.LogosDir
	if dir == "" {
		dir = "data/logos"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建台标存储目录: %w", err)
	}

	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存台标文件: %w", err)
	}

	return "/logos/" + filename, nil
}

func (s *Service) ServeLocalLogo(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if file == "" {
		file = r.PathValue("path")
	}
	if file == "" {
		path := r.URL.Path
		if strings.HasPrefix(path, "/logos/") {
			file = strings.TrimPrefix(path, "/logos/")
		} else if strings.HasPrefix(path, "/api/panel/logos/local/") {
			file = strings.TrimPrefix(path, "/api/panel/logos/local/")
		}
	}
	file = filepath.Base(file)
	if file == "." || file == "/" || file == "" {
		http.NotFound(w, r)
		return
	}
	dir := s.LogosDir
	if dir == "" {
		dir = "data/logos"
	}
	fullPath := filepath.Join(dir, file)
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		if data, readErr := presetLogosFS.ReadFile("preset_logos/" + file); readErr == nil {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, fullPath)
}

// SaveUploadedLogo saves uploaded image data into local logos storage and returns its local URL path.
func (s *Service) SaveUploadedLogo(name string, originalFilename string, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, 10<<20))
	if err != nil {
		return "", fmt.Errorf("读取上传内容失败: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("上传文件内容为空")
	}

	sniffed := http.DetectContentType(data)
	urlExt := strings.ToLower(filepath.Ext(originalFilename))
	ext := ".png"
	prefixLen := len(data)
	if prefixLen > 512 {
		prefixLen = 512
	}
	trimmed := strings.TrimSpace(string(data[:prefixLen]))
	switch {
	case strings.Contains(sniffed, "svg") || urlExt == ".svg" || strings.HasPrefix(trimmed, "<svg") || strings.HasPrefix(trimmed, "<?xml"):
		ext = ".svg"
	case strings.Contains(sniffed, "webp") || urlExt == ".webp":
		ext = ".webp"
	case strings.Contains(sniffed, "jpeg") || urlExt == ".jpg" || urlExt == ".jpeg":
		ext = ".jpg"
	case strings.Contains(sniffed, "gif") || urlExt == ".gif":
		ext = ".gif"
	case strings.Contains(sniffed, "icon") || urlExt == ".ico":
		ext = ".ico"
	case strings.Contains(sniffed, "png") || urlExt == ".png":
		ext = ".png"
	default:
		if !strings.HasPrefix(sniffed, "image/") && urlExt != ".svg" {
			return "", errors.New("仅支持上传常见图片文件格式（PNG、JPG、WEBP、SVG、GIF、ICO）")
		}
		ext = ".png"
	}

	hash := fmt.Sprintf("%x", md5.Sum(data))[:8]
	safeName := sanitizeFilename(name)
	if safeName == "" {
		safeName = sanitizeFilename(strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename)))
	}
	if safeName == "" {
		safeName = "logo"
	}
	outFilename := fmt.Sprintf("%s_%s%s", safeName, hash, ext)

	dir := s.LogosDir
	if dir == "" {
		dir = "data/logos"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建台标存储目录: %w", err)
	}

	filePath := filepath.Join(dir, outFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存台标文件: %w", err)
	}

	return "/logos/" + outFilename, nil
}
