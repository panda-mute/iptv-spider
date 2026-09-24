package spider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"iptv-spider/model"
	"iptv-spider/modules/panel"
	"iptv-spider/utils"
)

func newTestServer(t testing.TB, h http.Handler) *httptest.Server {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping test server in restricted sandbox: %v", err)
		return nil
	}
	ts := httptest.NewUnstartedServer(h)
	ts.Listener = l
	ts.Start()
	return ts
}

func TestPHPCompatibleAuthenticationAndCategories(t *testing.T) {
	var host string
	mux := http.NewServeMux()
	mux.HandleFunc("/iptv3a/4kLogAuth.do", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("FCCSupport") != "1" || r.URL.Query().Get("Mode") != "MENU.SMG-4K" {
			t.Error("missing STB parameters")
		}
		fmt.Fprint(w, `<form action="/login"><div><input name="a" value="b"></div></form>`)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("a") != "b" {
			t.Error("nested form input missing")
		}
		fmt.Fprint(w, `<script>var encrytoken = "f0057a9a69a842652e6f1667334f571f";</script><form action="ottauth"><input name="id" value="test"></form>`)
	})
	mux.HandleFunc("/ottauth", func(w http.ResponseWriter, r *http.Request) {
		b, err := hex.DecodeString(r.FormValue("authenticator"))
		if err != nil {
			t.Error(err)
		}
		var a model.Authenticator
		if err = json.Unmarshal(utils.NewAESForNodejs([]byte("123456")).Decrypt(b), &a); err != nil {
			t.Error(err)
		}
		if a.IP != "030,001,002,003" || a.UserID != "12345678@etv3" || a.UpdateTime == "20230301175307" {
			t.Error("incorrect authenticator", a.IP, a.UserID, a.UpdateTime)
		}
		fmt.Fprint(w, `<script>var channelArray=['ChannelID="123",UserChannelID="1",ChannelURL="igmp://239.1.1.1:5140",ChannelFCCIP="10.0.0.1",TimeShiftURL="rtsp://10.1.1.1:554/live?AuthInfo=test"','ChannelID="75",UserChannelID="75",ChannelURL="igmp://239.1.1.2:5140",TimeShiftURL="null"'];</script><form id="epgform" action="/epg-index"><input name="session" value="abc"></form>`)
	})
	mux.HandleFunc("/epg-index", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<script>top.document.location = "/iptvepg/function/load";</script>`)
	})
	mux.HandleFunc("/iptvepg/function/load", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "pre-session", Path: "/"})
		fmt.Fprint(w, `<form action="funcportalAuth"><input name="identity" value="stb"></form>`)
	})
	mux.HandleFunc("/iptvepg/function/funcportalAuth", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("JSESSIONID")
		if err != nil || cookie.Value != "pre-session" {
			t.Error("missing pre-auth cookie")
		}
		if r.FormValue("stbinfo") != "" {
			t.Error("unneeded RSA stbinfo")
		}
		fmt.Fprintf(w, `<script>jsSetConfig('SessionID','final-session');jsSetConfig("IpPort","%s");jsSetConfig('framecode','frame1666');</script>`, host)
	})
	mux.HandleFunc("/iptvepg/frame1666/portal.jsp", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("JSESSIONID")
		if err != nil || cookie.Value != "final-session" {
			t.Error("portal cookie not updated")
		}
		fmt.Fprint(w, "OK")
	})
	mux.HandleFunc("/iptvepg/frame1666/function/ajax/epg7getProperties.jsp", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"all","name":"全部"},{"id":"local","name":"地方"},{"id":"hd","name":"高清"}]}`)
	})
	categories := map[string]bool{}
	mux.HandleFunc("/iptvepg/frame1666/function/ajax/epg7getChannelByAjax.jsp", func(w http.ResponseWriter, r *http.Request) {
		categories[r.FormValue("cateID")] = true
		if r.FormValue("type") != "tvod" {
			t.Error("missing category type")
		}
		fmt.Fprint(w, `{"data":[{"mixNo":"1","ID":"123","name":"测试台","code":"one","isTs":"1"},{"mixNo":"75","ID":"75","name":"无时移","code":"two","isTs":"1"}]}`)
	})
	server := newTestServer(t, mux)
	if server == nil {
		return
	}
	defer server.Close()
	host = strings.TrimPrefix(server.URL, "http://")
	s := panel.DefaultSettings()
	s.IPTV = panel.IPTV{Enabled: true, UID: "12345678@etv3", SN: "000400000000000000000000", MAC: "00:11:22:33:44:55", IP: "30.1.2.3", Type: "B860A", AuthHost: host}
	c, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Authenticate(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := c.Channels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Channels) != 2 || result.Channels[0].Group != "地方" || result.Channels[0].URL != "igmp://239.1.1.1:5140" || !categories["local"] || !categories["hd"] || categories["all"] {
		t.Fatal(result, categories)
	}
	if result.Channels[0].CatchupDays != 7 || (result.Channels[0].OperatorID != "123" && result.Channels[0].OperatorID != "one") || result.Channels[1].UnicastURL != "" || result.Channels[1].CatchupDays != 0 {
		t.Fatal("invalid playback metadata", result.Channels)
	}
	for _, channel := range result.Channels {
		if err := channel.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMalformedPortalFailsCleanly(t *testing.T) {
	base, _ := url.Parse("http://example.com/a/b")
	if _, _, err := parseForm([]byte("<html>error</html>"), base, "form"); err == nil {
		t.Fatal("missing form accepted")
	}
	uri, form, err := parseForm([]byte(`<form action="../login"><div><input name="x" value="a&amp;b"></div></form>`), base, "form")
	if err != nil || uri != "http://example.com/login" || form.Get("x") != "a&b" {
		t.Fatal(uri, form, err)
	}
	if len(parseChannels([]byte("invalid"))) != 0 {
		t.Fatal("unexpected channel")
	}
}
