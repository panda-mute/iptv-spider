package model

import (
	"gorm.io/gorm"
	"time"
)

type Channel struct {
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	ChannelID      string         `gorm:"comment:频道ID" key:"ChannelID"`
	UserChannelID  string         `gorm:"primarykey;comment:用户频道ID" key:"UserChannelID"`
	ChannelURL     string         `gorm:"comment:频道地址" key:"ChannelURL"`
	TimeShift      string         `gorm:"default:0;comment:回放功能是否开启" key:"TimeShift"`
	ChannelSDP     string         `gorm:"comment:回放服务器" key:"ChannelSDP"`
	TimeShiftURL   string         `gorm:"size:256;comment:回放服务URL地址" key:"TimeShiftURL"`
	ChannelType    string         `gorm:"comment:频道类型" key:"ChannelType"`
	ChannelFCCPort string         `gorm:"comment:FCC端口" key:"ChannelFCCPort"`
	ChannelFCCIP   string         `gorm:"comment:FCC IP" key:"ChannelFCCIP"`
}
