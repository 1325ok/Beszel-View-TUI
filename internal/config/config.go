package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Threshold struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
}
type Config struct {
	HubURL, Email, Password, Token               string
	Refresh, Rotation, AlertDuration, AlertBlink time.Duration
	Layout, Size                                 string
	Width, Height                                int
	CPU, RAM, Disk                               Threshold
	InsecureTLS, SingleMode                      bool
}

func defaults() Config {
	return Config{Refresh: 3 * time.Second, Rotation: 5 * time.Second, AlertDuration: 20 * time.Second, AlertBlink: 750 * time.Millisecond, Layout: "auto", Size: "auto", CPU: Threshold{75, 90}, RAM: Threshold{80, 92}, Disk: Threshold{80, 92}}
}

type fileConfig struct {
	HubURL        string    `json:"hub_url"`
	Email         string    `json:"email"`
	Password      string    `json:"password"`
	Token         string    `json:"token"`
	Layout        string    `json:"layout"`
	Size          string    `json:"size"`
	Refresh       string    `json:"refresh_interval"`
	Rotation      string    `json:"rotation_interval"`
	AlertDuration string    `json:"alert_duration"`
	AlertBlink    string    `json:"alert_blink_interval"`
	CPU           Threshold `json:"cpu"`
	RAM           Threshold `json:"ram"`
	Disk          Threshold `json:"disk"`
	InsecureTLS   bool      `json:"insecure_tls"`
	SingleMode    bool      `json:"single_mode"`
}

func Load(args []string) (Config, error) {
	c := defaults()
	path := configPath(args)
	if path != "" {
		if err := applyFile(&c, path); err != nil {
			return c, err
		}
	}
	if err := applyEnv(&c); err != nil {
		return c, err
	}
	fs := flag.NewFlagSet("beszelview", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "[ Beszel View ]")
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fs.PrintDefaults()
	}
	fs.StringVar(&c.HubURL, "hub-url", c.HubURL, "Beszel Hub URL")
	fs.StringVar(&c.Email, "email", c.Email, "Beszel user email")
	fs.StringVar(&c.Password, "password", c.Password, "Beszel password")
	fs.StringVar(&c.Token, "token", c.Token, "PocketBase auth token")
	fs.DurationVar(&c.Refresh, "refresh", c.Refresh, "API refresh interval")
	fs.DurationVar(&c.Rotation, "rotation", c.Rotation, "screen rotation interval")
	fs.DurationVar(&c.AlertDuration, "alert-duration", c.AlertDuration, "alert screen duration")
	fs.DurationVar(&c.AlertBlink, "alert-blink", c.AlertBlink, "alert blink interval")
	fs.StringVar(&c.Layout, "layout", c.Layout, "small, large, or auto")
	fs.StringVar(&c.Size, "size", c.Size, "render size (auto or WIDTHxHEIGHT, e.g. 80x24)")
	fs.BoolVar(&c.InsecureTLS, "insecure-tls", c.InsecureTLS, "skip TLS verification")
	fs.BoolVar(&c.SingleMode, "single", c.SingleMode, "show all devices on one page without rotation")
	fs.Float64Var(&c.CPU.Warning, "cpu-warning", c.CPU.Warning, "CPU warning percent")
	fs.Float64Var(&c.CPU.Critical, "cpu-critical", c.CPU.Critical, "CPU critical percent")
	fs.Float64Var(&c.RAM.Warning, "ram-warning", c.RAM.Warning, "RAM warning percent")
	fs.Float64Var(&c.RAM.Critical, "ram-critical", c.RAM.Critical, "RAM critical percent")
	fs.Float64Var(&c.Disk.Warning, "disk-warning", c.Disk.Warning, "disk warning percent")
	fs.Float64Var(&c.Disk.Critical, "disk-critical", c.Disk.Critical, "disk critical percent")
	clean := removeConfigArgs(args)
	if err := fs.Parse(clean); err != nil {
		return c, err
	}
	c.HubURL = strings.TrimRight(c.HubURL, "/")
	if c.HubURL == "" {
		return c, errors.New("hub URL is required (--hub-url or BESZEL_HUB_URL)")
	}
	if c.Token == "" && (c.Email == "" || c.Password == "") {
		return c, errors.New("provide --token or both --email and --password")
	}
	if c.Layout != "auto" && c.Layout != "small" && c.Layout != "large" {
		return c, fmt.Errorf("invalid layout %q", c.Layout)
	}
	if err := parseSize(&c); err != nil {
		return c, err
	}
	if c.Layout == "large" && c.Width > 0 && (c.Width < 80 || c.Height < 22) {
		return c, errors.New("large layout requires a fixed size of at least 80x22")
	}
	if c.Refresh <= 0 || c.Rotation <= 0 || c.AlertDuration <= 0 || c.AlertBlink <= 0 {
		return c, errors.New("durations must be positive")
	}
	for _, item := range []struct {
		name      string
		threshold Threshold
	}{{"CPU", c.CPU}, {"RAM", c.RAM}, {"disk", c.Disk}} {
		if item.threshold.Warning <= 0 || item.threshold.Critical <= 0 || item.threshold.Warning >= item.threshold.Critical || item.threshold.Critical > 100 {
			return c, fmt.Errorf("%s thresholds must satisfy 0 < warning < critical <= 100", item.name)
		}
	}
	return c, nil
}

