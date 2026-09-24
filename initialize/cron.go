package initialize

import (
	"github.com/robfig/cron/v3"
	"iptv-spider/global"
	"sync"
	"time"
)

var onceCron sync.Once

func InitCron() {
	if global.CRON == nil {
		onceCron.Do(func() {
			nyc, _ := time.LoadLocation("Asia/Shanghai")
			global.CRON = cron.New(cron.WithSeconds(), cron.WithLocation(nyc))
			global.CRON.Start()
		})
	}
}
