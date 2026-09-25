package panel

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	cctvPattern = regexp.MustCompile(`(?i)CCTV[-_\s]*([48]K|\d+\+?)`)
	cetvPattern = regexp.MustCompile(`(?i)(?:CETV|中国教育)[-_\s]*(?:电视台)?[-_\s]*([1-4])`)
)

// ExtractCCTVID extracts the CCTV channel identifier ("1", "2", "5+", "16", "4K", "8K").
// Returns empty string if not a CCTV channel pattern.
func ExtractCCTVID(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "plus", "+")
	s = strings.ReplaceAll(s, "PLUS", "+")
	m := cctvPattern.FindStringSubmatch(s)
	if len(m) > 1 {
		return strings.ToUpper(m[1])
	}
	return ""
}

// ExtractCETVID extracts CETV channel number ("1" to "4").
func ExtractCETVID(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	m := cetvPattern.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// SimplifyForMatch strips punctuation, symbols, whitespace, and trailing resolution/quality tags,
// converting to lower case for fuzzy matching.
func SimplifyForMatch(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	res := sb.String()
	tags := []string{
		"4k超高清", "超高清", "4k", "8k", "uhd", "fhd", "hd", "sd", "50p", "高清", "标清",
	}
	for _, tag := range tags {
		if strings.HasSuffix(res, tag) {
			res = strings.TrimSuffix(res, tag)
			break
		}
	}
	return res
}

func isPrefixBoundaryMatch(prefix, full string) bool {
	if !strings.HasPrefix(full, prefix) {
		return false
	}
	rem := full[len(prefix):]
	if len(rem) > 0 {
		first := rem[0]
		if first >= '0' && first <= '9' {
			return false
		}
	}
	return true
}

func isHDor4K(name string) bool {
	upper := strings.ToUpper(name)
	return strings.Contains(upper, "HD") || strings.Contains(upper, "4K") || strings.Contains(upper, "8K") ||
		strings.Contains(upper, "UHD") || strings.Contains(upper, "FHD") ||
		strings.Contains(name, "高清") || strings.Contains(name, "超高清")
}

func nameSimilarity(s1, s2 string) float64 {
	r1 := []rune(s1)
	r2 := []rune(s2)
	if len(r1) == 0 && len(r2) == 0 {
		return 1.0
	}
	if len(r1) == 0 || len(r2) == 0 {
		return 0.0
	}
	d := levenshteinDistance(r1, r2)
	maxLen := math.Max(float64(len(r1)), float64(len(r2)))
	return 1.0 - float64(d)/maxLen
}

func levenshteinDistance(s1, s2 []rune) int {
	len1 := len(s1)
	len2 := len(s2)
	d := make([][]int, len1+1)
	for i := range d {
		d[i] = make([]int, len2+1)
		d[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		d[0][j] = j
	}
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[len1][len2]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

type ChannelMatchCandidate struct {
	ID         string
	Name       string
	OperatorID string
	Group      string
	Score      int
	IsSaved    bool
	Source     string
}

func stripProvincePrefix(s string) string {
	for _, p := range provincePrefixes {
		if strings.HasPrefix(s, p) {
			rem := s[len(p):]
			if len([]rune(rem)) >= 2 {
				return rem
			}
		}
	}
	return s
}

// MatchChannelCandidates returns a ranked list of fuzzy-matched channel candidates.
func MatchChannelCandidates(query string, candidates []Channel, maxResults int) []ChannelMatchCandidate {
	query = strings.TrimSpace(query)
	if query == "" || len(candidates) == 0 {
		return nil
	}
	if maxResults <= 0 {
		maxResults = 10
	}

	canQ := CanonicalName(query)
	simQ := SimplifyForMatch(query)
	cctvQ := ExtractCCTVID(query)
	cetvQ := ExtractCETVID(query)
	provQ := stripProvincePrefix(simQ)

	type scoredCandidate struct {
		cand  ChannelMatchCandidate
		score int
	}

	var matches []scoredCandidate
	for _, c := range candidates {
		if !IsValidChannelID(c.ID) {
			continue
		}
		targetName := strings.TrimSpace(c.Name)
		if targetName == "" {
			continue
		}

		score := 0
		canT := CanonicalName(targetName)
		simT := SimplifyForMatch(targetName)
		cctvT := ExtractCCTVID(targetName)
		cetvT := ExtractCETVID(targetName)
		provT := stripProvincePrefix(simT)

		if cctvQ != "" || cctvT != "" {
			if cctvQ != "" && cctvT != "" && cctvQ == cctvT {
				score = 95
				if canQ != "" && canT != "" && canQ == canT {
					score = 100
				} else if simQ != "" && simT != "" && simQ == simT {
					score = 100
				}
			} else {
				continue
			}
		} else if cetvQ != "" || cetvT != "" {
			if cetvQ != "" && cetvT != "" && cetvQ == cetvT {
				score = 95
				if canQ != "" && canT != "" && canQ == canT {
					score = 100
				}
			} else {
				continue
			}
		} else {
			if canQ != "" && canT != "" && canQ == canT {
				score = 100
			} else if simQ != "" && simT != "" && simQ == simT {
				score = 100
			} else if provQ != "" && provT != "" && provQ == provT {
				score = 95
			} else if simQ != "" && simT != "" && (isPrefixBoundaryMatch(simQ, simT) || isPrefixBoundaryMatch(simT, simQ)) {
				score = 85
			} else if len([]rune(simQ)) >= 2 && len([]rune(simT)) >= 2 && (strings.Contains(simT, simQ) || strings.Contains(simQ, simT)) {
				score = 75
			} else if len([]rune(provQ)) >= 2 && len([]rune(provT)) >= 2 && (strings.Contains(provT, provQ) || strings.Contains(provQ, provT)) {
				score = 70
			} else if simQ != "" && simT != "" {
				sim := nameSimilarity(simQ, simT)
				if sim >= 0.60 {
					score = 50 + int(sim*35)
				}
			}
		}

		if score >= 60 {
			matches = append(matches, scoredCandidate{
				cand: ChannelMatchCandidate{
					ID:         c.ID,
					Name:       c.Name,
					OperatorID: c.OperatorID,
					Group:      c.Group,
					Score:      score,
					Source:     c.Source,
				},
				score: score,
			})
		}
	}

	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if (a.cand.Source == "iptv") != (b.cand.Source == "iptv") {
			return a.cand.Source == "iptv"
		}
		aHD := isHDor4K(a.cand.Name)
		bHD := isHDor4K(b.cand.Name)
		if aHD != bHD {
			return aHD
		}
		return len(a.cand.Name) < len(b.cand.Name)
	})

	seen := map[string]bool{}
	var out []ChannelMatchCandidate
	for _, m := range matches {
		sig := m.cand.ID + "|" + m.cand.Name + "|" + m.cand.OperatorID
		if seen[sig] {
			continue
		}
		seen[sig] = true
		out = append(out, m.cand)
		if len(out) >= maxResults {
			break
		}
	}
	return out
}

// MatchChannelName searches candidates for the best fuzzy match for the query channel name,
// prioritizing channels that have a valid numeric channel ID.
func MatchChannelName(query string, candidates []Channel) *Channel {
	cands := MatchChannelCandidates(query, candidates, 1)
	if len(cands) == 0 {
		return nil
	}
	top := cands[0]
	return &Channel{
		ID:         top.ID,
		Name:       top.Name,
		OperatorID: top.OperatorID,
		Group:      top.Group,
		Source:     top.Source,
	}
}
