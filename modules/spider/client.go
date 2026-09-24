// Package spider implements the Shanghai Telecom STB flow without global state
// or a database dependency. It supports the funcportalAuth flow in shdx.php.
package spider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"iptv-spider/model"
	"iptv-spider/modules/panel"
	"iptv-spider/utils"
)

type Client struct {
	http     *http.Client
	stb      panel.IPTV
	portal   string
	channels map[string]model.Channel
}

func New(s panel.Settings) (*Client, error) {
	if !utils.CheckUserID(s.IPTV.UID) || !utils.CheckSNCode(s.IPTV.SN) || !utils.CheckMacAddressV1(s.IPTV.MAC) || !utils.CheckIPv4Address(s.IPTV.IP) {
		return nil, errors.New("IPTV 账号、SN、MAC 或机顶盒 IP 格式不正确")
	}
	// Use the host's default route, without interface binding or HTTP proxies.
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{DialContext: dialer.DialContext, ResponseHeaderTimeout: 15 * time.Second, MaxIdleConns: 10}
	jar, _ := cookiejar.New(nil)
	return &Client{http: &http.Client{Transport: transport, Jar: jar, Timeout: 20 * time.Second}, stb: s.IPTV, channels: map[string]model.Channel{}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) request(ctx context.Context, uri string, form url.Values) ([]byte, *url.URL, error) {
	method := "GET"
	var body io.Reader
	if form != nil {
		method = "POST"
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "webkit;Resolution(PAL,720P,1080P,2106P,4K)")
	req.Header.Set("X-Requested-With", "com.android.smart.terminal.ctsh.iptv")
	req.Header.Set("Accept", "*/*")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", req.URL.Scheme+"://"+req.URL.Host)
		req.Header.Set("Referer", uri)
		req.Header.Set("VIS-AJAX", "AjaxHttpRequest")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("专网请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("专网服务返回 HTTP %d", resp.StatusCode)
	}
	if sid := resp.Header.Get("X-Frame-SessionID"); sid != "" {
		c.http.Jar.SetCookies(resp.Request.URL, []*http.Cookie{{Name: "JSESSIONID", Value: sid, Path: "/"}})
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if err != nil {
		return nil, nil, err
	}
	if len(b) > 8<<20 {
		return nil, nil, errors.New("专网响应过大")
	}
	return b, resp.Request.URL, nil
}

func parseForm(b []byte, base *url.URL, selector string) (string, url.Values, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		return "", nil, err
	}
	form := doc.Find(selector).First()
	action, ok := form.Attr("action")
	if !ok || action == "" {
		return "", nil, errors.New("认证页面缺少表单，检查账号和专网连接")
	}
	u, err := url.Parse(action)
	if err != nil {
		return "", nil, err
	}
	u = base.ResolveReference(u)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", nil, errors.New("认证表单 URL 协议无效")
	}
	values := url.Values{}
	form.Find("input[name]").Each(func(_ int, n *goquery.Selection) { values.Set(n.AttrOr("name", ""), n.AttrOr("value", "")) })
	return u.String(), values, nil
}

var tokenPattern = regexp.MustCompile(`encrytoken\s*=\s*["']([^"']+)["']`)
var redirectPattern = regexp.MustCompile(`top\.document\.location\s*=\s*["']([^"']+)["']`)
var configPattern = regexp.MustCompile(`jsSetConfig\s*\(\s*['"](\w+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`)
var arrayPattern = regexp.MustCompile(`(?s)channelArray\s*=\s*\[(.*?)\]`)
var recordPattern = regexp.MustCompile(`'([^']+)'`)
var fieldPattern = regexp.MustCompile(`(\w+)="([^"]*)"`)

