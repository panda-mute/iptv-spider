package http_client

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"iptv-spider/global"
	"net"
	"net/http"
	"strings"
)

type HttpClient struct {
	client    *resty.Client
	userAgent string
	resp      *resty.Response
}

func (c *HttpClient) Request(uri, method string, form map[string]string) *HttpClient {
	r := c.client.R()
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		r.SetQueryParams(form)
	case "POST":
		r.SetFormData(form)
	}
	global.LOG.Debug(fmt.Sprintf("%s %s", method, uri))
	var err error
	c.resp, err = r.Execute(method, uri)
	if err != nil {
		global.LOG.Error(fmt.Sprintf("%s %s", method, uri))
		global.LOG.Error(err.Error())
	}
	global.LOG.Debug(fmt.Sprintf("Resp Body: %s", string(c.resp.Body())))
	return c
}

func (c *HttpClient) GetResp() *resty.Response {
	return c.resp
}

func (c *HttpClient) GetRespBytes() []byte {
	return c.resp.Body()
}

func NewHttpClient(opts ...HttpClientOption) *HttpClient {
	c := &HttpClient{
		userAgent: "webkit;Resolution(PAL,720P,1080P,2106P,4K)",
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.client == nil {
		c.client = resty.New()
	}
	c.afterAction()
	return c
}

func (c *HttpClient) afterAction() {
	// setUserAgent
	c.client.SetHeader("User-Agent", c.userAgent)
}

func (c *HttpClient) SetCookies(cookies ...*http.Cookie) {
	if len(cookies) <= 0 {
		return
	}
	c.client.SetCookies(cookies)
}

func (c *HttpClient) Cookies() []*http.Cookie {
	return c.client.Cookies
}

type HttpClientOption func(client *HttpClient)

func WithUserAgent(ua string) HttpClientOption {
	return func(c *HttpClient) {
		c.userAgent = ua
	}
}

func WithLocalAddr(addr string) HttpClientOption {
	return func(c *HttpClient) {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			global.LOG.Warn("ResolveTCPAddr failed: " + addr)
		} else {
			global.LOG.Warn("ResolveTCPAddr success: " + addr)
		}
		c.client = resty.NewWithLocalAddr(tcpAddr)
	}
}
