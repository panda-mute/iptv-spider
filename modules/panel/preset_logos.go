package panel

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed preset_logos/*.png
var presetLogosFS embed.FS

// EnsurePresetLogos copies embedded preset logos to s.LogosDir if not already initialized.
// It creates a .initialized sentinel file so this copy is performed only once.
func (s *Service) EnsurePresetLogos() error {
	dir := s.LogosDir
	if dir == "" {
		dir = "data/logos"
	}
	marker := filepath.Join(dir, ".initialized")
	if _, err := os.Stat(marker); err == nil {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	entries, err := fs.ReadDir(presetLogosFS, "preset_logos")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}
		targetPath := filepath.Join(dir, entry.Name())
		if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
			data, err := presetLogosFS.ReadFile("preset_logos/" + entry.Name())
			if err != nil {
				continue
			}
			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return os.WriteFile(marker, []byte("initialized\n"), 0644)
}

var (
	cctvRegex   = regexp.MustCompile(`(?i)^cctv[-_\s]*(\d+\+?)(?:[-_\s]*(?:hd|fhd|uhd|高清|标清))?$`)
	defSuffixRe = regexp.MustCompile(`(?i)[-_\s]*(?:HD|FHD|SD|高清|标清)$`)
	space4KRe   = regexp.MustCompile(`(?i)\s*4k`)
)

// FindLocalLogo searches for a matching logo in s.LogosDir or embedded presets.
// Returns "/logos/<filename>" if found, or empty string.
func (s *Service) FindLocalLogo(name string) string {
	if s == nil || s.LogosDir == "" {
		return ""
	}
	if s.Store != nil {
		if m, found := s.Store.FindMapping(name); found && m.TargetName != "" {
			name = m.TargetName
		}
	}
	return findLocalLogoInDir(s.LogosDir, name)
}

func findLocalLogoInDir(dir, name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "未知频道") {
		return ""
	}

	candidates := generateLocalLogoCandidates(name)
	for _, cand := range candidates {
		targetPath := filepath.Join(dir, cand)
		if fi, err := os.Stat(targetPath); err == nil && !fi.IsDir() {
			return "/logos/" + cand
		}
		if _, err := presetLogosFS.ReadFile("preset_logos/" + cand); err == nil {
			return "/logos/" + cand
		}
	}

	// Fallback to fuzzy key matching on files in dir
	targetKey := logoKey(name)
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
				continue
			}
			baseName := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			if logoKey(baseName) == targetKey {
				return "/logos/" + e.Name()
			}
		}
	}

	return ""
}

func generateLocalLogoCandidates(name string) []string {
	var candidates []string
	add := func(fn string) {
		fn = strings.TrimSpace(fn)
		if fn == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(fn), ".png") {
			fn += ".png"
		}
		for _, c := range candidates {
			if c == fn {
				return
			}
		}
		candidates = append(candidates, fn)
	}

	// 1. Known alias mappings from legacy data
	switch name {
	case "体育频道":
		add("五星体育.png")
	case "卡酷卡通":
		add("卡酷少儿.png")
	}

	// 2. Exact name
	add(name)

	// 3. 4K space variations (keep 4K distinct from HD!)
	if strings.Contains(strings.ToLower(name), "4k") {
		norm4K := space4KRe.ReplaceAllString(name, "4K")
		add(norm4K)
		withSpace := strings.ReplaceAll(norm4K, "4K", " 4K")
		add(withSpace)
	}

	// 4. 东方购物 prefix (东方购物-1 / 东方购物-2 share 东方购物.png)
	if strings.HasPrefix(name, "东方购物") {
		add("东方购物.png")
	}

	// 5. CCTV channel normalization (CCTV-1, CCTV-1HD, CCTV1HD -> CCTV-1.png)
	if m := cctvRegex.FindStringSubmatch(name); len(m) > 1 {
		add("CCTV-" + strings.ToUpper(m[1]) + ".png")
		add("CCTV" + strings.ToUpper(m[1]) + ".png")
	}

	// 6. Strip trailing HD/高清/标清 (while preserving 4K)
	if stripped := defSuffixRe.ReplaceAllString(name, ""); stripped != "" && stripped != name {
		add(stripped)
		if strings.Contains(strings.ToLower(stripped), "4k") {
			add(space4KRe.ReplaceAllString(stripped, "4K"))
		}
	}

	return candidates
}
