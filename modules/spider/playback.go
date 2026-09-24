package spider

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"time"
)

// HTTPPlayback obtains fresh operator URLs instead of caching expiring signatures.
func (c *Client) HTTPPlayback(ctx context.Context, id string, start, end time.Time) (string, error) {
	form := url.Values{"action": {"getChannelPlayUrl"}, "channelID": {id}}
	if !start.IsZero() {
		b, _, err := c.request(ctx, c.portal+"/function/ajax/epg7getChannelByAjax.jsp", url.Values{"action": {"getPreCurNextProg"}, "channelID": {id}, "time": {strconv.FormatInt(start.UnixMilli(), 10)}})
		if err != nil {
			return "", err
		}
		var prog struct {
			Data struct {
				ChannelID string `json:"channelID"`
				Curr      *struct {
					ID        string `json:"ID"`
					StartTime int64  `json:"startTime"`
					EndTime   int64  `json:"endTime"`
				} `json:"curr"`
			} `json:"data"`
		}
		if json.Unmarshal(b, &prog) != nil || prog.Data.Curr == nil || prog.Data.Curr.ID == "" {
			return "", errors.New("该时间没有可回看的节目")
		}
		p := prog.Data.Curr
		if p.StartTime > start.UnixMilli() || p.EndTime <= start.UnixMilli() {
			return "", errors.New("运营商未返回所选时间的节目")
		}
		form = url.Values{"action": {"getTvodPlayUrl"}, "channelID": {id}, "playbillID": {p.ID}, "startTime": {strconv.FormatInt(p.StartTime/1000, 10)}, "endTime": {strconv.FormatInt(p.EndTime/1000, 10)}}
	}
	b, _, err := c.request(ctx, c.portal+"/function/ajax/epg7getChannelByAjax.jsp", form)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			PlayURL string `json:"playURL"`
			LiveURL string `json:"liveUrl"`
		} `json:"data"`
	}
	if json.Unmarshal(b, &resp) != nil {
		return "", errors.New("运营商播放地址响应无效")
	}
	value := resp.Data.PlayURL
	if start.IsZero() && resp.Data.LiveURL != "" {
		value = resp.Data.LiveURL
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return "", errors.New("运营商未返回 HTTP 播放地址")
	}
	if !start.IsZero() {
		q := u.Query()
		q.Set("starttime", strconv.FormatInt(start.Unix(), 10))
		// Preserve the operator's programme end/signature; a requested end may be
		// consumed by the player, but undocumented URL parameters are not invented.
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
