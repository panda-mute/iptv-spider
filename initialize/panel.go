package initialize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"iptv-spider/global"
	"iptv-spider/model"
	"iptv-spider/modules/panel"
	"iptv-spider/modules/spider"
	"iptv-spider/utils"

	"go.uber.org/zap"
)

func Panel() (*panel.Service, error) {
	defaults := panel.DefaultSettings()
	stb := global.CONFIG.Stb
	if utils.CheckUserID(stb.UID) && utils.CheckSNCode(stb.SN) && utils.CheckMacAddressV1(stb.MAC) && utils.CheckIPv4Address(stb.IP) {
		defaults.IPTV = panel.IPTV{Enabled: true, UID: stb.UID, SN: stb.SN, MAC: stb.MAC, IP: stb.IP, Type: stb.Type, AuthHost: stb.AuthHost}
	}
	dataDir := os.Getenv("IPTV_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	store, err := panel.OpenStore(filepath.Join(dataDir, "panel.json"), defaults)
	if err != nil {
		return nil, err
	}
	s := panel.NewService(store)
	s.LogosDir = filepath.Join(dataDir, "logos")
	if err := s.EnsurePresetLogos(); err != nil {
		if global.LOG != nil {
			global.LOG.Warn("初始化预设台标失败", zap.Error(err))
		}
	}
	if err := s.LoadLogoSources(); err != nil {
		return nil, err
	}
	if err := setupPanelEPG(s); err != nil {
		return nil, err
	}
	s.ResolveHTTP = func(ctx context.Context, settings panel.Settings, id string, start, end time.Time) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		client, err := spider.New(settings)
		if err != nil {
			return "", err
		}
		defer client.Close()
		if err = client.Authenticate(ctx); err != nil {
			return "", err
		}
		return client.HTTPPlayback(ctx, id, start, end)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.RefreshLogoSources(ctx)
	}()
	s.EPGURL = global.CONFIG.Epg.XmlUrl
	if global.DB != nil {
		s.Load = legacyChannels
	}
	s.Fetch = func(settings panel.Settings, epg bool) error {
		if epg && global.DB == nil {
			return errors.New("EPG 同步需要在 config.yaml 中配置 MySQL；频道同步不需要数据库")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		client, err := spider.New(settings)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Authenticate(ctx); err != nil {
			return err
		}
		result, err := client.Channels(ctx)
		if err != nil {
			return err
		}
		var discoveredFCCs []string
		for _, r := range result.Raw {
			if r.ChannelFCCIP != "" && r.ChannelFCCPort != "" {
				ep := r.ChannelFCCIP + ":" + r.ChannelFCCPort
				if panel.Endpoint(ep) {
					discoveredFCCs = append(discoveredFCCs, ep)
				}
			}
		}
		if len(discoveredFCCs) > 0 {
			_ = store.AddFCCs(discoveredFCCs...)
		}
		if err = store.Import(result.Channels); err != nil {
			return err
		}
		if global.DB == nil {
			return nil
		}
		err = global.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&result.Raw).Error; err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "mix_no"}}, DoUpdates: clause.AssignmentColumns([]string{"code", "auth_code", "name", "ch_id", "is_charge", "is_hd", "is4_k", "comm_name", "updated_at"})}).Create(&result.Info).Error
		})
		if err != nil {
			return err
		}
		if epg {
			var infos []model.ChannelInfo
			if err = global.DB.Find(&infos).Error; err != nil {
				return err
			}
			for _, info := range model.RemoveDuplicateChannelInfo(infos) {
				if !info.IsShow || !info.IsPullEPG {
					continue
				}
				programs, err := client.Programs(ctx, info)
				if err != nil {
					return err
				}
				if len(programs) == 0 {
					continue
				}
				for i := range programs {
					programs[i].CommName = info.CommName
				}
				err = global.DB.Transaction(func(tx *gorm.DB) error {
					if err := tx.Unscoped().Where("comm_name = ?", info.CommName).Delete(&model.EPGDetails{}).Error; err != nil {
						return err
					}
					return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&programs).Error
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return s, nil
}

func legacyChannels() ([]panel.Channel, error) {
	var raw []model.Channel
	var infos []model.ChannelInfo
	var mappings []model.M3u8Mapping
	if err := global.DB.Find(&raw).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Find(&infos).Error; err != nil {
		return nil, err
	}
	if err := global.DB.Find(&mappings).Error; err != nil {
		return nil, err
	}
	byID := map[string]model.Channel{}
	for _, c := range raw {
		byID[c.UserChannelID] = c
	}
	byName := map[string]model.M3u8Mapping{}
	for _, m := range mappings {
		byName[m.CommName] = m
	}
	channels := []panel.Channel{}
	for _, info := range infos {
		c, ok := byID[info.MixNo]
		if !ok || c.ChannelURL == "" {
			continue
		}
		m := byName[info.CommName]
		group := m.CustomGroups
		if group == "" {
			group = m.AutoGroups
		}
		chID := info.MixNo
		if chID == "" || chID == "0" {
			chID = panel.InvalidChannelID
		}
		channels = append(channels, panel.Channel{ID: chID, Name: info.Name, Group: group, Logo: m.Logo, URL: c.ChannelURL, Enabled: info.IsShow, Source: "iptv"})
	}
	return channels, nil
}
