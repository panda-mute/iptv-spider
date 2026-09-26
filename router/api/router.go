package api

import (
	"github.com/kataras/iris/v12"
	"iptv-spider/global"
	"iptv-spider/modules/panel"
	"net/http"
	"time"
)

func InitApiRouters(rg iris.Party) {
	rg.Get("/schedule", schedule)

	rg.Get("/run", func(ctx iris.Context) {
		ctx.StatusCode(410)
		ctx.JSON(iris.Map{"error": "管理操作已迁移到受保护的 POST /api/panel/refresh 接口"})
	})
	if panel.Current != nil {
		rg.Get("/playlist", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		rg.Head("/playlist", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		rg.Get("/playlist.m3u", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		rg.Head("/playlist.m3u", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		rg.Get("/play", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Head("/play", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Get("/play.m3u8", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Head("/play.m3u8", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Get("/play/live.m3u8", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Head("/play/live.m3u8", iris.FromStd(http.HandlerFunc(panel.Current.ServePlay)))
		rg.Get("/epg/programmes", iris.FromStd(http.HandlerFunc(panel.Current.ServeEPGPrograms)))
	}

	rg.Get("/m3u8", generateM3u8)
	rg.Head("/m3u8", generateM3u8)

	rg.Get("/tsM3u8", generateTsM3u8)
	rg.Head("/tsM3u8", generateTsM3u8)

	rg.Get("/epg", generateXmlTv)
	rg.Head("/epg", generateXmlTv)

}

func schedule(ctx iris.Context) {
	type s struct {
		ID       int
		PreTime  time.Time
		NextTime time.Time
	}
	var schedule []s
	for _, entry := range global.CRON.Entries() {
		schedule = append(schedule, s{
			ID:       int(entry.ID),
			PreTime:  entry.Prev,
			NextTime: entry.Next,
		})
	}
	ctx.JSON(schedule)
}

// 生成m3u8文件 节目去重
func generateM3u8(ctx iris.Context) {
	if panel.Current != nil {
		panel.Current.ServePlaylist(ctx.ResponseWriter(), ctx.Request())
		return
	}
	ctx.StatusCode(503)
	ctx.JSON(iris.Map{"error": "面板服务尚未初始化"})
}

func generateTsM3u8(ctx iris.Context) {
	if panel.Current == nil {
		ctx.StatusCode(503)
		return
	}
	r := ctx.Request().Clone(ctx.Request().Context())
	u := *r.URL
	q := u.Query()
	q.Set("mode", "unicast")
	u.RawQuery = q.Encode()
	r.URL = &u
	panel.Current.ServePlaylist(ctx.ResponseWriter(), r)
}
