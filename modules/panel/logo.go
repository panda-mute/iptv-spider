package panel

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

const LogoSource = "https://iptv.yang-1989.xyz/logo"

// Snapshot of the user-selected source; keeps matching available offline.
//
//go:embed logos.json
var bundledLogos []byte

func logoKey(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	name = strings.NewReplacer(" ", "", "-", "", "_", "", "（", "(", "）", ")").Replace(name)
	for _, suffix := range []string{"(HD)", "(高清)", "高清", "HD"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

func parseLogoCatalog(data []byte) map[string]string {
	var body struct {
		Logos []struct {
			Name string `json:"name"`
			File string `json:"file"`
		} `json:"logos"`
	}
	if json.Unmarshal(data, &body) != nil {
		return nil
	}
	sort.Slice(body.Logos, func(i, j int) bool { return body.Logos[i].File < body.Logos[j].File })
	names := map[string]string{}
	for _, logo := range body.Logos {
		key := logoKey(logo.Name)
		if key == "" || logo.File == "" || strings.ContainsAny(logo.File, "/\\\r\n") || logo.File == "." || logo.File == ".." {
			continue
		}
		if _, exists := names[key]; !exists {
			names[key] = LogoSource + "/" + url.PathEscape(logo.File)
		}
	}
	return names
}

func channelLogo(name string) string {
	return configuredLogo(defaultLogoSources(), name, "")
}
