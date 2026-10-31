package screen

import (
	"github.com/1325ok/Beszel-View-TUI/internal/model"
	"time"
)

type Kind int

const (
	Overview Kind = iota
	Detail
	Alert
)

type View struct {
	Kind    Kind
	Index   int
	Started time.Time
}
type Controller struct {
	view            View
	rotation, alert time.Duration
}

func New(now time.Time, r, a time.Duration) *Controller {
	return &Controller{view: View{Kind: Overview, Started: now}, rotation: r, alert: a}
}
func (c *Controller) Current(now time.Time, s model.Snapshot) View {
	duration := c.rotation
	if c.view.Kind == Alert {
		duration = c.alert
	}
	if now.Sub(c.view.Started) < duration {
		return c.safe(s)
	}
	c.advance(now, s)
	return c.safe(s)
}
func (c *Controller) safe(s model.Snapshot) View {
	if c.view.Kind == Detail && c.view.Index >= len(s.Devices) {
		c.view = View{Kind: Overview, Started: c.view.Started}
	}
	return c.view
}
func (c *Controller) advance(now time.Time, s model.Snapshot) {
	switch c.view.Kind {
	case Overview:
		if len(s.Devices) > 0 {
			c.view = View{Kind: Detail, Index: 0, Started: now}
		} else {
			c.view.Started = now
		}
	case Detail:
		if c.view.Index+1 < len(s.Devices) {
			c.view = View{Kind: Detail, Index: c.view.Index + 1, Started: now}
		} else if hasAlerts(s) {
			c.view = View{Kind: Alert, Started: now}
		} else {
			c.view = View{Kind: Overview, Started: now}
		}
	case Alert:
		c.view = View{Kind: Overview, Started: now}
	}
}
func hasAlerts(s model.Snapshot) bool {
	if !s.HubOnline {
		return true
	}
	for _, d := range s.Devices {
		if d.Status == model.Down || d.Status == model.Warning {
			return true
		}
	}
	return false
}
