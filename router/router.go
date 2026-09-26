package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12"
	"iptv-spider/modules/panel"
	"iptv-spider/router/api"
)

func InitRouters(app *iris.Application) {
	registerMacros(app)
	// 中间件注册
	//app.UseRouter(middleware.Cors())
	// 各个路由分组
	apiRouterGroup := app.Party("/api")
	{
		api.InitApiRouters(apiRouterGroup)
	}
	if panel.Current != nil {
		token := os.Getenv("IPTV_PANEL_TOKEN")
		if token == "" {
			dataDir := os.Getenv("IPTV_DATA_DIR")
			if dataDir == "" {
				dataDir = "data"
			}
			if b, err := os.ReadFile(filepath.Join(dataDir, "admin-token")); err == nil {
				token = strings.TrimSpace(string(b))
			}
		}
		handler := iris.FromStd(panel.Current.Handler(token))
		app.Get("/", handler)
		app.Get("/app.js", handler)
		app.Get("/style.css", handler)
		app.Get("/player", handler)
		app.Get("/player.html", handler)
		app.Get("/assets/{path:path}", handler)
		app.Get("/status.html", handler)
		app.Get("/playlist.m3u", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		app.Head("/playlist.m3u", iris.FromStd(http.HandlerFunc(panel.Current.ServePlaylist)))
		app.Get("/epg.xml", iris.FromStd(http.HandlerFunc(panel.Current.ServeEPG)))
		app.Head("/epg.xml", iris.FromStd(http.HandlerFunc(panel.Current.ServeEPG)))
		app.Get("/epg.xml.gz", iris.FromStd(http.HandlerFunc(panel.Current.ServeEPG)))
		app.Head("/epg.xml.gz", iris.FromStd(http.HandlerFunc(panel.Current.ServeEPG)))
		app.Get("/logos/{path:path}", iris.FromStd(http.HandlerFunc(panel.Current.ServeLocalLogo)))
		app.Any("/api/panel/{path:path}", handler)
	}

}
