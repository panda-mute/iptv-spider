package auth

import (
	"encoding/hex"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
	"iptv-spider/global"
	"iptv-spider/utils"
	"net/url"
	"strings"
)

func (c *Client) epgIndex(doc *goquery.Document) *goquery.Document {
	uri, method, formMap := utils.GetFromParamByHtml(doc, "form#epgform")
	// 保存 Token
	c.UserToken = formMap["UserToken"]
	resp := c.httpClient.Request(uri, method, formMap)
	return utils.CreateHtmlDocByBytes(uri, resp.GetRespBytes())
}

func (c *Client) epgLoadBalance(doc *goquery.Document) *goquery.Document {
	var uri string
	scs := utils.GetScriptsFormHtml(doc)
	for _, sc := range scs {
		if !strings.Contains(sc, "top.document.location") {
			continue
		}
		sArr := strings.Split(sc, "\n")
		for _, s := range sArr {
			if !strings.Contains(s, "top.document.location") {
				continue
			}
			index := strings.Index(s, "'")
			last := strings.LastIndex(s, "'")
			if index < 0 || last < 0 {
				global.LOG.Error("No top.document.location")
				return nil
			}
			uri = s[index+1 : last]
		}
	}
	u, err := url.Parse(uri)
	if err != nil {
		global.LOG.Error(err.Error())
		return nil
	}
	c.EPGLoginHost = u.Host
	resp := c.httpClient.Request(uri, "GET", nil)
	return utils.CreateHtmlDocByBytes(uri, resp.GetRespBytes())
}

func (c *Client) epgPortalAuth(doc *goquery.Document) (*goquery.Document, error) {
	uri, method, formMap := utils.GetFromParamByHtml(doc, "form")

	r := utils.RSA{}
	r.LoadPriKey(utils.GetRSAPriKey())

	token := formMap["UserToken"]
	plainData := utils.InsertStrInUserToken(token)
	if plainData == "" {
		global.LOG.Error("InsertStrInUserToken: plainData is empty, Token: " + token)
		return nil, fmt.Errorf("token is empty")
	}
	stbInfo := hex.EncodeToString(r.PriEncrypt([]byte(plainData)))
	formMap["stbtype"] = c.stbType
	formMap["stbinfo"] = strings.ToUpper(stbInfo)

	resp := c.httpClient.Request(uri, method, formMap)
	respDoc := utils.CreateHtmlDocByBytes(uri, resp.GetRespBytes())

	//jsSetConfig('SessionID','777ABD76B5E1C3554016FBA0EFDA2F66');
	//jsSetConfig('framecode','frame1413');
	//jsSetConfig('IpPort','218.83.165.40:8084');
	//jsSetConfig('EPGDefaultChannelNo','0');

	infoMap := c.parseEpgAuthInfo(respDoc)
	c.JSESSIONID = infoMap["SessionID"]
	c.EPGHostUrl = fmt.Sprintf("http://%s/iptvepg/%s", infoMap["IpPort"], infoMap["framecode"])

	return respDoc, nil
}

func (c *Client) parseEpgAuthInfo(doc *goquery.Document) map[string]string {
	cache := map[string]string{}
	c.jsVM.Reset()
	c.jsVM.RunScriptForHtml(doc)
	c.jsVM.Set("jsSetConfig", func(call otto.FunctionCall) otto.Value {
		k := call.Argument(0).String()
		v := call.Argument(1).String()
		cache[k] = v
		return otto.Value{}
	})
	scs := utils.GetScriptsFormHtml(doc)
	for _, sc := range scs {
		sArr := strings.Split(sc, "\n")
		for _, s := range sArr {
			if !strings.Contains(s, "jsSetConfig") {
				continue
			}
			c.jsVM.RunScript(s)
		}
	}
	return cache
}

func (c *Client) epgGetPortal() {
	// http://218.83.165.40:8084/iptvepg/frame1413/portal.jsp
	p := "portal.jsp"
	uri := fmt.Sprintf("%s/%s", c.EPGHostUrl, p)
	c.httpClient.Request(uri, "GET", nil)
}
