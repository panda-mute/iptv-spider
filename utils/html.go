package utils

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"iptv-spider/global"
	"net/url"
	"strings"
)

func CreateHtmlDocByBytes(uri string, resp []byte) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp))
	doc.Url, _ = url.Parse(uri)
	if err != nil {
		global.LOG.Error(err.Error())
		return nil
	}
	return doc
}

func GetFromParamByHtml(d *goquery.Document, sec ...string) (uri, method string, formMap map[string]string) {
	s := "form"
	if len(sec) == 1 {
		s = sec[0]
	}
	nodes := d.Find(s)
	// 处理Form表单
	if len(nodes.Nodes) != 1 {
		return
	}
	form := nodes.First()
	uri = form.AttrOr("action", "")
	if !strings.HasPrefix(uri, "http") {
		u := fmt.Sprintf("%s://%s", d.Url.Scheme, d.Url.Host)
		if strings.HasPrefix(uri, "/") {
			u += uri
		} else {
			i := strings.LastIndex(d.Url.Path, "/")
			if i > 0 {
				u = fmt.Sprintf("%s%s/%s", u, d.Url.Path[:i], uri)
			}
		}
		uri = u
	}
	method = form.AttrOr("method", "get")
	child := form.Children()
	formMap = map[string]string{}
	child.Each(func(_ int, s *goquery.Selection) {
		k := s.AttrOr("name", "")
		v := s.AttrOr("value", "")
		formMap[k] = v
	})
	return
}

func GetScriptsFormHtml(d *goquery.Document) []string {
	var scriptsArr []string
	scripts := d.Find("script")
	scripts.Each(func(_ int, s *goquery.Selection) {
		c := s.Nodes[0].FirstChild
		if c == nil {
			return
		}
		scriptsArr = append(scriptsArr, c.Data)
	})
	return scriptsArr
}
