package initialize

import (
	"fmt"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"
	"iptv-spider/global"
)

func Cache() cache.Cache {
	t := global.CONFIG.Cache.Type
	var c string
	switch t {
	case "redis":
		c = fmt.Sprintf(`{"key":"%s","conn":"%s","dbNum":"%d","password":"%s"}`,
			global.CONFIG.Cache.Prefix,
			global.CONFIG.Redis.Addr,
			global.CONFIG.Redis.DB,
			global.CONFIG.Redis.Password)
	case "memory":
		fallthrough
	default:
		t = "memory"
		memoryInterval := global.CONFIG.Cache.Interval
		if memoryInterval == 0 {
			memoryInterval = 60
		}
		c = fmt.Sprintf(`{"interval": %d}`, memoryInterval)
	}
	bm, err := cache.NewCache(t, c)
	if err != nil {
		panic(err)
	}
	if bm == nil {
		panic("initialize failed, Bm is nil!")
	}
	return bm
}
