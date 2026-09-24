package model

import (
	"strings"

	"gorm.io/gorm"
)

type EPGDetails struct {
	gorm.Model        `json:"-"`
	CommName          string `gorm:"index;comment:节目通用名称" json:"-"`
	InterRecordStatus string `gorm:"comment:未知" json:"interRecordStatus"`
	Name              string `gorm:"comment:节目名称" json:"name"`
	ChannelName       string `gorm:"-" json:"channelName"`
	StartTime         int64  `gorm:"comment:开始时间" json:"startTime"`
	ID                string `gorm:"comment:ID" json:"ID"`
	EndTime           int64  `gorm:"index;comment:结束时间" json:"endTime"`
	MixNo             string `gorm:"-" json:"mixNo"`
	ChannelID         string `gorm:"-" json:"channelID"`
	Status            string `gorm:"comment:状态" json:"status"`
	Description       string `gorm:"type:text;comment:节目详情/简介" json:"description"`
	Desc              string `gorm:"-" json:"desc"`
	Intro             string `gorm:"-" json:"intro"`
	SubTitle          string `gorm:"-" json:"sub_title"`
	SubTitleCamel     string `gorm:"-" json:"subTitle"`
	ProgramIntro      string `gorm:"-" json:"programIntro"`
	Summary           string `gorm:"-" json:"summary"`
	DetailText        string `gorm:"-" json:"detail"`
}

func (e EPGDetails) Detail() string {
	for _, s := range []string{e.Description, e.Desc, e.Intro, e.ProgramIntro, e.SubTitle, e.SubTitleCamel, e.Summary, e.DetailText} {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}
