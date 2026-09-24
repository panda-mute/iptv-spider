package model

import "gorm.io/gorm"

// M3u8Mapping m3u 部分字段映射
type M3u8Mapping struct {
	gorm.Model
	CommName     string `gorm:"uniqueIndex;comment:节目通用名称" json:"-"`
	Logo         string `gorm:"comment:Logo 地址"`
	AutoGroups   string `gorm:"comment:自动分组"`
	CustomGroups string `gorm:"comment:自定义组分类"`
}
