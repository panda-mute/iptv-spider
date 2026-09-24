package auth

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/golang-module/carbon"
	"iptv-spider/global"
	"iptv-spider/model"
	"iptv-spider/modules/m3u"
	"iptv-spider/modules/panel"
	"iptv-spider/utils"
	"net/url"
)

const timeFormat = carbon.ShortDateTimeLayout + " -0700"

func GenerateM3u8(udpxy, scheme, xteve, all string) []byte {
	if panel.Current != nil {
		channels, err := panel.Current.Export(udpxy, scheme, xteve, all)
		if err != nil {
			global.LOG.Error(err.Error())
			return nil
		}
		return []byte(panel.Current.M3U(channels))
	}

	m3uWriter := m3u.NewWriter()
	m3uWriter.WriteHeaderWithInfo(global.CONFIG.Epg.XmlUrl)

	// 查询数据库
	var channelInfoList []model.ChannelInfo
	global.DB.Order("mix_no asc").
		Find(&channelInfoList)
	// 去重
	newChanInfo := model.RemoveDuplicateChannelInfo(channelInfoList)
	for _, info := range newChanInfo {
		// 不展示
		if !info.IsShow {
			continue
		}
		channel := model.Channel{}
		global.DB.Where("user_channel_id = ?", info.MixNo).
			Find(&channel)

		m3u8Mapping := model.M3u8Mapping{}
		global.DB.Where("comm_name = ?", info.CommName).
			Find(&m3u8Mapping)

		if all != "true" && (m3u8Mapping.AutoGroups == "购物" ||
			m3u8Mapping.CustomGroups == "购物") {
			continue
		}
		uri := assemblyUrl(udpxy, scheme, xteve, channel.ChannelURL)
		m3uWriter.Write(uri, info, m3u8Mapping)
	}
	return m3uWriter.Bytes()
}

func GenerateTimeShiftM3u8() []byte {
	m3uWriter := m3u.NewWriter()
	m3uWriter.WriteHeaderWithInfo(global.CONFIG.Epg.XmlUrl)
	// 查询数据库
	var channelInfoList []model.ChannelInfo
	global.DB.Find(&channelInfoList)
	// 去重
	newChanInfo := model.RemoveDuplicateChannelInfo(channelInfoList)
	for _, info := range newChanInfo {
		// 不展示
		if !info.IsShow {
			continue
		}
		channel := model.Channel{}
		global.DB.Where("user_channel_id = ?", info.MixNo).
			Find(&channel)

		m3u8Mapping := model.M3u8Mapping{}
		global.DB.Where("comm_name = ?", info.CommName).
			Find(&m3u8Mapping)

		if m3u8Mapping.AutoGroups == "购物" || m3u8Mapping.CustomGroups == "购物" {
			continue
		}
		uri := assemblyUrl("", "", "", channel.TimeShiftURL)
		m3uWriter.Write(uri, info, m3u8Mapping)
	}
	return m3uWriter.Bytes()
}

func assemblyUrl(udpxy, scheme, xteve, uri string) string {
	u, _ := url.Parse(uri)
	if xteve == "true" {
		return fmt.Sprintf("udp://@%s", u.Host)
	} else if udpxy != "" {
		return fmt.Sprintf("http://%s/udp/%s", udpxy, u.Host)
	} else if scheme != "" {
		return fmt.Sprintf("%s://%s", scheme, u.Host)
	}
	return uri
}

func GenerateXmlTv(daysAgo int) ([]byte, error) {
	if daysAgo < 1 {
		daysAgo = 1
	} else if daysAgo > 7 {
		daysAgo = 7
	}
	var now = carbon.Now()
	var xmlTv = model.XmlTV{
		Generator: fmt.Sprintf("%s %s", global.CONFIG.Epg.Generator, now.ToDateTimeString()),
		Source:    global.CONFIG.Epg.Source,
	}
	// 取数据
	var channelInfoList []model.ChannelInfo
	global.DB.Find(&channelInfoList)
	// 去重
	newChanInfo := model.RemoveDuplicateChannelInfo(channelInfoList)
	for _, info := range newChanInfo {
		// 不展示
		if !info.IsShow {
			continue
		}
		chId := info.MixNo
		xmlTv.Channel = append(xmlTv.Channel, &model.XmlTvChannel{
			ID:          chId,
			DisplayName: []model.DisplayName{{Lang: "zh", Value: info.CommName}},
		})
		if !info.IsPullEPG {
			xmlTv.Program = append(xmlTv.Program, &model.Program{
				Channel: chId,
				Title:   []*model.Title{{Lang: "zh"}},
				Desc:    []*model.Desc{{Lang: "zh"}},
			})
			continue
		}

		var epgData []model.EPGDetails
		global.DB.Where("comm_name = ?", info.CommName).
			Where("end_time > ?", now.SubDays(daysAgo).TimestampMilli()).
			Order("start_time asc").
			Find(&epgData)

		for _, epg := range epgData {
			startTime := carbon.CreateFromTimestampMilli(epg.StartTime).Layout(timeFormat)
			endTime := carbon.CreateFromTimestampMilli(epg.EndTime).Layout(timeFormat)
			var desc []*model.Desc
			if d := epg.Detail(); d != "" {
				desc = []*model.Desc{{Lang: "zh", Value: d}}
			} else {
				desc = []*model.Desc{{Lang: "zh"}}
			}
			xmlTv.Program = append(xmlTv.Program, &model.Program{
				Channel: chId,
				Start:   startTime,
				Stop:    endTime,
				Title:   []*model.Title{{Lang: "zh", Value: epg.Name}},
				Desc:    desc,
			})
		}
	}
	// 序列化
	epgBytes, err := xml.MarshalIndent(&xmlTv, "", "  ")
	if err != nil {
		global.LOG.Error("节目表单生成出错: " + err.Error())
		return nil, errors.New("节目表单生成出错")
	}
	epgBytes = append([]byte(model.PrefixHeader+"\n"), epgBytes...)
	return epgBytes, nil
}

func GenerateAndUploadM3u() {
	m3uBytes := GenerateM3u8("", "", "true", "")
	utils.UploadToOSS("/sh/tel-xteve.m3u", m3uBytes)
}

func GenerateAndUploadXmlTv() {
	xmlTvBytes, _ := GenerateXmlTv(1)
	utils.UploadToOSS("/sh/tel-epg.xml", xmlTvBytes)
}

func GenerateAndUploadXmlTvDays7() {
	xmlTvBytes, _ := GenerateXmlTv(7)
	utils.UploadToOSS("/sh/tel-epg-7.xml", xmlTvBytes)
}
