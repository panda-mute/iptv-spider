package initialize

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"iptv-spider/global"
	"iptv-spider/model"
	"os"
	"time"

	"iptv-spider/modules/panel"
	"iptv-spider/modules/spider"
)

func setupPanelEPG(s *panel.Service) error {
	s.EPGBackend = "file"
	if global.DB != nil {
		db := global.DB
		if err := db.AutoMigrate(&model.PanelEPGSnapshot{}); err != nil {
			return err
		}
		s.EPGBackend = "mysql"
		s.LoadEPGData = func() ([]byte, error) {
			var row model.PanelEPGSnapshot
			err := db.First(&row, 1).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, os.ErrNotExist
			}
			return row.Payload, err
		}
		s.SaveEPGData = func(b []byte) error {
			return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.PanelEPGSnapshot{ID: 1, Payload: b}).Error
		}
	}

	if err := s.OpenEPG(); err != nil {
		return fmt.Errorf("读取本地 EPG: %w", err)
	}
	s.FetchEPG = func(ctx context.Context, cfg panel.Settings, progress func(int, int, int)) (map[string][]panel.Programme, error) {
		client, err := spider.New(cfg)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		if err = client.Authenticate(ctx); err != nil {
			return nil, err
		}
		catalog, err := client.Channels(ctx)
		if err != nil {
			return nil, err
		}
		rawMap := map[string]string{}
		for _, r := range catalog.Raw {
			if r.UserChannelID != "" && r.ChannelID != "" {
				rawMap[r.UserChannelID] = r.ChannelID
			}
		}
		result := map[string][]panel.Programme{}
		count, failed := 0, 0
		// Group SD/HD/4K variants sharing the same canonical programme identity.
		seenCanonical := map[string][]panel.Programme{}
		seenChID := map[string]bool{}
		now := time.Now().In(time.FixedZone("CST", 8*3600))
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		start, end := midnight.AddDate(0, 0, -cfg.EPG.PastDays), midnight.AddDate(0, 0, cfg.EPG.FutureDays+1)
		for i, info := range catalog.Info {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			canName := panel.CanonicalName(info.Name)
			var p []panel.Programme
			if cached, ok := seenCanonical[canName]; ok && len(cached) > 0 {
				p = cached
			} else if !seenChID[info.ChID] {
				seenChID[info.ChID] = true
				programs, e := client.ProgramsRange(ctx, info, start, end)
				if e != nil {
					failed++
				} else if len(programs) > 0 {
					p = make([]panel.Programme, 0, len(programs))
					for _, v := range programs {
						p = append(p, panel.Programme{Title: v.Name, Start: v.StartTime / 1000, End: v.EndTime / 1000, Desc: v.Detail()})
					}
					if canName != "" {
						seenCanonical[canName] = p
					}
					count += len(p)
				}
				// Bound operator request rate; cancellation interrupts the wait.
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case <-time.After(100 * time.Millisecond):
				}
			}
			if len(p) > 0 {
				if info.ChID != "" {
					result[info.ChID] = p
				}
				if info.MixNo != "" {
					result[info.MixNo] = p
				}
				if rawChID := rawMap[info.MixNo]; rawChID != "" {
					result[rawChID] = p
				}
				if info.Code != "" {
					result[info.Code] = p
				}
				if canName != "" {
					result[canName] = p
				}
			}
			progress(i+1, len(catalog.Info), count)
		}
		if failed > 0 {
			return result, fmt.Errorf("%d 个频道同步失败；已保存成功频道，保留失败频道旧数据", failed)
		}
		if len(result) == 0 {
			return nil, errors.New("运营商没有返回可用节目单")
		}
		return result, nil
	}
	return nil
}
