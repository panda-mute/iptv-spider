package model

import "gorm.io/gorm"

type AuthInfo struct {
	gorm.Model
	UID          string `gorm:"uniqueIndex;comment:用户ID"`
	UserToken    string `gorm:"comment:用户Token"`
	JSESSIONID   string `gorm:"comment:BIM EPG 会话ID"`
	BimAuthInfo  string `gorm:"type:longtext;comment:BIM 认证信息"`
	EPGHostUrl   string `gorm:"comment:EPG服务器地址"`
	EPGLoginHost string `gorm:"comment:EPG认证服务器主机"`
	R4kLoginHost string `gorm:"comment:R4K服务器主机"`
}