func parseChannels(b []byte) map[string]model.Channel {
	channels := map[string]model.Channel{}
	a := arrayPattern.FindSubmatch(b)
	if len(a) < 2 {
		return channels
	}
	for _, record := range recordPattern.FindAllSubmatch(a[1], -1) {
		fields := map[string]string{}
		for _, f := range fieldPattern.FindAllSubmatch(record[1], -1) {
			fields[string(f[1])] = string(f[2])
		}
		id := fields["UserChannelID"]
		chID := fields["ChannelID"]
		chCode := fields["ChannelCode"]
		ch := model.Channel{UserChannelID: id, ChannelID: fields["ChannelID"], ChannelURL: fields["ChannelURL"], TimeShiftURL: fields["TimeShiftURL"], TimeShift: fields["TimeShift"], ChannelFCCIP: fields["ChannelFCCIP"], ChannelFCCPort: fields["ChannelFCCPort"]}
		if id != "" {
			channels[id] = ch
		}
		if chID != "" && chID != id {
			channels[chID] = ch
		}
		if chCode != "" && chCode != id && chCode != chID {
			channels[chCode] = ch
		}
	}
	return channels
}

func (c *Client) Authenticate(ctx context.Context) error {
	query := url.Values{"Action": {"Login"}, "UserID": {c.stb.UID}, "SN": {c.stb.SN}, "Type": {"iptv4k"}, "Mode": {"MENU.SMG-4K"}, "FCCSupport": {"1"}}
	b, base, err := c.request(ctx, "http://"+c.stb.AuthHost+"/iptv3a/4kLogAuth.do?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	uri, form, err := parseForm(b, base, "form")
	if err != nil {
		return err
	}
	b, base, err = c.request(ctx, uri, form)
	if err != nil {
		return err
	}
	token := tokenPattern.FindSubmatch(b)
	if len(token) < 2 {
		return errors.New("认证页面缺少 encrytoken")
	}
	uri, form, err = parseForm(b, base, "form")
	if err != nil {
		return err
	}
	a := model.NewAuthenticator(string(token[1]), c.stb.UID, c.stb.SN, c.stb.IP, c.stb.MAC)
	a.UpdateTime = time.Now().Format("20060102150405")
	form.Set("authenticator", a.GetEncryptString())
	b, base, err = c.request(ctx, uri, form)
	if err != nil {
		return err
	}
	c.channels = parseChannels(b)
	if len(c.channels) == 0 {
		return errors.New("认证未返回频道列表，请检查机顶盒身份信息")
	}
	uri, form, err = parseForm(b, base, "form#epgform")
	if err != nil {
		return err
	}
	b, base, err = c.request(ctx, uri, form)
	if err != nil {
		return err
	}
	redirect := redirectPattern.FindSubmatch(b)
	if len(redirect) < 2 {
		return errors.New("缺少 EPG 门户跳转")
	}
	u, err := url.Parse(string(redirect[1]))
	if err != nil {
		return err
	}
	b, base, err = c.request(ctx, base.ResolveReference(u).String(), nil)
	if err != nil {
		return err
	}
	uri, form, err = parseForm(b, base, "form")
	if err != nil {
		return err
	}
	// The newer funcportalAuth form has no UserToken and needs no stbinfo.
	if token := form.Get("UserToken"); token != "" {
		plain := utils.InsertStrInUserToken(token)
		if plain == "" {
			return errors.New("无效的门户 UserToken")
		}
		r := utils.RSA{}
		r.LoadPriKey(utils.GetRSAPriKey())
		form.Set("stbinfo", strings.ToUpper(hex.EncodeToString(r.PriEncrypt([]byte(plain)))))
		form.Set("stbtype", c.stb.Type)
	}
	b, _, err = c.request(ctx, uri, form)
	if err != nil {
		return err
	}
	config := map[string]string{}
	for _, v := range configPattern.FindAllSubmatch(b, -1) {
		config[string(v[1])] = string(v[2])
	}
	if config["SessionID"] == "" || config["IpPort"] == "" {
		return errors.New("门户未返回 SessionID / IpPort")
	}
	frame := config["framecode"]
	if frame == "" {
		frame = "frame1666"
	}
	c.portal = "http://" + config["IpPort"] + "/iptvepg/" + frame
	portal, err := url.Parse(c.portal)
	if err != nil {
		return err
	}
	c.http.Jar.SetCookies(portal, []*http.Cookie{{Name: "JSESSIONID", Value: config["SessionID"], Path: "/"}})
	_, _, err = c.request(ctx, c.portal+"/portal.jsp", nil)
	return err
}

type Result struct {
	Channels []panel.Channel
	Raw      []model.Channel
	Info     []model.ChannelInfo
}

func hasLetters(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func (c *Client) Channels(ctx context.Context) (Result, error) {
	var result Result
	b, _, err := c.request(ctx, c.portal+"/function/ajax/epg7getProperties.jsp", url.Values{"action": {"getChannelCate"}})
	if err != nil {
		return result, err
	}
	var categories struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err = json.Unmarshal(b, &categories); err != nil || len(categories.Data) == 0 {
		return result, errors.New("获取频道分类失败")
	}
	seen := map[string]bool{}
	for _, category := range categories.Data {
		if category.Name == "全部" && len(categories.Data) > 1 {
			continue
		}
		b, _, err = c.request(ctx, c.portal+"/function/ajax/epg7getChannelByAjax.jsp", url.Values{"action": {"getChannelList"}, "cateID": {category.ID}, "type": {"tvod"}})
		if err != nil {
			return result, err
		}
		var response struct {
			Data []model.ChannelInfo `json:"data"`
		}
		if err = json.Unmarshal(b, &response); err != nil || response.Data == nil {
			return result, fmt.Errorf("获取分类 %s 的频道失败", category.Name)
		}
		for _, info := range response.Data {
			if seen[info.MixNo] {
				continue
			}
			raw, ok := c.channels[info.MixNo]
			if !ok {
				raw, ok = c.channels[info.ChID]
			}
			if !ok && info.Code != "" {
				raw, ok = c.channels[info.Code]
			}
			if !ok || raw.ChannelURL == "" {
				continue
			}
			seen[info.MixNo] = true
			info.IsShow = true
			info.IsPullEPG = true
			if raw.TimeShiftURL == "null" {
				raw.TimeShiftURL = ""
			}
			days := 0
			if info.IsTs == "1" && raw.TimeShiftURL != "" {
				days = 7
			}
			opID := info.ChID
			if hasLetters(raw.ChannelID) {
				opID = raw.ChannelID
			} else if hasLetters(info.ChID) {
				opID = info.ChID
			} else if hasLetters(info.Code) {
				opID = info.Code
			} else if opID == "" {
				opID = raw.ChannelID
			}
			if opID == "" {
				opID = info.Code
			}
			chID := info.MixNo
			if chID == "" || chID == "0" {
				chID = panel.InvalidChannelID
			}
			result.Channels = append(result.Channels, panel.Channel{OperatorID: opID, UnicastURL: raw.TimeShiftURL, CatchupDays: days, ID: chID, Name: info.Name, Group: category.Name, URL: raw.ChannelURL, Enabled: true, Source: "iptv"})
			result.Raw = append(result.Raw, raw)
			result.Info = append(result.Info, info)
		}
	}
	if len(result.Channels) == 0 {
		return result, errors.New("没有可用的频道，保留现有列表")
	}
	return result, nil
}

func (c *Client) Programs(ctx context.Context, info model.ChannelInfo) ([]model.EPGDetails, error) {
	now := time.Now()
	return c.ProgramsRange(ctx, info, now.AddDate(0, 0, -7), now.AddDate(0, 0, 3))
}

func (c *Client) ProgramsRange(ctx context.Context, info model.ChannelInfo, start, end time.Time) ([]model.EPGDetails, error) {
	b, _, err := c.request(ctx, c.portal+"/function/ajax/epg7getChannelByAjax.jsp", url.Values{"action": {"getChannelProg"}, "code": {info.Code}, "channelID": {info.ChID}, "startTime": {fmt.Sprint(start.UnixMilli())}, "endTime": {fmt.Sprint(end.UnixMilli())}, "offset": {"0"}, "limit": {"2000"}})
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []model.EPGDetails `json:"data"`
	}
	err = json.Unmarshal(b, &response)
	if err == nil && response.Data == nil {
		err = errors.New("运营商节目单响应无效")
	}
	return response.Data, err
}
