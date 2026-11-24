package panel

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"path"
	"sort"
	"strings"
)

// Snapshot of tvlogo.netioe.com's index.json, retaining only matching fields.
//
//go:embed logos_default.json
var defaultLogoIndex []byte

func defaultIndexCatalog(dark bool) map[string]string {
	normal, darker, _ := parseIndexCatalog(defaultLogoIndex, PrimaryLogoSource)
	if dark {
		return darker
	}
	return normal
}

// parseIndexCatalog reads the name/path/variant catalog served by TV Logo.
// URLs always point to the selected source, rather than its upstream mirrors.
func parseIndexCatalog(data []byte, source string) (map[string]string, map[string]string, error) {
	var entries []struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Variant bool   `json:"variant"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		// Prefer domestic channel artwork when normalized names collide.
		iDomestic := strings.HasPrefix(entries[i].Path, "logo/国内/")
		jDomestic := strings.HasPrefix(entries[j].Path, "logo/国内/")
		if iDomestic != jDomestic {
			return iDomestic
		}
		if entries[i].Variant != entries[j].Variant {
			return !entries[i].Variant
		}
		return entries[i].Path < entries[j].Path
	})
	normal, dark := map[string]string{}, map[string]string{}
	for _, entry := range entries {
		key := logoKey(entry.Name)
		if key == "" || !strings.HasPrefix(entry.Path, "logo/") || strings.ContainsAny(entry.Path, "\\?#\r\n") {
			continue
		}
		switch strings.ToLower(path.Ext(entry.Path)) {
		case ".png", ".webp", ".jpg", ".jpeg", ".gif", ".svg":
		default:
			continue
		}
		pieces := strings.Split(entry.Path, "/")
		valid := true
		for i, piece := range pieces {
			if piece == "" || piece == "." || piece == ".." {
				valid = false
				break
			}
			pieces[i] = url.PathEscape(piece)
		}
		if !valid {
			continue
		}
		uri := sourceKey(source) + "/" + strings.Join(pieces, "/")
		catalog := normal
		if isDarkLogoPath(entry.Path) {
			catalog = dark
		}
		if catalog[key] == "" {
			catalog[key] = uri
		}
		for _, province := range provincePrefixes {
			if strings.HasPrefix(key, province) {
				alias := strings.TrimPrefix(key, province)
				if len([]rune(alias)) >= 2 && catalog[alias] == "" {
					catalog[alias] = uri
				}
				break
			}
		}
	}
	return normal, dark, nil
}
