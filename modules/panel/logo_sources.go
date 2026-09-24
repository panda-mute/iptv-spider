package panel

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const PrimaryLogoSource = "https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo"

//go:embed logos_primary.json
var primaryLogoPaths []byte

type LogoSources struct {
	Sources   []string `json:"sources,omitempty"`
	Primary   string   `json:"primary,omitempty"`
	Secondary string   `json:"secondary,omitempty"`
}

func defaultLogoSources() LogoSources {
	return LogoSources{
		Sources: []string{PrimaryLogoSource},
	}
}

func (v LogoSources) GetSources() []string {
	if len(v.Sources) > 0 {
		var out []string
		for _, s := range v.Sources {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if v.Primary != "" {
		return []string{strings.TrimSpace(v.Primary)}
	}
	return []string{PrimaryLogoSource}
}

func (v LogoSources) validate() error {
	for _, src := range v.GetSources() {
		if src != "" && (!httpURL(src) || strings.ContainsAny(src, "?\r\n")) {
			return errors.New("台标源应为 GitHub 目录 URL 或提供 /api/list 的 HTTP(S) 台标库地址")
		}
	}
	return nil
}

var provincePrefixes = []string{
	"上海", "北京", "广东", "江苏", "浙江", "山东", "湖南", "湖北", "四川", "天津",
	"重庆", "安徽", "河北", "河南", "福建", "江西", "辽宁", "吉林", "黑龙江", "陕西",
	"山西", "云南", "贵州", "广西", "海南", "内蒙古", "宁夏", "新疆", "青海", "西藏",
	"香港", "台湾", "深圳",
}

func isDarkLogoPath(p string) bool {
	lower := strings.ToLower(p)
	return strings.Contains(p, "深色") || strings.Contains(lower, "dark")
}

func parseGithubLogos(paths []string, repo, ref, dir string) (map[string]string, map[string]string) {
	sort.Slice(paths, func(i, j int) bool {
		iDark := isDarkLogoPath(paths[i])
		jDark := isDarkLogoPath(paths[j])
		if iDark != jDark {
			return !iDark
		}
		if strings.Count(paths[i], "/") != strings.Count(paths[j], "/") {
			return strings.Count(paths[i], "/") < strings.Count(paths[j], "/")
		}
		return paths[i] < paths[j]
	})
	norm := map[string]string{}
	dark := map[string]string{}
	for _, p := range paths {
		if !strings.HasPrefix(p, strings.TrimRight(dir, "/")+"/") {
			continue
		}
		ext := strings.ToLower(path.Ext(p))
		if ext != ".png" && ext != ".webp" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".svg" {
			continue
		}
		rawKey := strings.TrimSuffix(path.Base(p), path.Ext(p))
		key := logoKey(rawKey)
		if key == "" {
			continue
		}
		pieces := strings.Split(p, "/")
		for i := range pieces {
			pieces[i] = url.PathEscape(pieces[i])
		}
		u := "https://raw.githubusercontent.com/" + repo + "/" + url.PathEscape(ref) + "/" + strings.Join(pieces, "/")
		if isDarkLogoPath(p) {
			if dark[key] == "" {
				dark[key] = u
			}
			for _, prov := range provincePrefixes {
				if strings.HasPrefix(key, prov) {
					stripped := strings.TrimPrefix(key, prov)
					if len([]rune(stripped)) >= 2 && dark[stripped] == "" {
						dark[stripped] = u
					}
					break
				}
			}
		} else {
			if norm[key] == "" {
				norm[key] = u
			}
			for _, prov := range provincePrefixes {
				if strings.HasPrefix(key, prov) {
					stripped := strings.TrimPrefix(key, prov)
					if len([]rune(stripped)) >= 2 && norm[stripped] == "" {
						norm[stripped] = u
					}
					break
				}
			}
		}
	}
	return norm, dark
}

var sourceCatalogs = struct {
	sync.RWMutex
	data map[string]map[string]string
	dark map[string]map[string]string
}{
	data: map[string]map[string]string{PrimaryLogoSource: primaryCatalog()},
	dark: map[string]map[string]string{PrimaryLogoSource: primaryDarkCatalog()},
}

func primaryCatalog() map[string]string {
	var paths []string
	_ = json.Unmarshal(primaryLogoPaths, &paths)
	norm, _ := parseGithubLogos(paths, "sggc/SDU-IPTV-PRO", "main", "logo")
	return norm
}

func primaryDarkCatalog() map[string]string {
	var paths []string
	_ = json.Unmarshal(primaryLogoPaths, &paths)
	_, dark := parseGithubLogos(paths, "sggc/SDU-IPTV-PRO", "main", "logo")
	return dark
}

func githubLogos(paths []string, repo, ref, dir string) map[string]string {
	norm, dark := parseGithubLogos(paths, repo, ref, dir)
	out := make(map[string]string, len(norm)+len(dark))
	for k, v := range dark {
		out[k] = v
	}
	for k, v := range norm {
		out[k] = v
	}
	return out
}
func sourceKey(src string) string { return strings.TrimRight(strings.TrimSpace(src), "/") }
func logoRegion(name, region string) string {
	return ""
}
func logoCandidates(cfg LogoSources, name, _ string) map[string]string {
	sources := cfg.GetSources()
	sourceCatalogs.RLock()
	defer sourceCatalogs.RUnlock()
	out := map[string]string{}
	// Lower priority: dark logos from later sources to earlier sources
	for i := len(sources) - 1; i >= 0; i-- {
		src := sourceKey(sources[i])
		for k, v := range sourceCatalogs.dark[src] {
			out[k] = v
		}
	}
	// Higher priority: normal logos from later sources to earlier sources (earlier overwrite later)
	for i := len(sources) - 1; i >= 0; i-- {
		src := sourceKey(sources[i])
		for k, v := range sourceCatalogs.data[src] {
			out[k] = v
		}
	}
	return out
}
func lookupLogo(catalog map[string]string, name string) string {
	if len(catalog) == 0 {
		return ""
	}
	key := logoKey(name)
	if v := catalog[key]; v != "" {
		return v
	}
	aliases := map[string]string{
		"体育频道":   "五星体育",
		"中国教育1":  "CETV1",
		"中国教育2":  "CETV2",
		"中国教育4":  "CETV4",
		"卡酷少儿":   "卡酷卡通",
		"高尔夫网球":  "CCTV央视高网",
		"央视文化精品": "CCTV文化精品",
		"新闻综合":   "上海新闻综合",
		"都市频道":   "上海都市频道",
		"第一财经":   "上海第一财经",
		"五星体育":   "上海五星体育",
		"东方影视":   "上海东方影视",
		"纪实人文":   "上海纪实人文",
		"上海纪实":   "上海纪实人文",
		"七彩戏剧":   "上海七彩戏剧",
		"东方财经":   "上海东方财经",
		"风云足球":   "CCTV风云足球",
		"风云剧场":   "CCTV风云剧场",
		"风云音乐":   "CCTV风云音乐",
		"兵器科技":   "CCTV兵器科技",
		"第一剧场":   "CCTV第一剧场",
		"央视台球":   "CCTV央视台球",
		"卫生健康":   "CCTV卫生健康",
		"世界地理":   "CCTV世界地理",
		"文化精品":   "CCTV文化精品",
	}
	if alias := aliases[key]; alias != "" {
		if v := catalog[alias]; v != "" {
			return v
		}
	}
	if v := catalog["上海"+key]; v != "" {
		return v
	}
	return catalog["CCTV"+key]
}

func configuredLogo(cfg LogoSources, name, _ string) string {
	sources := cfg.GetSources()
	sourceCatalogs.RLock()
	defer sourceCatalogs.RUnlock()

	// 1. Normal logos have highest priority across all sources in order
	for _, src := range sources {
		if v := lookupLogo(sourceCatalogs.data[sourceKey(src)], name); v != "" {
			return v
		}
	}
	// 2. Dark logos only matched as lower priority fallback across all sources in order
	for _, src := range sources {
		if v := lookupLogo(sourceCatalogs.dark[sourceKey(src)], name); v != "" {
			return v
		}
	}
	return ""
}
func getLogoJSON(ctx context.Context, uri string, v any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "iptv-spider")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("台标源 HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if err != nil {
		return err
	}
	if len(b) > 8<<20 {
		return errors.New("台标目录过大")
	}
	return json.Unmarshal(b, v)
}
func refreshSource(ctx context.Context, src string) error {
	src = sourceKey(src)
	if src == "" {
		return nil
	}
	u, _ := url.Parse(src)
	var names, darkNames map[string]string
	if u.Hostname() == "github.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 5 || parts[2] != "tree" {
			return errors.New("GitHub 源应为 /用户/仓库/tree/分支/目录")
		}
		repo, ref, dir := strings.Join(parts[:2], "/"), parts[3], strings.Join(parts[4:], "/")
		var tree struct {
			Truncated bool `json:"truncated"`
			Tree      []struct {
				Path string `json:"path"`
				Type string `json:"type"`
			} `json:"tree"`
		}
		if err := getLogoJSON(ctx, "https://api.github.com/repos/"+repo+"/git/trees/"+url.PathEscape(ref)+"?recursive=1", &tree); err != nil {
			return err
		}
		if tree.Truncated {
			return errors.New("GitHub 目录被截断，保留现有索引")
		}
		paths := []string{}
		for _, v := range tree.Tree {
			if v.Type == "blob" {
				paths = append(paths, v.Path)
			}
		}
		names, darkNames = parseGithubLogos(paths, repo, ref, dir)
	} else {
		var data json.RawMessage
		if err := getLogoJSON(ctx, src+"/api/list", &data); err != nil {
			return err
		}
		names = parseLogoCatalog(data)
		for k, v := range names {
			names[k] = src + strings.TrimPrefix(v, LogoSource)
		}
	}
	if len(names) == 0 && len(darkNames) == 0 {
		return errors.New("没有读取到台标图片，保留现有索引")
	}
	sourceCatalogs.Lock()
	sourceCatalogs.data[src] = names
	if sourceCatalogs.dark == nil {
		sourceCatalogs.dark = map[string]map[string]string{}
	}
	sourceCatalogs.dark[src] = darkNames
	sourceCatalogs.Unlock()
	return nil
}
func (s *Service) LogoSourcesStatus() map[string]any {
	cfg, _ := s.Store.Snapshot()
	sources := cfg.Logos.GetSources()
	sourceCatalogs.RLock()
	defer sourceCatalogs.RUnlock()

	type sourceInfo struct {
		URL       string `json:"url"`
		Count     int    `json:"count"`
		DarkCount int    `json:"dark_count"`
	}
	list := make([]sourceInfo, 0, len(sources))
	total := 0
	for _, src := range sources {
		k := sourceKey(src)
		cnt := len(sourceCatalogs.data[k])
		darkCnt := len(sourceCatalogs.dark[k])
		total += cnt + darkCnt
		list = append(list, sourceInfo{
			URL:       src,
			Count:     cnt,
			DarkCount: darkCnt,
		})
	}
	primarySrc := PrimaryLogoSource
	if cfg.Logos.Primary != "" {
		primarySrc = cfg.Logos.Primary
	}
	return map[string]any{
		"sources":         list,
		"total_count":     total,
		"primary_count":   len(sourceCatalogs.data[sourceKey(primarySrc)]),
		"secondary_count": 0,
	}
}

