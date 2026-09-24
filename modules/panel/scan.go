package panel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"
)

func multicastHost(source string) string {
	u, err := url.Parse(source)
	if err != nil {
		return ""
	}
	return u.Host
}

// Transport stream sync bytes must repeat. An HTTP 200 or HTML page is not a channel.
func isTransportStream(b []byte) bool {
	for _, stride := range []int{188, 192, 204} {
		for i := 0; i < stride && i+2*stride < len(b); i++ {
			if b[i] == 0x47 && b[i+stride] == 0x47 && b[i+2*stride] == 0x47 {
				return true
			}
		}
	}
	return false
}

func probe(ctx context.Context, client *http.Client, uri string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("转发服务返回 HTTP %d", resp.StatusCode)
	}
	buf := make([]byte, 0, 65536)
	chunk := make([]byte, 4096)
	for len(buf) < 65536 {
		n, e := resp.Body.Read(chunk)
		buf = append(buf, chunk[:n]...)
		if isTransportStream(buf) {
			res, _ := ProbeResolutionFromMPEGTS(ctx, io.MultiReader(bytes.NewReader(buf), io.LimitReader(resp.Body, 8<<20)))
			return res, nil
		}
		if e != nil {
			if e != io.EOF {
				return "", e
			}
			break
		}
	}
	return "", errors.New("未检测到 MPEG-TS 数据")
}

func (s *Service) StartScan() error {
	settings, _ := s.Store.Snapshot()
	total, err := settings.Scan.Bounds()
	if err != nil {
		return err
	}
	if settings.Forward.Address == "" {
		return errors.New("请先配置组播转发服务")
	}
	known, err := s.Channels()
	if err != nil {
		return err
	}
	knownURLs := map[string]bool{}
	for _, c := range known {
		knownURLs[multicastHost(c.URL)] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan.State == "running" || s.scan.State == "stopping" {
		return errors.New("扫描正在运行")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.scan = Job{State: "running", Total: total, StartedAt: time.Now()}
	s.scanDone = make(chan struct{})
	go s.runScan(ctx, settings, knownURLs, s.scanDone)
	return nil
}

func (s *Service) StopScan() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil && s.scan.State == "running" {
		s.scan.State = "stopping"
		s.cancel()
	}
}

func (s *Service) runScan(ctx context.Context, settings Settings, known map[string]bool, done chan struct{}) {
	defer close(done)
	transport := &http.Transport{MaxIdleConnsPerHost: settings.Scan.Workers, ResponseHeaderTimeout: time.Duration(settings.Scan.Timeout) * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < settings.Scan.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				if ctx.Err() != nil {
					return
				}
				source := "igmp://" + target
				pc, cancel := context.WithTimeout(ctx, time.Duration(settings.Scan.Timeout)*time.Second)
				res, err := probe(pc, client, PlaybackURL(settings.Forward, source))
				cancel()
				if ctx.Err() != nil {
					return
				}
				var saveErr error
				if err == nil {
					if !known[target] {
						saveErr = s.Store.Discover(Channel{
							Key:        "scan-" + target,
							ID:         InvalidChannelID,
							Name:       "未知频道 " + target,
							Group:      "待识别",
							URL:        source,
							Enabled:    true,
							Source:     "scan",
							Resolution: res,
						})
					} else if res != "" {
						_ = s.Store.UpdateResolution("scan-"+target, res)
					}
				}
				s.mu.Lock()
				s.scan.Done++
				if err == nil {
					s.scan.Found++
				} else {
					s.scan.Error = err.Error()
				}
				if saveErr != nil {
					s.scan.Error = "保存扫描结果失败: " + saveErr.Error()
					s.scan.State = "failed"
					s.cancel()
				}
				s.mu.Unlock()
			}
		}()
	}
	a, _ := netip.ParseAddr(settings.Scan.StartIP)
	b, _ := netip.ParseAddr(settings.Scan.EndIP)
dispatch:
	for ip := a; ip.Compare(b) <= 0; ip = ip.Next() {
		for p := settings.Scan.StartPort; p <= settings.Scan.EndPort; p++ {
			select {
			case <-ctx.Done():
				break dispatch
			case jobs <- fmt.Sprintf("%s:%d", ip, p):
			}
		}
	}
	close(jobs)
	wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan.State != "failed" {
		if ctx.Err() != nil {
			s.scan.State = "cancelled"
		} else {
			s.scan.State = "completed"
		}
	}
	s.cancel()
	s.cancel = nil
}
