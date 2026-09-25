package panel

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	cctvNumericRegex   = regexp.MustCompile(`(?i)CCTV[-_\s]*([48]K|\d+)(?:\+?|\s*plus)?`)
	cctvDedicatedRegex = regexp.MustCompile(`(?i)CCTV[-_\s]*([48]K)`)
	cctv5PlusRegex     = regexp.MustCompile(`(?i)CCTV[-_\s]*5\s*(?:\+|plus)`)
	centralNumRegex    = regexp.MustCompile(`中央(?:电视台)?[-_\s]*(\d+)`)
	centralZhRegex     = regexp.MustCompile(`中央(?:电视台)?[-_\s]*([一二三四五六七八九十]+)`)
)

var cctvAliasRanks = map[string]float64{
	"综合":   1.0,
	"财经":   2.0,
	"综艺":   3.0,
	"中文国际": 4.0,
	"体育":   5.0,
	"体育赛事": 5.5,
	"电影":   6.0,
	"国防军事": 7.0,
	"电视剧":  8.0,
	"纪录":   9.0,
	"科教":   10.0,
	"戏曲":   11.0,
	"社会与法": 12.0,
	"新闻":   13.0,
	"少儿":   14.0,
	"音乐":   15.0,
	"奥林匹克": 16.0,
	"农业农村": 17.0,
}

var chineseNumMap = map[string]int{
	"一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6, "七": 7, "八": 8, "九": 9,
	"十": 10, "十一": 11, "十二": 12, "十三": 13, "十四": 14, "十五": 15, "十六": 16, "十七": 17,
}