func (s *Service) RefreshLogoSources(ctx context.Context) map[string]any {
	cfg, _ := s.Store.Snapshot()
	sources := cfg.Logos.GetSources()
	errs := []string{}
	for _, src := range sources {
		if err := refreshSource(ctx, src); err != nil {
			errs = append(errs, sourceKey(src)+": "+err.Error())
		}
	}
	if err := s.saveLogoSources(); err != nil {
		errs = append(errs, "保存台标目录失败: "+err.Error())
	}
	result := s.LogoSourcesStatus()
	result["errors"] = errs
	return result
}

// Keep custom source indexes usable across restarts and temporary source outages.
func (s *Service) LoadLogoSources() error {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(s.Store.path), "logo-catalogs.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var wrapped struct {
		Data map[string]map[string]string `json:"data"`
		Dark map[string]map[string]string `json:"dark"`
	}
	sourceCatalogs.Lock()
	defer sourceCatalogs.Unlock()
	if err = json.Unmarshal(b, &wrapped); err == nil && (len(wrapped.Data) > 0 || len(wrapped.Dark) > 0) {
		for src, names := range wrapped.Data {
			valid := map[string]string{}
			for k, v := range names {
				if httpURL(v) {
					valid[k] = v
				}
			}
			if len(valid) > 0 {
				sourceCatalogs.data[src] = valid
			}
		}
		for src, names := range wrapped.Dark {
			valid := map[string]string{}
			for k, v := range names {
				if httpURL(v) {
					valid[k] = v
				}
			}
			if len(valid) > 0 {
				if sourceCatalogs.dark == nil {
					sourceCatalogs.dark = map[string]map[string]string{}
				}
				sourceCatalogs.dark[src] = valid
			}
		}
		return nil
	}

	var legacy map[string]map[string]string
	if err = json.Unmarshal(b, &legacy); err == nil {
		for src, names := range legacy {
			valid := map[string]string{}
			for k, v := range names {
				if httpURL(v) {
					valid[k] = v
				}
			}
			if len(valid) > 0 {
				sourceCatalogs.data[src] = valid
			}
		}
	}
	return nil
}

func (s *Service) saveLogoSources() error {
	cfg, _ := s.Store.Snapshot()
	sources := cfg.Logos.GetSources()
	sourceCatalogs.RLock()
	data := map[string]map[string]string{}
	dark := map[string]map[string]string{}
	for _, src := range sources {
		k := sourceKey(src)
		if k != "" {
			if m, ok := sourceCatalogs.data[k]; ok {
				data[k] = m
			}
			if m, ok := sourceCatalogs.dark[k]; ok {
				dark[k] = m
			}
		}
	}
	sourceCatalogs.RUnlock()

	wrapped := struct {
		Data map[string]map[string]string `json:"data"`
		Dark map[string]map[string]string `json:"dark"`
	}{
		Data: data,
		Dark: dark,
	}
	b, err := json.Marshal(wrapped)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.Store.path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".logos-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), filepath.Join(dir, "logo-catalogs.json"))
}
