package auth

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	"iptv-spider/global"
	"iptv-spider/model"
	"iptv-spider/modules/http_client"
	"iptv-spider/modules/jsvm"
	"iptv-spider/utils"
	"net/http"
	"net/url"
)

var (
	ErrUserID  = errors.New("错误的userID")
	ErrMacAddr = errors.New("错误的Mac地址")
	ErrSNCode  = errors.New("错误的SN码")
	ErrIPAddr  = errors.New("错误的IP地址")
)

var globalClient *Client

type baseConfig struct {
	userID           string
	sn               string
	ip               string
	macAddr          string
	stbType          string
	userAgent        string
	pre4kLogAuthAddr string
}

type Client struct {
	jsVM        *jsvm.VM
	httpClient  *http_client.HttpClient
	htmlDocTemp *goquery.Document
	baseConfig
	model.AuthInfo
}

func (c *Client) authSetupOne() *goquery.Document {
	global.LOG.Info("认证流程一")
	doc := c.pre4kLogAuth()
	doc = c.r4kLogAuth(doc)
	doc = c.ottAuth(doc)
	channels := c.processChannel(doc)
	// 频道列表入库
	global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_channel_id"}},
		UpdateAll: true,
	}).Create(&channels)
	c.htmlDocTemp = doc
	return doc
}

func (c *Client) authSetupTwo() (*goquery.Document, error) {
	global.LOG.Info("认证流程二")
	doc := c.epgIndex(c.htmlDocTemp)
	doc = c.epgLoadBalance(doc)
	doc, err := c.epgPortalAuth(doc)
	if err != nil {
		global.LOG.Info("认证流程结束, 出现错误")
		return nil, err
	}
	c.epgGetPortal()
	// Cookies 入库
	global.LOG.Info("认证流程结束, 认证信息入库")
	global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uid"}},
		UpdateAll: true,
	}).Create(&c.AuthInfo)
	c.updateCookies()
	return doc, nil
}

func (c *Client) StartAuth() error {
	global.LOG.Info("开始认证流程")
	c.authSetupOne()
	_, err := c.authSetupTwo()
	return err
}

func (c *Client) HeartBeat() {
	p := "iptvepg/heartbeat.jsp"
	u, _ := url.Parse(c.EPGHostUrl)
	uri := fmt.Sprintf("http://%s/%s", u.Host, p)
	c.httpClient.Request(uri, "GET", nil)
}

func (c *Client) fetchAuthInfoFormDB() {
	global.LOG.Info("从数据库获取 AuthInfo")
	global.DB.Find(&c.AuthInfo, c.AuthInfo)
	if c.AuthInfo.ID > 0 {
		c.updateCookies()
		err := c.checkSessionState()
		if err != nil {
			global.LOG.Error("从数据库获取 AuthInfo 失败", zap.Any("Msg", err.Error()))
			return
		}
	} else {
		err := c.StartAuth()
		if err != nil {
			return
		}
	}
}

func (c *Client) updateCookies() {
	u, err := url.Parse(c.AuthInfo.EPGHostUrl)
	if err != nil {
		global.LOG.Error("更新认证信息至运行环境失败 ", zap.Any("Msg", err.Error()))
		return
	}
	c.httpClient.SetCookies(&http.Cookie{
		Name:     "JSESSIONID",
		Value:    c.AuthInfo.JSESSIONID,
		Path:     "/",
		Domain:   u.Host,
		Secure:   false,
		HttpOnly: true,
	})
}

func NewClient(uid, sn, mac, ip string, options ...ClientOption) (*Client, error) {
	// 验证参数
	if !utils.CheckUserID(uid) {
		return nil, ErrUserID
	}
	if !utils.CheckSNCode(sn) {
		return nil, ErrSNCode
	}
	if !utils.CheckMacAddressV1(mac) {
		return nil, ErrMacAddr
	}
	if !utils.CheckIPv4Address(ip) {
		return nil, ErrIPAddr
	}

	c := &Client{
		jsVM: jsvm.New(),
		baseConfig: baseConfig{
			userID:           uid,
			sn:               sn,
			macAddr:          mac,
			ip:               ip,
			userAgent:        "webkit;Resolution(PAL,720P,1080P,2106P,4K)",
			pre4kLogAuthAddr: "222.68.208.73:7001",
			stbType:          "B860A",
		},
	}
	for _, opt := range options {
		opt(c)
	}
	c.httpClient = http_client.NewHttpClient(http_client.WithUserAgent(c.userAgent))
	c.UID = uid
	c.fetchAuthInfoFormDB()
	return c, nil
}

func NewGlobalClient(uid, sn, mac, ip string, options ...ClientOption) (client *Client, err error) {
	if globalClient == nil {
		globalClient, err = NewClient(uid, sn, mac, ip, options...)
		if err != nil {
			return nil, err
		}
	}
	return globalClient, nil
}

func GetGlobalClient() *Client {
	return globalClient
}
