package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strconv"
	"strings"
	"time"
	"iptv-spider/modules/panel"
)

type ChannelInfo struct {
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	TsTime        int            `gorm:"comment:TimeShiftTime 时移时间" json:"tsTime"`
	Code          string         `gorm:"comment:频道代码" json:"code"`
	AuthCode      string         `gorm:"comment:付费认证代码" json:"authCode"`
	Name          string         `gorm:"comment:频道名称" json:"name"`
	ChID          string         `gorm:"comment:频道ID" json:"ID"`
	MixNo         string         `gorm:"primarykey;comment:用户频道映射" json:"mixNo"`
	MediaID       string         `gorm:"comment:未知" json:"mediaID"`
	IsTs          string         `gorm:"comment:是否支持回放" json:"isTs"`
	IsCharge      string         `gorm:"comment:是否需要付费" json:"isCharge"`
	IsHD          bool           `gorm:"default:false;comment:是否是高清频道" json:"-"`
	Is4K          bool           `gorm:"default:false;comment:是否是4K频道" json:"-"`
	IsPullEPG     bool           `gorm:"default:true;comment:是否拉取节目单" json:"-"`
	IsShow        bool           `gorm:"default:true;comment:是否展示该节目" json:"-"`
	CommName      string         `gorm:"comment:通用标题" json:"-"`
	LastFetchTime time.Time      `gorm:"comment:节目单最后更新时间" json:"-"`
}

func (h *ChannelInfo) processData() {
	name := strings.ToUpper(h.Name)
	h.CommName = panel.CanonicalName(name)
	if h.CommName == "" {
		h.CommName = name
	}
	if strings.Contains(name, "HD") || strings.Contains(name, "高清") {
		h.IsHD = true
	}
	if strings.Contains(name, "4K") || strings.Contains(name, "超高清") {
		h.IsHD = true
		h.Is4K = true
	}
}

func (h *ChannelInfo) updateMapping(tx *gorm.DB) {
	if h.CommName == "" {
		return
	}
	var groups []string
	if strings.Contains(h.CommName, "CCTV") {
		groups = append(groups, "央视")
	} else if strings.Contains(h.Name, "卫视") {
		groups = append(groups, "卫视")
	} else if strings.Contains(h.Name, "购物") {
		groups = append(groups, "购物")
	} else if strings.Contains(h.Name, "年级") {
		groups = append(groups, "空中课堂")
	} else if strings.Contains(h.Name, "百事通") {
		groups = append(groups, "百事通")
	}
	tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "comm_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"auto_groups"}),
	}).Create(&M3u8Mapping{
		CommName:   h.CommName,
		AutoGroups: strings.Join(groups, ","),
	})
}
func (h *ChannelInfo) BeforeCreate(tx *gorm.DB) (err error) {
	h.processData()
	h.updateMapping(tx)
	return
}

func (h *ChannelInfo) BeforeUpdate(tx *gorm.DB) (err error) {
	h.processData()
	h.updateMapping(tx)
	return
}

// RemoveDuplicateChannelInfo  ChannelInfo 数组去重
func RemoveDuplicateChannelInfo(in []ChannelInfo) []ChannelInfo {
	newMap := make(map[string]ChannelInfo, len(in))
	for _, child := range in {
		if ch, ok := newMap[child.CommName]; ok {
			// 判断能否替换
			newMap[child.CommName] = check(ch, child)
			continue
		}
		newMap[child.CommName] = child
	}
	var newArr []ChannelInfo
	for _, v := range newMap {
		newArr = append(newArr, v)
	}
	// sort by mix_no
	sort.Slice(newArr, func(i, j int) bool {
		numi, err := strconv.Atoi(newArr[i].MixNo)
		if err != nil {
			return false
		}
		numj, err := strconv.Atoi(newArr[j].MixNo)
		if err != nil {
			return true
		}
		return numi < numj
	})
	return newArr
}

func check(c1, c2 ChannelInfo) ChannelInfo {
	if c1.Is4K {
		return c1
	} else if c2.Is4K {
		return c2
	} else if c1.IsHD {
		return c1
	}
	return c2
}
