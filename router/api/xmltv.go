package api

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"iptv-spider/global"
	"iptv-spider/modules/auth"
	"iptv-spider/modules/panel"
	"iptv-spider/utils"
	"strconv"
	"time"
)

// generateXmlTv 生成xmlTv文件 节目去重
func generateXmlTv(ctx iris.Context) {
	if panel.Current != nil {
		panel.Current.ServeEPG(ctx.ResponseWriter(), ctx.Request())
		return
	}
	if global.DB == nil {
		ctx.StatusCode(503)
		ctx.JSON(iris.Map{"error": "EPG 需要配置 MySQL"})
		return
	}
	ref := ctx.FormValue("ref")
	daysAgo := ctx.FormValueDefault("daysAgo", "1")

	d, err := strconv.Atoi(daysAgo)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}

	// 缓存
	reqMD5Key := utils.CalcMD5KeyForRequest("generateXmlTv", daysAgo)
	if ref != "true" && global.CACHE.IsExist(reqMD5Key) {
		// 存在缓存，直接返回
		ctx.ContentType(context.ContentXMLHeaderValue)
		ctx.Write(global.CACHE.Get(reqMD5Key).([]byte))
		return
	}
	// 并发时合并请求
	resp, err, _ := global.ConcurrencyControl.Do(reqMD5Key, func() (interface{}, error) {
		epgBytes, err := auth.GenerateXmlTv(d)
		timeOut := time.Duration(global.CONFIG.Cache.DefTimeOut)
		global.CACHE.Put(reqMD5Key, epgBytes, time.Minute*timeOut)
		return epgBytes, err
	})
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.ContentType(context.ContentXMLHeaderValue)
	ctx.Write(resp.([]byte))
}
