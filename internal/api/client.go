package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1325ok/Beszel-View-TUI/internal/config"
	"github.com/1325ok/Beszel-View-TUI/internal/model"
)

type Client struct {
	cfg   config.Config
	http  *http.Client
	mu    sync.Mutex
	token string
}

func New(c config.Config) *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.InsecureTLS}
	return &Client{cfg: c, http: &http.Client{Timeout: 10 * time.Second, Transport: tr}, token: c.Token}
}

type page struct {
	Items            []json.RawMessage `json:"items"`
	Page, TotalPages int
}

func (c *Client) Fetch(ctx context.Context) ([]model.Device, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	var all []model.Device
	for p := 1; ; p++ {
		u := fmt.Sprintf("%s/api/collections/systems/records?page=%d&perPage=200&sort=name", c.cfg.HubURL, p)
		var pg page
		if err := c.get(ctx, u, &pg); err != nil {
			return nil, err
		}
		for _, raw := range pg.Items {
			d, err := decode(raw)
			if err != nil {
				return nil, fmt.Errorf("decode system record: %w", err)
			}
			all = append(all, d)
		}
		if pg.TotalPages <= p || pg.TotalPages == 0 {
			break
		}
	}
	return all, nil
}
func (c *Client) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"identity": c.cfg.Email, "password": c.cfg.Password})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.cfg.HubURL+"/api/collections/users/auth-with-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return httpError(res)
	}
	var v struct {
		Token string `json:"token"`
	}
	if e = json.NewDecoder(res.Body).Decode(&v); e != nil {
		return e
	}
	if v.Token == "" {
		return fmt.Errorf("login returned no token")
	}
	c.token = v.Token
	return nil
}
func (c *Client) get(ctx context.Context, u string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	c.mu.Lock()
	t := c.token
	c.mu.Unlock()
	req.Header.Set("Authorization", t)
	res, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode == 401 && c.cfg.Token == "" {
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
	}
	if res.StatusCode/100 != 2 {
		return httpError(res)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
func httpError(r *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
	return fmt.Errorf("hub returned %s: %s", r.Status, strings.TrimSpace(string(b)))
}
func decode(raw []byte) (model.Device, error) {
	var x map[string]any
	if e := json.Unmarshal(raw, &x); e != nil {
		return model.Device{}, e
	}
	info, _ := x["info"].(map[string]any)
	d := model.Device{ID: text(x, "id"), Name: text(x, "name"), Description: text(x, "description"), Updated: date(text(x, "updated"))}
	s := strings.ToLower(text(x, "status"))
	if s == "up" || s == "online" {
		d.Status = model.Online
	} else if s == "warning" || s == "warn" {
		d.Status = model.Warning
	} else if s == "down" || s == "offline" {
		d.Status = model.Down
	} else {
		d.Status = model.Unknown
	}
	d.CPU = num(info, "cpu")
	d.RAM = num(info, "mp", "memory_percent")
	d.Disk = num(info, "dp", "disk_percent")
	d.Load = num(info, "l", "load")
	d.Temp = num(info, "t", "temp")
	d.RX = num(info, "rb", "rx", "network_rx")
	d.TX = num(info, "sb", "tx", "network_tx")
	d.MemoryUsed = num(info, "mu")
	d.MemoryTotal = num(info, "m")
	d.DiskUsed = num(info, "du")
	d.DiskTotal = num(info, "d")
	d.Uptime = time.Duration(num(info, "u", "uptime")) * time.Second
	return d, nil
}
func text(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
func num(m map[string]any, ks ...string) float64 {
	for _, k := range ks {
		switch v := m[k].(type) {
		case float64:
			return v
		case json.Number:
			f, _ := v.Float64()
			return f
		case string:
			f, _ := strconv.ParseFloat(v, 64)
			return f
		}
	}
	return 0
}
func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05.000Z", s)
	if t.IsZero() {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}
