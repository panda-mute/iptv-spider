package panel

import (
	"regexp"
	"strings"
)

var (
	bracketQualityPattern = regexp.MustCompile(`(?i)[(\[（](?:高清|标清|4K|8K|HD|FHD|UHD|50P)[)\]）]`)
	bracketAnyPattern     = regexp.MustCompile(`(?i)[(\[（][^)\]）]*[)\]）]`)
	cctvDedicated4k8k     = regexp.MustCompile(`(?i)^CCTV[-_\s]*([48]K)$`)
	trailingResolution    = regexp.MustCompile(`(?i)[-_\s]*(?:4K超高清|超高清|4K|8K|UHD)$`)
	trailingDefinition    = regexp.MustCompile(`(?i)[-_\s]*(?:HD|FHD|SD|高清|标清)$`)
	cctvNumberPattern     = regexp.MustCompile(`(?i)^(CCTV)[-_\s]*(\d+\+?)$`)
)

// CanonicalName normalizes channel names across transmission formats, resolutions, and quality tags.
// For example:
//   "cctv1", "CCTV1", "CCTV-1HD", "CCTV1高清" -> "CCTV-1"
//   "湖南卫视", "湖南卫视HD", "湖南卫视高清", "湖南卫视4K" -> "湖南卫视"
//   "CCTV-4K" -> "CCTV-4K" (dedicated 4K channel preserved)
func CanonicalName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	s = bracketQualityPattern.ReplaceAllString(s, "")
	s = bracketAnyPattern.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if m := cctvDedicated4k8k.FindStringSubmatch(s); len(m) > 1 {
		return "CCTV-" + strings.ToUpper(m[1])
	}
	s = trailingResolution.ReplaceAllString(s, "")
	s = trailingDefinition.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if m := cctvNumberPattern.FindStringSubmatch(s); len(m) > 2 {
		s = "CCTV-" + m[2]
	}
	return strings.ToUpper(strings.TrimSpace(s))
}