func parseSize(c *Config) error {
	c.Size = strings.ToLower(strings.TrimSpace(c.Size))
	if c.Size == "" || c.Size == "auto" {
		c.Size = "auto"
		c.Width, c.Height = 0, 0
		return nil
	}
	parts := strings.Split(c.Size, "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid size %q: use auto or WIDTHxHEIGHT (for example 80x24)", c.Size)
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w < 38 || h < 13 {
		return fmt.Errorf("invalid size %q: minimum fixed size is 38x13", c.Size)
	}
	c.Width, c.Height = w, h
	c.Size = fmt.Sprintf("%dx%d", w, h)
	return nil
}

func configPath(args []string) string {
	for i, arg := range args {
		if (arg == "--config" || arg == "-config") && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
			return strings.SplitN(arg, "=", 2)[1]
		}
	}
	return ""
}

func removeConfigArgs(a []string) []string {
	out := []string{}
	for i := 0; i < len(a); i++ {
		if a[i] == "--config" || a[i] == "-config" {
			i++
			continue
		}
		if strings.HasPrefix(a[i], "--config=") || strings.HasPrefix(a[i], "-config=") {
			continue
		}
		out = append(out, a[i])
	}
	return out
}
func applyFile(c *Config, p string) error {
	b, e := os.ReadFile(p)
	if e != nil {
		return e
	}
	var f fileConfig
	if e = json.Unmarshal(b, &f); e != nil {
		return e
	}
	if f.HubURL != "" {
		c.HubURL = f.HubURL
	}
	if f.Email != "" {
		c.Email = f.Email
	}
	if f.Password != "" {
		c.Password = f.Password
	}
	if f.Token != "" {
		c.Token = f.Token
	}
	if f.Layout != "" {
		c.Layout = f.Layout
	}
	if f.Size != "" {
		c.Size = f.Size
	}
	for s, d := range map[string]*time.Duration{f.Refresh: &c.Refresh, f.Rotation: &c.Rotation, f.AlertDuration: &c.AlertDuration, f.AlertBlink: &c.AlertBlink} {
		if s != "" {
			v, e := time.ParseDuration(s)
			if e != nil {
				return e
			}
			*d = v
		}
	}
	if f.CPU.Warning != 0 || f.CPU.Critical != 0 {
		c.CPU = f.CPU
	}
	if f.RAM.Warning != 0 || f.RAM.Critical != 0 {
		c.RAM = f.RAM
	}
	if f.Disk.Warning != 0 || f.Disk.Critical != 0 {
		c.Disk = f.Disk
	}
	c.InsecureTLS = f.InsecureTLS
	c.SingleMode = f.SingleMode
	return nil
}
func applyEnv(c *Config) error {
	str := map[string]*string{"BESZEL_HUB_URL": &c.HubURL, "BESZEL_EMAIL": &c.Email, "BESZEL_PASSWORD": &c.Password, "BESZEL_TOKEN": &c.Token, "BESZEL_LAYOUT": &c.Layout, "BESZEL_SIZE": &c.Size}
	for k, p := range str {
		if v := os.Getenv(k); v != "" {
			*p = v
		}
	}
	dur := map[string]*time.Duration{"BESZEL_REFRESH": &c.Refresh, "BESZEL_ROTATION": &c.Rotation, "BESZEL_ALERT_DURATION": &c.AlertDuration, "BESZEL_ALERT_BLINK": &c.AlertBlink}
	for k, p := range dur {
		if v := os.Getenv(k); v != "" {
			x, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			*p = x
		}
	}
	if v := os.Getenv("BESZEL_INSECURE_TLS"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("BESZEL_INSECURE_TLS: %w", err)
		}
		c.InsecureTLS = parsed
	}
	if v := os.Getenv("BESZEL_SINGLE"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("BESZEL_SINGLE: %w", err)
		}
		c.SingleMode = parsed
	}
	return nil
}