// CompareNatural compares two strings naturally, sorting digit segments as integer numbers.
func CompareNatural(a, b string) int {
	la, lb := len(a), len(b)
	ia, ib := 0, 0
	for ia < la && ib < lb {
		ra, sizeA := utf8.DecodeRuneInString(a[ia:])
		rb, sizeB := utf8.DecodeRuneInString(b[ib:])

		isDigitA := unicode.IsDigit(ra)
		isDigitB := unicode.IsDigit(rb)

		if isDigitA && isDigitB {
			startA := ia
			for ia < la {
				r, s := utf8.DecodeRuneInString(a[ia:])
				if !unicode.IsDigit(r) {
					break
				}
				ia += s
			}
			startB := ib
			for ib < lb {
				r, s := utf8.DecodeRuneInString(b[ib:])
				if !unicode.IsDigit(r) {
					break
				}
				ib += s
			}
			numStrA := strings.TrimLeft(a[startA:ia], "0")
			numStrB := strings.TrimLeft(b[startB:ib], "0")
			if len(numStrA) != len(numStrB) {
				if len(numStrA) < len(numStrB) {
					return -1
				}
				return 1
			}
			if numStrA != numStrB {
				if numStrA < numStrB {
					return -1
				}
				return 1
			}
			lenA := ia - startA
			lenB := ib - startB
			if lenA != lenB {
				if lenA < lenB {
					return -1
				}
				return 1
			}
			continue
		}

		if isDigitA != isDigitB {
			if isDigitA {
				return -1
			}
			return 1
		}

		lowerA := unicode.ToLower(ra)
		lowerB := unicode.ToLower(rb)
		if lowerA != lowerB {
			if lowerA < lowerB {
				return -1
			}
			return 1
		}
		ia += sizeA
		ib += sizeB
	}
	if ia < la {
		return 1
	}
	if ib < lb {
		return -1
	}
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// CCTVNumericRank returns the numeric sequence position (e.g. 1.0, 2.0, 5.5, 16.0, 18.0)
// and whether the channel is identified as CCTV.
func CCTVNumericRank(name string) (float64, bool) {
	s := strings.TrimSpace(name)
	if s == "" {
		return 0, false
	}
	upper := strings.ToUpper(s)

	// CCTV 4K / 8K
	if strings.Contains(upper, "4K") && (strings.Contains(upper, "CCTV") || strings.Contains(s, "中央") || strings.Contains(s, "央视")) {
		return 18.0, true
	}
	if strings.Contains(upper, "8K") && (strings.Contains(upper, "CCTV") || strings.Contains(s, "中央") || strings.Contains(s, "央视")) {
		return 19.0, true
	}

	// CCTV 5+
	if cctv5PlusRegex.MatchString(s) || strings.Contains(s, "中央五+") || strings.Contains(s, "中央5+") {
		return 5.5, true
	}

	// CCTV-1 to CCTV-17 numeric
	if m := cctvNumericRegex.FindStringSubmatch(s); len(m) > 1 {
		val, err := strconv.Atoi(m[1])
		if err == nil && val >= 1 && val <= 17 {
			if strings.Contains(upper, "+") || strings.Contains(upper, "PLUS") {
				return float64(val) + 0.5, true
			}
			return float64(val), true
		}
	}

	// Central 1..17
	if m := centralNumRegex.FindStringSubmatch(s); len(m) > 1 {
		val, err := strconv.Atoi(m[1])
		if err == nil && val >= 1 && val <= 17 {
			if strings.Contains(s, "+") {
				return float64(val) + 0.5, true
			}
			return float64(val), true
		}
	}

	// Central 一..十七
	if m := centralZhRegex.FindStringSubmatch(s); len(m) > 1 {
		if val, ok := chineseNumMap[m[1]]; ok {
			if strings.Contains(s, "+") {
				return float64(val) + 0.5, true
			}
			return float64(val), true
		}
	}

	// Channel aliases like CCTV新闻 -> 13
	isCCTVScope := strings.Contains(upper, "CCTV") || strings.Contains(s, "中央") || strings.Contains(s, "央视")
	if isCCTVScope {
		for alias, rank := range cctvAliasRanks {
			if strings.Contains(s, alias) {
				return rank, true
			}
		}
		// Non-numbered CCTV channel (e.g. CCTV风云足球, CCTV兵器科技)
		return 100.0, true
	}

	return 0, false
}

// GroupRank determines priority order of channel groups.
func GroupRank(group string, customOrder []string) int {
	norm := strings.TrimSpace(group)
	if norm == "" || norm == "待识别" || norm == "未识别" || norm == "未分组" {
		for i, co := range customOrder {
			if strings.EqualFold(strings.TrimSpace(co), norm) {
				return i * 10
			}
		}
		return 9999
	}

	for i, co := range customOrder {
		cTrim := strings.TrimSpace(co)
		if strings.EqualFold(cTrim, norm) {
			return i * 10
		}
		// Fuzzy match custom group names (e.g. 央视 vs 央视频道)
		if (strings.EqualFold(cTrim, "4k") && strings.Contains(strings.ToLower(norm), "4k")) ||
			(strings.Contains(cTrim, "央视") && strings.Contains(norm, "央视")) ||
			(strings.Contains(cTrim, "卫视") && strings.Contains(norm, "卫视")) ||
			(strings.Contains(cTrim, "高清") && (strings.Contains(norm, "高清") || strings.Contains(norm, "数字"))) ||
			((strings.Contains(cTrim, "上海") || strings.Contains(cTrim, "本地")) && (strings.Contains(norm, "上海") || strings.Contains(norm, "本地"))) ||
			(strings.Contains(cTrim, "少儿") && strings.Contains(norm, "少儿")) ||
			(strings.Contains(cTrim, "标清") && strings.Contains(norm, "标清")) ||
			(strings.Contains(cTrim, "数字") && strings.Contains(norm, "数字")) ||
			((cTrim == "其它" || cTrim == "其他") && (norm == "其它" || norm == "其他")) {
			return i * 10
		}
	}

	lower := strings.ToLower(norm)
	switch {
	case strings.Contains(lower, "4k") || strings.Contains(lower, "8k") || strings.Contains(lower, "超高清"):
		return 50
	case strings.Contains(lower, "央视") || strings.Contains(lower, "cctv") || strings.Contains(lower, "中央"):
		return 100
	case strings.Contains(lower, "卫视"):
		return 200
	case strings.Contains(lower, "高清"):
		return 250
	case strings.Contains(lower, "上海") || strings.Contains(lower, "本地") || strings.Contains(lower, "地方"):
		return 300
	case strings.Contains(lower, "影视") || strings.Contains(lower, "电影") || strings.Contains(lower, "电视剧") || strings.Contains(lower, "剧场"):
		return 400
	case strings.Contains(lower, "体育") || strings.Contains(lower, "赛事") || strings.Contains(lower, "竞技"):
		return 500
	case strings.Contains(lower, "少儿") || strings.Contains(lower, "动画") || strings.Contains(lower, "动漫") || strings.Contains(lower, "儿童"):
		return 600
	case strings.Contains(lower, "纪实") || strings.Contains(lower, "纪录") || strings.Contains(lower, "科教"):
		return 700
	case strings.Contains(lower, "新闻") || strings.Contains(lower, "资讯"):
		return 800
	case strings.Contains(lower, "音乐") || strings.Contains(lower, "综艺") || strings.Contains(lower, "娱乐") || strings.Contains(lower, "戏曲"):
		return 900
	case strings.Contains(lower, "数字") || strings.Contains(lower, "专区") || strings.Contains(lower, "专业") || strings.Contains(lower, "付费") || strings.Contains(lower, "轮播") || strings.Contains(lower, "百视通") || strings.Contains(lower, "bestv"):
		return 1000
	case strings.Contains(lower, "国际") || strings.Contains(lower, "港澳台"):
		return 1100
	case strings.Contains(lower, "标清"):
		return 7000
	case strings.Contains(lower, "其它") || strings.Contains(lower, "其他") || strings.Contains(lower, "购物") || strings.Contains(lower, "测试"):
		return 8000
	default:
		return 5000
	}
}

// SortChannels sorts a slice of channels according to group order, custom in-group order,
// CCTV numeric sequence, digital channel ID, resolution, and natural name order.
func SortChannels(channels []Channel, customGroupOrder []string, groupChannelOrder ...map[string][]string) {
	var inGroupOrders map[string][]string
	if len(groupChannelOrder) > 0 && groupChannelOrder[0] != nil {
		inGroupOrders = groupChannelOrder[0]
	}

	sort.SliceStable(channels, func(i, j int) bool {
		cI, cJ := channels[i], channels[j]

		// 1. Group comparison
		if cI.Group != cJ.Group {
			rI := GroupRank(cI.Group, customGroupOrder)
			rJ := GroupRank(cJ.Group, customGroupOrder)
			if rI != rJ {
				return rI < rJ
			}
			return CompareNatural(cI.Group, cJ.Group) < 0
		}

		// 2. Custom in-group channel order (if configured for this group)
		if inGroupOrders != nil {
			if customList, ok := inGroupOrders[cI.Group]; ok && len(customList) > 0 {
				getRank := func(c Channel) int {
					key := c.EnsureKey()
					for idx, target := range customList {
						target = strings.TrimSpace(target)
						if target != "" && (target == key || target == c.ID || (c.OriginalID != "" && target == c.OriginalID) || target == c.Name) {
							return idx
						}
					}
					return 999999
				}
				idxI := getRank(cI)
				idxJ := getRank(cJ)
				if idxI != idxJ {
					return idxI < idxJ
				}
			}
		}

		// 3. CCTV numeric sorting inside the group
		rCCTVI, isCCTVI := CCTVNumericRank(cI.Name)
		rCCTVJ, isCCTVJ := CCTVNumericRank(cJ.Name)
		if isCCTVI && isCCTVJ {
			if rCCTVI != rCCTVJ {
				return rCCTVI < rCCTVJ
			}
			// Same CCTV rank (e.g. CCTV-1 SD vs CCTV-1 HD):
			// Check digital ID if positive and different
			numI, errI := strconv.Atoi(cI.ID)
			numJ, errJ := strconv.Atoi(cJ.ID)
			if errI == nil && errJ == nil && numI > 0 && numJ > 0 && numI != numJ {
				return numI < numJ
			}
			// Compare resolution: higher resolution first
			cmpRes := CompareResolution(cI.Resolution, cJ.Resolution)
			if cmpRes != 0 {
				return cmpRes < 0
			}
			return CompareNatural(cI.Name, cJ.Name) < 0
		}
		if isCCTVI != isCCTVJ {
			// CCTV channels always precede non-CCTV channels in the same group
			return isCCTVI
		}

		// 4. Non-CCTV channels: compare digital channel ID
		validI := IsValidChannelID(cI.ID)
		validJ := IsValidChannelID(cJ.ID)
		if validI && validJ {
			numI, errI := strconv.Atoi(cI.ID)
			numJ, errJ := strconv.Atoi(cJ.ID)
			if errI == nil && errJ == nil && numI != numJ {
				return numI < numJ
			}
		} else if validI != validJ {
			return validI
		}

		// 5. Compare resolution
		cmpRes := CompareResolution(cI.Resolution, cJ.Resolution)
		if cmpRes != 0 {
			return cmpRes < 0
		}

		// 6. Compare name naturally
		cmpName := CompareNatural(cI.Name, cJ.Name)
		if cmpName != 0 {
			return cmpName < 0
		}

		return cI.URL < cJ.URL
	})
}
