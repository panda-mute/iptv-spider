package auth

import (
	"fmt"
	"github.com/golang-module/carbon"
	"iptv-spider/global"
	"iptv-spider/model"
)

// CleanEPGDetailsData 节目单数据清理
func CleanEPGDetailsData() {
	// 如果节目单的结束时间大于48小时，则硬删除该数据
	t := carbon.Now().SubDays(8).TimestampMilli()
	r := global.DB.Unscoped().Where("end_time < ?", t).Delete(&model.EPGDetails{})
	log := fmt.Sprintf("清理 EPGDetails;\t条件 end_time < %d", t)
	global.LOG.Info(log)
	if r.RowsAffected > 0 {
		global.LOG.Info(fmt.Sprintf("清理 EPGDetails \t条目: %d", r.RowsAffected))
	}
}

// CleanChannelData 频道列表数据清理
func CleanChannelData() {
	// 如果频道列表的更新时间大于96小时，则删除该数据
	t := carbon.Now().SubHours(96)
	log := fmt.Sprintf("清理 Channel;\t条件 updated_at < %s", t.ToString())
	global.LOG.Info(log)
	r := global.DB.Unscoped().Where("updated_at < ?", t.ToStdTime()).Delete(&model.Channel{})
	if r.RowsAffected > 0 {
		global.LOG.Info(fmt.Sprintf("清理 Channel \t条目: %d", r.RowsAffected))
	}
}

// CleanChannelInfoData 频道信息数据清理
func CleanChannelInfoData() {
	// 如果频道信息的更新时间大于48小时，则软删除该数据
	t := carbon.Now().SubHours(48)
	r := global.DB.
		Where("updated_at < ?", t.ToStdTime()).
		Where("is_show = ?", true).
		Where("is_pull_epg = ?", true).
		Delete(&model.ChannelInfo{})
	log := fmt.Sprintf("清理 ChannelInfo;\t条件 updated_at < %s", t.ToString())
	global.LOG.Info(log)
	if r.RowsAffected > 0 {
		global.LOG.Info(fmt.Sprintf("清理 ChannelInfo 条目: %d", r.RowsAffected))
	}
}
