package spider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestHTTPPlaybackUsesProgrammeAndPreservesSignedQuery(t *testing.T) {
	now := time.Now()
	seen := map[string]int{}
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		action := r.Form.Get("action")
		seen[action]++
		if r.Form.Get("channelID") != "op1" {
			t.Error("wrong operator ID")
		}
		switch action {
		case "getChannelPlayUrl":
			fmt.Fprint(w, `{"data":{"playUrl":"","liveUrl":"http://media.test/live.m3u8?token=a%2Bb&endtime=2000000000"}}`)
		case "getPreCurNextProg":
			fmt.Fprintf(w, `{"data":{"channelID":"op1","curr":{"ID":"prog1","startTime":%d,"endTime":%d}}}`, now.Add(-time.Hour).UnixMilli(), now.Add(time.Hour).UnixMilli())
		case "getTvodPlayUrl":
			if r.Form.Get("playbillID") != "prog1" || r.Form.Get("startTime") != strconv.FormatInt(now.Add(-time.Hour).Unix(), 10) {
				t.Error(r.Form)
			}
			fmt.Fprint(w, `{"data":{"playURL":"http://media.test/a.m3u8?token=a%2Bb&endtime=2000000000"}}`)
		default:
			t.Error(action)
		}
	}))
	if server == nil {
		return
	}
	defer server.Close()
	c := &Client{http: server.Client(), portal: server.URL}
	for _, start := range []time.Time{now.Add(-time.Minute), {}} {
		value, err := c.HTTPPlayback(context.Background(), "op1", start, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(value)
		if u.Query().Get("token") != "a+b" || u.Query().Get("endtime") != "2000000000" || (!start.IsZero() && u.Query().Get("starttime") == "") {
			t.Fatal(value)
		}
	}
	if seen["getTvodPlayUrl"] != 1 || seen["getChannelPlayUrl"] != 1 {
		t.Fatal(seen)
	}
}
